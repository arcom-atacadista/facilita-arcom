package servidor_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// semearDivida cria cliente + dívida direto no banco. Vai por SQL e não pela
// API de propósito: a importação da carteira vem do Gateway ARCOM, não de um
// endpoint de cadastro, então não existe rota pra isso.
func semearDivida(t *testing.T, gdb *gorm.DB, documento, contrato, responsavel string, valor float64, diasAtraso int) uuid.UUID {
	t.Helper()

	clienteID, dividaID := uuid.New(), uuid.New()
	vencimento := time.Now().UTC().AddDate(0, 0, -diasAtraso).Format("2006-01-02")

	if err := gdb.Exec(
		`INSERT INTO clientes (id, nome, documento, telefone) VALUES (?, ?, ?, ?)`,
		clienteID, "Cliente "+contrato, documento, "31988887777",
	).Error; err != nil {
		t.Fatalf("semear cliente: %v", err)
	}
	if err := gdb.Exec(
		`INSERT INTO dividas (id, cliente_id, contrato, valor_original, vencimento, responsavel_cobranca)
		 VALUES (?, ?, ?, ?, ?::date, ?)`,
		dividaID, clienteID, contrato, valor, vencimento, responsavel,
	).Error; err != nil {
		t.Fatalf("semear dívida: %v", err)
	}
	return dividaID
}

// criarOperador cria um usuário via API (como gerência faria) e devolve o
// cookie já logado.
func criarOperador(t *testing.T, h http.Handler, cookieGerencia *http.Cookie, email, papel, codigoCobranca string) *http.Cookie {
	t.Helper()

	const senha = "SenhaForte2026!"
	corpo := map[string]any{
		"nome": email, "email": email, "senha": senha,
		"papel": papel, "equipe": "interno",
	}
	if codigoCobranca != "" {
		corpo["codigoCobranca"] = codigoCobranca
	}

	rec := chamar(t, h, http.MethodPost, "/api/v1/usuarios", corpo, cookieGerencia)
	if rec.Code != http.StatusCreated {
		t.Fatalf("criar %s: status %d (%s)", papel, rec.Code, rec.Body.String())
	}
	return logar(t, h, email, senha)
}

func listarCarteira(t *testing.T, h http.Handler, cookie *http.Cookie) []string {
	t.Helper()

	rec := chamar(t, h, http.MethodGet, "/api/v1/carteira", nil, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("listar carteira: status %d (%s)", rec.Code, rec.Body.String())
	}

	var resp struct {
		Itens []struct {
			Contrato string `json:"contrato"`
		} `json:"itens"`
		Paginacao struct {
			Total int `json:"total"`
		} `json:"paginacao"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("resposta inesperada: %s", rec.Body.String())
	}
	if resp.Paginacao.Total != len(resp.Itens) {
		t.Errorf("paginacao.total = %d mas vieram %d itens", resp.Paginacao.Total, len(resp.Itens))
	}

	contratos := make([]string, 0, len(resp.Itens))
	for _, i := range resp.Itens {
		contratos = append(contratos, i.Contrato)
	}
	return contratos
}

// Este é o teste da correção de LGPD: no modelo Supabase toda policy era
// USING (true), então qualquer autenticado lia a base inteira de devedores.
func TestAnalistaVeApenasAPropriaCarteira(t *testing.T) {
	h, gdb := montarServidor(t)
	criarGerencia(t, h)
	cookieGerencia := logar(t, h, emailGerencia, senhaGerencia)

	semearDivida(t, gdb, "11111111111", "CT-ANA-1", "ANA", 15, 10)
	semearDivida(t, gdb, "22222222222", "CT-ANA-2", "ANA", 25, 40)
	semearDivida(t, gdb, "33333333333", "CT-BRUNO-1", "BRUNO", 35, 70)

	cookieAna := criarOperador(t, h, cookieGerencia, "ana@arcom.com.br", "analista", "ANA")
	contratos := listarCarteira(t, h, cookieAna)

	if len(contratos) != 2 {
		t.Fatalf("analista viu %d dívidas (%v), quer 2 — só a carteira dela", len(contratos), contratos)
	}
	for _, c := range contratos {
		if c == "CT-BRUNO-1" {
			t.Error("analista enxergou a carteira de outro operador")
		}
	}
}

func TestAnalistaSemCodigoDeCobrancaNaoVeNada(t *testing.T) {
	h, gdb := montarServidor(t)
	criarGerencia(t, h)
	cookieGerencia := logar(t, h, emailGerencia, senhaGerencia)

	semearDivida(t, gdb, "11111111111", "CT-1", "ANA", 500, 10)

	// Falha fechado: sem código, não é dono de carteira nenhuma. O contrário
	// (ver tudo) seria vazar a base inteira por um cadastro incompleto.
	cookieSemCodigo := criarOperador(t, h, cookieGerencia, "novato@arcom.com.br", "analista", "")
	if contratos := listarCarteira(t, h, cookieSemCodigo); len(contratos) != 0 {
		t.Errorf("analista sem código viu %v, quer nada", contratos)
	}
}

func TestCoordenacaoVeACarteiraInteira(t *testing.T) {
	h, gdb := montarServidor(t)
	criarGerencia(t, h)
	cookieGerencia := logar(t, h, emailGerencia, senhaGerencia)

	semearDivida(t, gdb, "11111111111", "CT-ANA-1", "ANA", 500, 10)
	semearDivida(t, gdb, "33333333333", "CT-BRUNO-1", "BRUNO", 900, 70)

	cookieCoord := criarOperador(t, h, cookieGerencia, "coord@arcom.com.br", "coordenacao", "")
	if contratos := listarCarteira(t, h, cookieCoord); len(contratos) != 2 {
		t.Errorf("coordenação viu %v, quer as 2 dívidas", contratos)
	}
}

func TestAnalistaNaoAbreDividaDeOutraCarteira(t *testing.T) {
	h, gdb := montarServidor(t)
	criarGerencia(t, h)
	cookieGerencia := logar(t, h, emailGerencia, senhaGerencia)

	dividaDoBruno := semearDivida(t, gdb, "33333333333", "CT-BRUNO-1", "BRUNO", 900, 70)
	cookieAna := criarOperador(t, h, cookieGerencia, "ana@arcom.com.br", "analista", "ANA")

	// Trocar o id na URL é o IDOR clássico. 404 e não 403: responder 403
	// confirmaria que a dívida existe na carteira de outra pessoa.
	rec := chamar(t, h, http.MethodGet, "/api/v1/carteira/"+dividaDoBruno.String(), nil, cookieAna)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, quer 404 (%s)", rec.Code, rec.Body.String())
	}
}

// A alçada existia no modelo antigo só como número na tela: nada no servidor
// impedia gravar um acordo com desconto acima dela.
func TestAlcadaEhAplicadaNoServidor(t *testing.T) {
	h, gdb := montarServidor(t)
	criarGerencia(t, h)
	cookieGerencia := logar(t, h, emailGerencia, senhaGerencia)

	// 40 dias de atraso = faixa 31-60, política de até 6 parcelas.
	divida := semearDivida(t, gdb, "11111111111", "CT-1", "ANA", 1000, 40)
	cookieAprendiz := criarOperador(t, h, cookieGerencia, "aprendiz@arcom.com.br", "aprendiz", "ANA")

	// Aprendiz tem alçada de 20%.
	rec := chamar(t, h, http.MethodPost, "/api/v1/acordos", map[string]any{
		"dividaId": divida, "tipoPagamento": "pix", "descontoPct": 50, "parcelas": 1,
	}, cookieAprendiz)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("desconto de 50%% com alçada de 20%%: status = %d, quer 403 (%s)", rec.Code, rec.Body.String())
	}

	var p struct {
		Codigo string `json:"codigo"`
		Detail string `json:"detail"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatalf("corpo inesperado: %s", rec.Body.String())
	}
	if p.Codigo != "acima_da_alcada" {
		t.Errorf("codigo = %q, quer acima_da_alcada", p.Codigo)
	}

	// E o acordo não pode ter sido gravado.
	var quantos int64
	if err := gdb.Raw(`SELECT count(*) FROM acordos WHERE divida_id = ?`, divida).Scan(&quantos).Error; err != nil {
		t.Fatalf("contar acordos: %v", err)
	}
	if quantos != 0 {
		t.Errorf("gravou %d acordo(s) apesar da recusa", quantos)
	}

	// Dentro da alçada, passa.
	rec = chamar(t, h, http.MethodPost, "/api/v1/acordos", map[string]any{
		"dividaId": divida, "tipoPagamento": "pix", "descontoPct": 20, "parcelas": 1,
	}, cookieAprendiz)
	if rec.Code != http.StatusCreated {
		t.Fatalf("desconto de 20%% com alçada de 20%%: status = %d, quer 201 (%s)", rec.Code, rec.Body.String())
	}
}

func TestSomaDasParcelasBateComOTotalDoAcordo(t *testing.T) {
	h, gdb := montarServidor(t)
	criarGerencia(t, h)
	cookieGerencia := logar(t, h, emailGerencia, senhaGerencia)

	// Valor que não divide redondo em 6.
	divida := semearDivida(t, gdb, "11111111111", "CT-1", "", 1250.90, 40)

	rec := chamar(t, h, http.MethodPost, "/api/v1/acordos", map[string]any{
		"dividaId": divida, "tipoPagamento": "boleto", "descontoPct": 10, "parcelas": 6,
	}, cookieGerencia)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, quer 201 (%s)", rec.Code, rec.Body.String())
	}

	var somaParcelas, totalAcordo float64
	if err := gdb.Raw(`
		SELECT coalesce(sum(p.valor), 0), max(a.valor_total)
		FROM acordos a JOIN parcelas p ON p.acordo_id = a.id
		WHERE a.divida_id = ?`, divida).Row().Scan(&somaParcelas, &totalAcordo); err != nil {
		t.Fatalf("somar parcelas: %v", err)
	}

	if fmt.Sprintf("%.2f", somaParcelas) != fmt.Sprintf("%.2f", totalAcordo) {
		t.Errorf("soma das parcelas = %.2f, valor_total do acordo = %.2f", somaParcelas, totalAcordo)
	}
}

func TestSegundoAcordoNaMesmaDividaEhRecusado(t *testing.T) {
	h, gdb := montarServidor(t)
	criarGerencia(t, h)
	cookieGerencia := logar(t, h, emailGerencia, senhaGerencia)

	divida := semearDivida(t, gdb, "11111111111", "CT-1", "", 1000, 40)
	corpo := map[string]any{"dividaId": divida, "tipoPagamento": "pix", "descontoPct": 5, "parcelas": 1}

	if rec := chamar(t, h, http.MethodPost, "/api/v1/acordos", corpo, cookieGerencia); rec.Code != http.StatusCreated {
		t.Fatalf("primeiro acordo: status %d (%s)", rec.Code, rec.Body.String())
	}
	// A dívida virou "negociado" — a segunda tentativa é barrada antes mesmo
	// do índice único.
	if rec := chamar(t, h, http.MethodPost, "/api/v1/acordos", corpo, cookieGerencia); rec.Code != http.StatusConflict {
		t.Fatalf("segundo acordo: status %d, quer 409 (%s)", rec.Code, rec.Body.String())
	}
}
