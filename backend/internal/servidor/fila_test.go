package servidor_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"testing"

	"gorm.io/gorm"

	"facilitaarcom/internal/cobranca"
	"facilitaarcom/internal/disparo"
)

// canalFalso substitui a Meta nos testes da fila. É o mesmo ponto de extensão
// que o canal real usa, então o que se prova aqui vale para produção.
type canalFalso struct {
	mu       sync.Mutex
	enviadas []disparo.Mensagem
	erro     error
}

func (c *canalFalso) Nome() string { return "falso" }

func (c *canalFalso) Enviar(_ context.Context, m disparo.Mensagem) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.erro != nil {
		return "", c.erro
	}
	c.enviadas = append(c.enviadas, m)
	return "wamid.FALSO", nil
}

func (c *canalFalso) recebidas() []disparo.Mensagem {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]disparo.Mensagem(nil), c.enviadas...)
}

// filaComCanal monta um serviço de disparo ligado ao canal dado, sobre o mesmo
// banco de teste que o servidor usa.
func filaComCanal(t *testing.T, canal disparo.Canal) (http.Handler, *disparo.Service, *http.Cookie, *gorm.DB) {
	t.Helper()

	// Um montarServidor só por teste: ele toma o lock do banco de teste, e
	// uma segunda chamada no mesmo teste ficaria esperando a si mesma.
	h, gdb := montarServidor(t)
	criarGerencia(t, h)
	cookie := logar(t, h, emailGerencia, senhaGerencia)

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	repoCobranca := cobranca.NovoRepo(gdb)
	svcCobranca := cobranca.NovoService(repoCobranca, "https://facilita.arcom.com.br")
	fila := disparo.NovoService(disparo.NovoRepo(gdb), repoCobranca, svcCobranca, canal, log)

	return h, fila, cookie, gdb
}

// O caminho completo: o operador pede o disparo, a fila entrega pelo canal e
// o registro passa a dizer enviado — com o wamid guardado para conciliar.
func TestFilaEntregaPeloCanalEGuardaAReferencia(t *testing.T) {
	canal := &canalFalso{}
	h, fila, cookie, gdb := filaComCanal(t, canal)

	divida := semearDivida(t, gdb, "11111111111", "CT-1", "", 1250.90, 40)
	if status, corpo := enfileirar(t, h, cookie, divida); status != http.StatusCreated {
		t.Fatalf("enfileirar: status %d (%v)", status, corpo)
	}

	enviados, err := fila.ProcessarFila(context.Background())
	if err != nil {
		t.Fatalf("ProcessarFila: %v", err)
	}
	if enviados != 1 {
		t.Fatalf("entregou %d mensagens, quer 1", enviados)
	}

	recebidas := canal.recebidas()
	if len(recebidas) != 1 {
		t.Fatalf("o canal recebeu %d mensagens", len(recebidas))
	}
	m := recebidas[0]

	// O que chega no canal é o template aprovado com os parâmetros em ordem —
	// não o texto montado, que a Meta não aceitaria fora da janela.
	if m.Template != "arcom_cobranca_atraso_medio" {
		t.Errorf("template = %q, quer o aprovado para a faixa 31-60", m.Template)
	}
	if m.Idioma != "pt_BR" {
		t.Errorf("idioma = %q", m.Idioma)
	}
	if len(m.Parametros) != 5 {
		t.Fatalf("parâmetros = %v, quer 5 na ordem da campanha", m.Parametros)
	}
	if m.Parametros[1] != "CT-1" || m.Parametros[2] != "R$ 1.250,90" || m.Parametros[3] != "40" {
		t.Errorf("parâmetros fora da ordem esperada: %v", m.Parametros)
	}
	if m.Telefone != "5531988887777" {
		t.Errorf("telefone = %q, quer normalizado com DDI", m.Telefone)
	}

	var status, referencia string
	var enviadoEm *string
	if err := gdb.Raw(
		`SELECT status, coalesce(referencia_externa,''), enviado_em::text FROM disparos WHERE divida_id = ?`, divida,
	).Row().Scan(&status, &referencia, &enviadoEm); err != nil {
		t.Fatalf("consultar disparo: %v", err)
	}
	if status != "enviado" {
		t.Errorf("status = %q, quer enviado", status)
	}
	if referencia != "wamid.FALSO" {
		t.Errorf("referência = %q, quer o id devolvido pelo canal", referencia)
	}
	if enviadoEm == nil {
		t.Error("enviado_em ficou nulo num disparo entregue")
	}
}

// Recusa permanente não pode voltar para a fila: insistir num template
// reprovado três vezes só gasta tempo e enche o log.
func TestRecusaPermanenteNaoVoltaParaAFila(t *testing.T) {
	canal := &canalFalso{erro: &disparo.ErroMeta{
		Status: 400, Codigo: 132001, Mensagem: "Template not found", Permanente: true,
	}}
	h, fila, cookie, gdb := filaComCanal(t, canal)

	divida := semearDivida(t, gdb, "11111111111", "CT-1", "", 1000, 40)
	if status, _ := enfileirar(t, h, cookie, divida); status != http.StatusCreated {
		t.Fatal("enfileirar falhou")
	}

	if _, err := fila.ProcessarFila(context.Background()); err != nil {
		t.Fatalf("ProcessarFila: %v", err)
	}

	var status, detalhe string
	if err := gdb.Raw(
		`SELECT status, coalesce(erro_detalhe,'') FROM disparos WHERE divida_id = ?`, divida,
	).Row().Scan(&status, &detalhe); err != nil {
		t.Fatalf("consultar: %v", err)
	}
	if status != "erro" {
		t.Errorf("status = %q, quer erro (sem nova tentativa)", status)
	}
	// A mensagem que o operador lê precisa dizer o que fazer.
	if detalhe == "" || detalhe == "falha na entrega pelo canal de mensagem" {
		t.Errorf("erro_detalhe = %q, quer a explicação específica da recusa", detalhe)
	}
}

// Falha temporária volta para a fila com espera, para tentar de novo.
func TestFalhaTemporariaVoltaParaAFila(t *testing.T) {
	canal := &canalFalso{erro: &disparo.ErroMeta{Status: 429, Codigo: 80007, Permanente: false}}
	h, fila, cookie, gdb := filaComCanal(t, canal)

	divida := semearDivida(t, gdb, "11111111111", "CT-1", "", 1000, 40)
	if status, _ := enfileirar(t, h, cookie, divida); status != http.StatusCreated {
		t.Fatal("enfileirar falhou")
	}

	if _, err := fila.ProcessarFila(context.Background()); err != nil {
		t.Fatalf("ProcessarFila: %v", err)
	}

	var status string
	var tentativas int
	if err := gdb.Raw(
		`SELECT status, tentativas FROM disparos WHERE divida_id = ?`, divida,
	).Row().Scan(&status, &tentativas); err != nil {
		t.Fatalf("consultar: %v", err)
	}
	if status != "na_fila" {
		t.Errorf("status = %q, quer na_fila para nova tentativa", status)
	}
	if tentativas != 1 {
		t.Errorf("tentativas = %d, quer 1", tentativas)
	}
}

// Sem canal, a fila não entrega e — o que mais importa — não mente dizendo
// que entregou.
func TestSemCanalNadaEhEntregueNemMarcado(t *testing.T) {
	h, fila, cookie, gdb := filaComCanal(t, nil)

	divida := semearDivida(t, gdb, "11111111111", "CT-1", "", 1000, 40)
	if status, _ := enfileirar(t, h, cookie, divida); status != http.StatusCreated {
		t.Fatal("enfileirar falhou")
	}

	enviados, err := fila.ProcessarFila(context.Background())
	if err != nil {
		t.Fatalf("ProcessarFila: %v", err)
	}
	if enviados != 0 {
		t.Errorf("entregou %d mensagens sem canal configurado", enviados)
	}

	var status string
	if err := gdb.Raw(`SELECT status FROM disparos WHERE divida_id = ?`, divida).Scan(&status).Error; err != nil {
		t.Fatalf("consultar: %v", err)
	}
	if status != "na_fila" {
		t.Errorf("status = %q, quer na_fila", status)
	}
}
