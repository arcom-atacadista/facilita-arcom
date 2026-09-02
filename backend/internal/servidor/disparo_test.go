package servidor_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

// enfileirar pede o disparo de uma dívida e devolve o corpo da resposta.
func enfileirar(t *testing.T, h http.Handler, cookie *http.Cookie, divida any) (int, map[string]any) {
	t.Helper()

	rec := chamar(t, h, http.MethodPost, "/api/v1/disparos", map[string]any{"dividaId": divida}, cookie)

	var corpo map[string]any
	if rec.Body.Len() > 0 {
		_ = json.Unmarshal(rec.Body.Bytes(), &corpo)
	}
	return rec.Code, corpo
}

// Este é o teste central da camada de disparo enquanto não há canal de envio:
// a mensagem entra na fila, NÃO é marcada como enviada, e o operador é avisado.
// Marcar como enviada sem enviar faria a operação acreditar que falou com o
// cliente — o pior resultado possível.
func TestSemCanalAMensagemFicaNaFilaENaoEhMarcadaComoEnviada(t *testing.T) {
	h, gdb := montarServidor(t)
	criarGerencia(t, h)
	cookie := logar(t, h, emailGerencia, senhaGerencia)

	divida := semearDivida(t, gdb, "11111111111", "CT-1", "", 1000, 40)

	status, corpo := enfileirar(t, h, cookie, divida)
	if status != http.StatusCreated {
		t.Fatalf("enfileirar: status %d (%v)", status, corpo)
	}

	aviso, _ := corpo["aviso"].(string)
	if aviso == "" {
		t.Error("resposta não avisou que não existe canal de envio configurado")
	}

	var statusDisparo, canal string
	var enviadoEm *time.Time
	if err := gdb.Raw(
		`SELECT status, canal, enviado_em FROM disparos WHERE divida_id = ?`, divida,
	).Row().Scan(&statusDisparo, &canal, &enviadoEm); err != nil {
		t.Fatalf("consultar disparo: %v", err)
	}

	if statusDisparo != "na_fila" {
		t.Errorf("status = %q, quer na_fila — nada foi enviado", statusDisparo)
	}
	if enviadoEm != nil {
		t.Errorf("enviado_em = %v, quer nulo", enviadoEm)
	}
	// "whatsapp" aqui sugeriria que a mensagem saiu pelo WhatsApp.
	if canal != "pendente" {
		t.Errorf("canal = %q, quer pendente", canal)
	}
}

func TestResumoDizQueNaoHaCanalConfigurado(t *testing.T) {
	h, _ := montarServidor(t)
	criarGerencia(t, h)
	cookie := logar(t, h, emailGerencia, senhaGerencia)

	rec := chamar(t, h, http.MethodGet, "/api/v1/disparos/resumo", nil, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("resumo: status %d (%s)", rec.Code, rec.Body.String())
	}

	var resumo struct {
		Canal string `json:"canal"`
		Ativo bool   `json:"ativo"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resumo); err != nil {
		t.Fatalf("resumo inesperado: %s", rec.Body.String())
	}
	if resumo.Ativo {
		t.Error("resumo diz que o canal está ativo, mas não existe canal")
	}
	if resumo.Canal != "pendente" {
		t.Errorf("canal = %q, quer pendente", resumo.Canal)
	}
}

func TestTravaAntiSpamRecusaSegundoDisparo(t *testing.T) {
	h, gdb := montarServidor(t)
	criarGerencia(t, h)
	cookie := logar(t, h, emailGerencia, senhaGerencia)

	divida := semearDivida(t, gdb, "11111111111", "CT-1", "", 1000, 40)

	if status, corpo := enfileirar(t, h, cookie, divida); status != http.StatusCreated {
		t.Fatalf("primeiro disparo: status %d (%v)", status, corpo)
	}
	// Dois operadores pedindo, ou a régua automática coincidindo com um
	// pedido manual, não podem gerar duas cobranças no mesmo dia.
	status, corpo := enfileirar(t, h, cookie, divida)
	if status != http.StatusConflict {
		t.Fatalf("segundo disparo: status %d, quer 409 (%v)", status, corpo)
	}

	var quantos int64
	if err := gdb.Raw(`SELECT count(*) FROM disparos WHERE divida_id = ?`, divida).Scan(&quantos).Error; err != nil {
		t.Fatalf("contar: %v", err)
	}
	if quantos != 1 {
		t.Errorf("gravou %d disparos, quer 1", quantos)
	}
}

func TestDisparoExigeTelefoneValido(t *testing.T) {
	h, gdb := montarServidor(t)
	criarGerencia(t, h)
	cookie := logar(t, h, emailGerencia, senhaGerencia)

	divida := semearDivida(t, gdb, "11111111111", "CT-1", "", 1000, 40)
	if err := gdb.Exec(`UPDATE clientes SET telefone = NULL WHERE documento = ?`, "11111111111").Error; err != nil {
		t.Fatalf("limpar telefone: %v", err)
	}

	status, corpo := enfileirar(t, h, cookie, divida)
	if status != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, quer 422 (%v)", status, corpo)
	}
}

func TestDisparoRecusaDividaForaDaRegua(t *testing.T) {
	h, gdb := montarServidor(t)
	criarGerencia(t, h)
	cookie := logar(t, h, emailGerencia, senhaGerencia)

	// 120 dias: fora da régua de 3 a 90, é caso de jurídico.
	fora := semearDivida(t, gdb, "11111111111", "CT-VELHA", "", 1000, 120)
	if status, corpo := enfileirar(t, h, cookie, fora); status != http.StatusConflict {
		t.Errorf("120 dias: status = %d, quer 409 (%v)", status, corpo)
	}

	// 1 dia: ainda não entrou na régua.
	nova := semearDivida(t, gdb, "22222222222", "CT-NOVA", "", 1000, 1)
	if status, corpo := enfileirar(t, h, cookie, nova); status != http.StatusConflict {
		t.Errorf("1 dia: status = %d, quer 409 (%v)", status, corpo)
	}
}

func TestMensagemEnfileiradaLevaOLinkEOsDadosDaFaixa(t *testing.T) {
	h, gdb := montarServidor(t)
	criarGerencia(t, h)
	cookie := logar(t, h, emailGerencia, senhaGerencia)

	divida := semearDivida(t, gdb, "11111111111", "CT-1", "", 1250.90, 40)
	if status, corpo := enfileirar(t, h, cookie, divida); status != http.StatusCreated {
		t.Fatalf("enfileirar: status %d (%v)", status, corpo)
	}

	var mensagem string
	if err := gdb.Raw(`SELECT mensagem FROM disparos WHERE divida_id = ?`, divida).Scan(&mensagem).Error; err != nil {
		t.Fatalf("consultar mensagem: %v", err)
	}

	for _, esperado := range []string{
		"CT-1",        // contrato
		"40 dias",     // dias de atraso da faixa 31-60
		"R$ 1.250,90", // valor formatado em real
		"https://facilita.arcom.com.br/negociar/", // link público
	} {
		if !strings.Contains(mensagem, esperado) {
			t.Errorf("mensagem não contém %q:\n%s", esperado, mensagem)
		}
	}
	if strings.Contains(mensagem, "{") {
		t.Errorf("sobrou marcador na mensagem: %s", mensagem)
	}
}

// O link que vai na mensagem tem que funcionar de verdade — é o caminho todo:
// disparo monta a mensagem, o cliente clica e negocia.
func TestOLinkDaMensagemAbreANegociacao(t *testing.T) {
	h, gdb := montarServidor(t)
	criarGerencia(t, h)
	cookie := logar(t, h, emailGerencia, senhaGerencia)

	divida := semearDivida(t, gdb, "11111111111", "CT-1", "", 1000, 40)
	if status, corpo := enfileirar(t, h, cookie, divida); status != http.StatusCreated {
		t.Fatalf("enfileirar: status %d (%v)", status, corpo)
	}

	var mensagem string
	if err := gdb.Raw(`SELECT mensagem FROM disparos WHERE divida_id = ?`, divida).Scan(&mensagem).Error; err != nil {
		t.Fatalf("consultar mensagem: %v", err)
	}

	i := strings.Index(mensagem, "/negociar/")
	if i < 0 {
		t.Fatalf("mensagem sem link: %s", mensagem)
	}
	token := strings.Fields(mensagem[i+len("/negociar/"):])[0]

	// Sem cookie: é o cliente abrindo do celular.
	rec := chamar(t, h, http.MethodGet, "/api/v1/negociar/"+token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("abrir o link da mensagem: status %d (%s)", rec.Code, rec.Body.String())
	}
}

func TestAnalistaNaoDisparaParaCarteiraDeOutro(t *testing.T) {
	h, gdb := montarServidor(t)
	criarGerencia(t, h)
	cookieGerencia := logar(t, h, emailGerencia, senhaGerencia)

	dividaDoBruno := semearDivida(t, gdb, "33333333333", "CT-BRUNO", "BRUNO", 1000, 40)
	cookieAna := criarOperador(t, h, cookieGerencia, "ana@arcom.com.br", "analista", "ANA")

	if status, corpo := enfileirar(t, h, cookieAna, dividaDoBruno); status != http.StatusNotFound {
		t.Fatalf("status = %d, quer 404 (%v)", status, corpo)
	}

	var quantos int64
	if err := gdb.Raw(`SELECT count(*) FROM disparos`).Scan(&quantos).Error; err != nil {
		t.Fatalf("contar: %v", err)
	}
	if quantos != 0 {
		t.Errorf("gravou %d disparo(s) fora da carteira", quantos)
	}
}

func TestHistoricoDeDisparoMascaraOTelefone(t *testing.T) {
	h, gdb := montarServidor(t)
	criarGerencia(t, h)
	cookie := logar(t, h, emailGerencia, senhaGerencia)

	divida := semearDivida(t, gdb, "11111111111", "CT-1", "", 1000, 40)
	if status, _ := enfileirar(t, h, cookie, divida); status != http.StatusCreated {
		t.Fatal("enfileirar falhou")
	}

	rec := chamar(t, h, http.MethodGet, "/api/v1/disparos", nil, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("listar: status %d (%s)", rec.Code, rec.Body.String())
	}

	// O número completo não precisa circular na tela de histórico.
	if strings.Contains(rec.Body.String(), "5531988887777") {
		t.Errorf("telefone completo apareceu no histórico: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "7777") {
		t.Errorf("histórico deveria mostrar os últimos dígitos: %s", rec.Body.String())
	}
}
