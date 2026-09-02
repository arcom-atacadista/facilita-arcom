package servidor_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"

	"facilitaarcom/internal/carteira"
	"facilitaarcom/internal/gatewayarcom"
)

// gatewayFalso responde no formato do Gateway ARCOM. Nenhum teste toca a API
// interna de verdade — ela é produção da empresa e a chave só existe lá.
type gatewayFalso struct {
	*httptest.Server
	corpo string
}

func novoGatewayFalso(t *testing.T, corpo string) *gatewayFalso {
	t.Helper()
	g := &gatewayFalso{corpo: corpo}
	g.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(g.corpo))
	}))
	t.Cleanup(g.Close)
	return g
}

// clienteApontandoPara redireciona o client do Gateway para o servidor falso,
// mantendo o caminho — mesmo efeito de trocar a URL base, sem abrir a
// constante para configuração.
func clienteApontandoPara(t *testing.T, g *gatewayFalso) *gatewayarcom.Cliente {
	t.Helper()
	return gatewayarcom.NovoCliente("chave-de-teste").ComBaseURL(g.URL)
}

func servicoDeCarteira(t *testing.T, g *gatewayFalso) (*carteira.Service, *gorm.DB) {
	t.Helper()
	gdb := bancoDeTeste(t)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return carteira.NovoService(gdb, clienteApontandoPara(t, g), gatewayarcom.MapeamentoPadrao(), log), gdb
}

// debitoJSON monta uma linha do dataset no formato que o Gateway devolve.
func debitoJSON(seq, cnpj, razao string, valor float64, diasAtraso int, responsavel string) string {
	venc := time.Now().UTC().AddDate(0, 0, -diasAtraso).Format("2006-01-02")
	return fmt.Sprintf(`{"seq_debito":%q,"cnpj":%q,"razao_social":%q,"vlr_liquido_deb":%v,"data_vencto":%q,"filial":"BH","responsavel_cobranca":%q}`,
		seq, cnpj, razao, valor, venc, responsavel)
}

func envelope(linhas ...string) string {
	return `{"itens":[` + strings.Join(linhas, ",") + `]}`
}

func TestSincronizacaoTrazACarteiraDoGateway(t *testing.T) {
	g := novoGatewayFalso(t, envelope(
		debitoJSON("CT-1", "12.345.678/0001-99", "Mercado do João LTDA", 1250.90, 40, "ANA"),
		debitoJSON("CT-2", "98765432000155", "Padaria Estrela ME", 480.00, 12, "BRUNO"),
	))
	svc, gdb := servicoDeCarteira(t, g)

	res, err := svc.Sincronizar(context.Background())
	if err != nil {
		t.Fatalf("Sincronizar: %v", err)
	}
	if res.Criadas != 2 {
		t.Fatalf("criou %d dívidas, quer 2 (resultado: %+v)", res.Criadas, res)
	}

	var nome, documento, contrato, responsavel, origem string
	var valor float64
	if err := gdb.Raw(`
		SELECT c.nome, c.documento, d.contrato, coalesce(d.responsavel_cobranca,''), d.origem, d.valor_original
		FROM dividas d JOIN clientes c ON c.id = d.cliente_id WHERE d.contrato = 'CT-1'`,
	).Row().Scan(&nome, &documento, &contrato, &responsavel, &origem, &valor); err != nil {
		t.Fatalf("consultar: %v", err)
	}

	if nome != "Mercado do João LTDA" {
		t.Errorf("nome = %q, quer a razão social", nome)
	}
	// O CNPJ vem com máscara do Gateway e precisa entrar normalizado, senão o
	// mesmo cliente viraria dois cadastros.
	if documento != "12345678000199" {
		t.Errorf("documento = %q, quer só dígitos", documento)
	}
	if responsavel != "ANA" {
		t.Errorf("responsável = %q — é o que dá o recorte de carteira", responsavel)
	}
	if origem != "gateway" {
		t.Errorf("origem = %q, quer gateway", origem)
	}
	if valor != 1250.90 {
		t.Errorf("valor = %v", valor)
	}
}

// Rodar duas vezes não pode duplicar nada: a sincronização é diária.
func TestSincronizacaoRodadaDuasVezesNaoDuplica(t *testing.T) {
	g := novoGatewayFalso(t, envelope(
		debitoJSON("CT-1", "12345678000199", "Mercado do João LTDA", 1250.90, 40, "ANA"),
	))
	svc, gdb := servicoDeCarteira(t, g)

	if _, err := svc.Sincronizar(context.Background()); err != nil {
		t.Fatalf("primeira rodada: %v", err)
	}

	// Valor muda no Gateway entre as rodadas.
	g.corpo = envelope(debitoJSON("CT-1", "12345678000199", "Mercado do João LTDA", 1300.00, 41, "ANA"))
	res, err := svc.Sincronizar(context.Background())
	if err != nil {
		t.Fatalf("segunda rodada: %v", err)
	}
	if res.Criadas != 0 || res.Atualizadas != 1 {
		t.Errorf("resultado = %+v, quer 0 criadas e 1 atualizada", res)
	}

	var dividas, clientes int64
	var valor float64
	_ = gdb.Raw(`SELECT count(*) FROM dividas`).Scan(&dividas).Error
	_ = gdb.Raw(`SELECT count(*) FROM clientes`).Scan(&clientes).Error
	_ = gdb.Raw(`SELECT valor_original FROM dividas WHERE contrato='CT-1'`).Scan(&valor).Error

	if dividas != 1 || clientes != 1 {
		t.Errorf("%d dívidas e %d clientes, quer 1 de cada", dividas, clientes)
	}
	if valor != 1300.00 {
		t.Errorf("valor = %v, quer o atualizado", valor)
	}
}

// Dívida que some do Gateway foi quase certamente paga. Continuar cobrando
// quem pagou é pior do que deixar de cobrar quem deve.
func TestDividaQueSaiDoGatewayEhFechada(t *testing.T) {
	g := novoGatewayFalso(t, envelope(
		debitoJSON("CT-1", "12345678000199", "Mercado A", 1000, 40, "ANA"),
		debitoJSON("CT-2", "98765432000155", "Mercado B", 500, 20, "ANA"),
	))
	svc, gdb := servicoDeCarteira(t, g)

	if _, err := svc.Sincronizar(context.Background()); err != nil {
		t.Fatalf("primeira rodada: %v", err)
	}

	// CT-2 sai da origem: foi pago.
	g.corpo = envelope(debitoJSON("CT-1", "12345678000199", "Mercado A", 1000, 41, "ANA"))
	res, err := svc.Sincronizar(context.Background())
	if err != nil {
		t.Fatalf("segunda rodada: %v", err)
	}
	if res.Fechadas != 1 {
		t.Errorf("fechou %d, quer 1", res.Fechadas)
	}

	var status1, status2 string
	_ = gdb.Raw(`SELECT status FROM dividas WHERE contrato='CT-1'`).Scan(&status1).Error
	_ = gdb.Raw(`SELECT status FROM dividas WHERE contrato='CT-2'`).Scan(&status2).Error
	if status1 != "aberto" {
		t.Errorf("CT-1 = %q, quer aberto (continua no Gateway)", status1)
	}
	if status2 != "quitado" {
		t.Errorf("CT-2 = %q, quer quitado (saiu do Gateway)", status2)
	}
}

// Se o Gateway devolver vazio por instabilidade e a gente fechasse tudo, a
// carteira inteira sumiria de uma vez.
func TestGatewayVazioNaoFechaACarteira(t *testing.T) {
	g := novoGatewayFalso(t, envelope(
		debitoJSON("CT-1", "12345678000199", "Mercado A", 1000, 40, "ANA"),
	))
	svc, gdb := servicoDeCarteira(t, g)

	if _, err := svc.Sincronizar(context.Background()); err != nil {
		t.Fatalf("primeira rodada: %v", err)
	}

	g.corpo = `{"itens":[]}`
	res, err := svc.Sincronizar(context.Background())
	if err != nil {
		t.Fatalf("rodada vazia: %v", err)
	}
	if res.Fechadas != 0 {
		t.Errorf("fechou %d dívidas numa rodada vazia", res.Fechadas)
	}

	var status string
	_ = gdb.Raw(`SELECT status FROM dividas WHERE contrato='CT-1'`).Scan(&status).Error
	if status != "aberto" {
		t.Errorf("status = %q — a dívida não podia ter sido fechada", status)
	}
}

// Dívida lançada à mão por um operador não some por não estar no Gateway.
func TestDividaManualNaoEhFechadaPelaSincronizacao(t *testing.T) {
	g := novoGatewayFalso(t, envelope(
		debitoJSON("CT-GW", "12345678000199", "Mercado A", 1000, 40, "ANA"),
	))
	svc, gdb := servicoDeCarteira(t, g)

	manual := semearDivida(t, gdb, "55555555555", "CT-MANUAL", "ANA", 700, 30)

	if _, err := svc.Sincronizar(context.Background()); err != nil {
		t.Fatalf("Sincronizar: %v", err)
	}

	var status, origem string
	if err := gdb.Raw(`SELECT status, origem FROM dividas WHERE id = ?`, manual).Row().Scan(&status, &origem); err != nil {
		t.Fatalf("consultar: %v", err)
	}
	if origem != "manual" {
		t.Errorf("origem = %q", origem)
	}
	if status != "aberto" {
		t.Errorf("a dívida manual foi fechada (%q) pela sincronização", status)
	}
}

// A régua é de 3 a 90 dias: o que ainda não venceu e o que já passou para o
// jurídico não é assunto desta plataforma.
func TestSincronizacaoRespeitaAFaixaDaRegua(t *testing.T) {
	g := novoGatewayFalso(t, envelope(
		debitoJSON("CT-NOVA", "11111111111111", "Recente", 100, 1, "ANA"),   // ainda não entrou
		debitoJSON("CT-BOA", "22222222222222", "Na faixa", 200, 45, "ANA"),  // entra
		debitoJSON("CT-VELHA", "33333333333333", "Antiga", 300, 200, "ANA"), // já saiu
	))
	svc, gdb := servicoDeCarteira(t, g)

	res, err := svc.Sincronizar(context.Background())
	if err != nil {
		t.Fatalf("Sincronizar: %v", err)
	}
	if res.Criadas != 1 {
		t.Errorf("criou %d, quer 1 (resultado: %+v)", res.Criadas, res)
	}
	if res.Ignoradas != 2 {
		t.Errorf("ignorou %d, quer 2", res.Ignoradas)
	}

	var contratos []string
	_ = gdb.Raw(`SELECT contrato FROM dividas ORDER BY contrato`).Scan(&contratos).Error
	if len(contratos) != 1 || contratos[0] != "CT-BOA" {
		t.Errorf("carteira = %v, quer só CT-BOA", contratos)
	}
}

// Uma dívida já negociada não pode voltar a "aberto" só porque continua
// aparecendo no Gateway até a primeira parcela compensar.
func TestSincronizacaoNaoReabreDividaNegociada(t *testing.T) {
	g := novoGatewayFalso(t, envelope(
		debitoJSON("CT-1", "12345678000199", "Mercado A", 1000, 40, "ANA"),
	))
	svc, gdb := servicoDeCarteira(t, g)

	if _, err := svc.Sincronizar(context.Background()); err != nil {
		t.Fatalf("primeira rodada: %v", err)
	}
	if err := gdb.Exec(`UPDATE dividas SET status='negociado' WHERE contrato='CT-1'`).Error; err != nil {
		t.Fatalf("marcar negociado: %v", err)
	}

	if _, err := svc.Sincronizar(context.Background()); err != nil {
		t.Fatalf("segunda rodada: %v", err)
	}

	var status string
	_ = gdb.Raw(`SELECT status FROM dividas WHERE contrato='CT-1'`).Scan(&status).Error
	if status != "negociado" {
		t.Errorf("status = %q, quer negociado", status)
	}
}

// Sem a chave do Gateway a sincronização recusa com erro claro, em vez de
// devolver carteira vazia como se não houvesse inadimplência.
func TestSincronizacaoSemChaveRecusa(t *testing.T) {
	gdb := bancoDeTeste(t)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := carteira.NovoService(gdb, gatewayarcom.NovoCliente(""), gatewayarcom.MapeamentoPadrao(), log)

	if _, err := svc.Sincronizar(context.Background()); err == nil {
		t.Fatal("aceitou sincronizar sem credencial")
	}
}

// A rodada fica registrada para a operação saber se a carteira de hoje entrou
// sem precisar abrir log de servidor.
func TestSincronizacaoFicaRegistrada(t *testing.T) {
	g := novoGatewayFalso(t, envelope(
		debitoJSON("CT-1", "12345678000199", "Mercado A", 1000, 40, "ANA"),
	))
	svc, _ := servicoDeCarteira(t, g)

	if _, err := svc.Sincronizar(context.Background()); err != nil {
		t.Fatalf("Sincronizar: %v", err)
	}

	u, err := svc.Ultima(context.Background())
	if err != nil {
		t.Fatalf("Ultima: %v", err)
	}
	if u == nil {
		t.Fatal("nenhuma rodada registrada")
	}
	if u.Status != "concluida" {
		t.Errorf("status = %q", u.Status)
	}
	if u.Criadas != 1 || u.Lidas != 1 {
		t.Errorf("registro = %+v", u)
	}
	if u.TerminadaEm == nil {
		t.Error("terminada_em ficou nula numa rodada concluída")
	}
}
