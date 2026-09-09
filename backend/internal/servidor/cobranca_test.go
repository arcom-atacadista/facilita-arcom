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
	// Um décimo do saldo é encargo. Semear encargo é o que faz o teste
	// exercitar desconto de verdade: a política incide SÓ sobre juros e multa,
	// então dívida sem encargo separado tem desconto zero por definição.
	encargos := valor / 10

	if err := gdb.Exec(
		`INSERT INTO dividas (id, cliente_id, contrato, valor_original, valor_encargos, vencimento, responsavel_cobranca)
		 VALUES (?, ?, ?, ?, ?, ?::date, ?)`,
		dividaID, clienteID, contrato, valor, encargos, vencimento, responsavel,
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
		"clienteId": clienteDaDivida(t, gdb, divida), "tipoPagamento": "pix", "descontoPct": 50, "parcelas": 1,
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
	if err := gdb.Raw(`
		SELECT count(*) FROM acordos a
		  JOIN acordo_dividas ad ON ad.acordo_id = a.id
		 WHERE ad.divida_id = ?`, divida).Scan(&quantos).Error; err != nil {
		t.Fatalf("contar acordos: %v", err)
	}
	if quantos != 0 {
		t.Errorf("gravou %d acordo(s) apesar da recusa", quantos)
	}

	// Dentro da alçada, passa.
	rec = chamar(t, h, http.MethodPost, "/api/v1/acordos", map[string]any{
		"clienteId": clienteDaDivida(t, gdb, divida), "tipoPagamento": "pix", "descontoPct": 20, "parcelas": 1,
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
		"clienteId": clienteDaDivida(t, gdb, divida), "tipoPagamento": "boleto", "descontoPct": 10, "parcelas": 6,
	}, cookieGerencia)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, quer 201 (%s)", rec.Code, rec.Body.String())
	}

	var somaParcelas, totalAcordo float64
	if err := gdb.Raw(`
		SELECT coalesce(sum(p.valor), 0), max(a.valor_total)
		  FROM acordos a
		  JOIN parcelas p ON p.acordo_id = a.id
		  JOIN acordo_dividas ad ON ad.acordo_id = a.id
		 WHERE ad.divida_id = ?`, divida).Row().Scan(&somaParcelas, &totalAcordo); err != nil {
		t.Fatalf("somar parcelas: %v", err)
	}

	if fmt.Sprintf("%.2f", somaParcelas) != fmt.Sprintf("%.2f", totalAcordo) {
		t.Errorf("soma das parcelas = %.2f, valor_total do acordo = %.2f", somaParcelas, totalAcordo)
	}
}

// O acordo é do CNPJ, então "segundo acordo" é sobre o mesmo cliente — antes
// era por título, o que deixava seis acordos ativos conviverem para o mesmo
// devedor.
func TestSegundoAcordoNoMesmoClienteEhRecusado(t *testing.T) {
	h, gdb := montarServidor(t)
	criarGerencia(t, h)
	cookieGerencia := logar(t, h, emailGerencia, senhaGerencia)

	divida := semearDivida(t, gdb, "11111111111", "CT-1", "", 1000, 40)
	corpo := map[string]any{"clienteId": clienteDaDivida(t, gdb, divida), "tipoPagamento": "pix", "descontoPct": 5, "parcelas": 1}

	if rec := chamar(t, h, http.MethodPost, "/api/v1/acordos", corpo, cookieGerencia); rec.Code != http.StatusCreated {
		t.Fatalf("primeiro acordo: status %d (%s)", rec.Code, rec.Body.String())
	}
	// Todos os títulos do CNPJ viraram "negociado" — a segunda tentativa é
	// barrada antes mesmo do índice único por cliente.
	if rec := chamar(t, h, http.MethodPost, "/api/v1/acordos", corpo, cookieGerencia); rec.Code != http.StatusConflict {
		t.Fatalf("segundo acordo: status %d, quer 409 (%s)", rec.Code, rec.Body.String())
	}
}

// semearDividaDoCliente acrescenta um título a um cliente que já existe — é o
// que permite montar a posição consolidada de um CNPJ com vários documentos.
func semearDividaDoCliente(t *testing.T, gdb *gorm.DB, clienteID uuid.UUID, contrato string, valor, encargos float64, diasAtraso int) uuid.UUID {
	t.Helper()
	dividaID := uuid.New()
	vencimento := time.Now().UTC().AddDate(0, 0, -diasAtraso).Format("2006-01-02")
	if err := gdb.Exec(
		`INSERT INTO dividas (id, cliente_id, contrato, valor_original, valor_encargos, vencimento)
		 VALUES (?, ?, ?, ?, ?, ?::date)`,
		dividaID, clienteID, contrato, valor, encargos, vencimento,
	).Error; err != nil {
		t.Fatalf("semear dívida do cliente: %v", err)
	}
	return dividaID
}

func clienteDaDivida(t *testing.T, gdb *gorm.DB, dividaID uuid.UUID) uuid.UUID {
	t.Helper()
	// Scan em string e parse: o gorm não converte o uuid do Postgres direto
	// para uuid.UUID, e o erro que ele dá ("converting driver.Value type
	// string to uint8") não indica isso em nada.
	var bruto string
	if err := gdb.Raw(`SELECT cliente_id::text FROM dividas WHERE id = ?`, dividaID).Scan(&bruto).Error; err != nil {
		t.Fatalf("achar cliente da dívida: %v", err)
	}
	id, err := uuid.Parse(bruto)
	if err != nil {
		t.Fatalf("cliente_id %q não é uuid: %v", bruto, err)
	}
	return id
}

type respostaOfertas struct {
	Posicao struct {
		Saldo            float64 `json:"saldo"`
		Encargos         float64 `json:"encargos"`
		Principal        float64 `json:"principal"`
		Titulos          int     `json:"titulos"`
		DiasAtrasoMaximo int     `json:"diasAtrasoMaximo"`
		Faixa            string  `json:"faixa"`
	} `json:"posicao"`
	Ofertas struct {
		Avista struct {
			DescontoPct    float64 `json:"descontoPct"`
			Desconto       float64 `json:"desconto"`
			ValorTotal     float64 `json:"valorTotal"`
			ExigeAprovacao bool    `json:"exigeAprovacao"`
		} `json:"avista"`
		Parcelado *struct {
			MaxParcelas int `json:"maxParcelas"`
		} `json:"parcelado"`
		Ampliada *struct {
			DescontoPct    float64 `json:"descontoPct"`
			Desconto       float64 `json:"desconto"`
			ExigeAprovacao bool    `json:"exigeAprovacao"`
		} `json:"ampliada"`
	} `json:"ofertas"`
	AlcadaMaxima float64 `json:"alcadaMaxima"`
}

func buscarOfertas(t *testing.T, h http.Handler, cookie *http.Cookie, clienteID uuid.UUID) respostaOfertas {
	t.Helper()
	rec := chamar(t, h, http.MethodGet, "/api/v1/carteira/clientes/"+clienteID.String()+"/ofertas", nil, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("ofertas: status %d, quer 200 (%s)", rec.Code, rec.Body.String())
	}
	var r respostaOfertas
	if err := json.Unmarshal(rec.Body.Bytes(), &r); err != nil {
		t.Fatalf("resposta inesperada: %s", rec.Body.String())
	}
	return r
}

func TestOfertasConsolidamTodosOsTitulosDoCNPJ(t *testing.T) {
	// A regra da ARCOM: não se parcela título isolado, o acordo engloba todos
	// os títulos em atraso do CNPJ.
	h, gdb := montarServidor(t)
	criarGerencia(t, h)
	cookie := logar(t, h, emailGerencia, senhaGerencia)

	primeira := semearDivida(t, gdb, "11222333000181", "CT-1", "", 1000, 40)
	cliente := clienteDaDivida(t, gdb, primeira)
	semearDividaDoCliente(t, gdb, cliente, "CT-2", 500, 50, 70)
	semearDividaDoCliente(t, gdb, cliente, "CT-3", 300, 30, 20)

	r := buscarOfertas(t, h, cookie, cliente)

	// 1000 (com 100 de encargo, do semearDivida) + 500 + 300.
	if quer := 1800.00; r.Posicao.Saldo != quer {
		t.Errorf("saldo = %v, quer %v", r.Posicao.Saldo, quer)
	}
	if quer := 180.00; r.Posicao.Encargos != quer {
		t.Errorf("encargos = %v, quer %v", r.Posicao.Encargos, quer)
	}
	if quer := 1620.00; r.Posicao.Principal != quer {
		t.Errorf("principal = %v, quer %v", r.Posicao.Principal, quer)
	}
	if r.Posicao.Titulos != 3 {
		t.Errorf("títulos = %d, quer 3", r.Posicao.Titulos)
	}
	// A faixa é a do título mais velho: a política aplicada é a do pior atraso.
	if r.Posicao.DiasAtrasoMaximo != 70 {
		t.Errorf("dias = %d, quer 70", r.Posicao.DiasAtrasoMaximo)
	}
	if r.Posicao.Faixa != "61-90" {
		t.Errorf("faixa = %q, quer 61-90", r.Posicao.Faixa)
	}
}

func TestDescontoDoEndpointNuncaPassaDosEncargos(t *testing.T) {
	// A trava contra a volta da conta antiga, exercitada pela API inteira e não
	// só pela função: a política da faixa 61-90 concede 20% à vista, e 20% do
	// saldo seriam R$ 360 em vez dos R$ 36 dos encargos.
	h, gdb := montarServidor(t)
	criarGerencia(t, h)
	cookie := logar(t, h, emailGerencia, senhaGerencia)

	primeira := semearDivida(t, gdb, "11222333000181", "CT-1", "", 1800, 70)
	cliente := clienteDaDivida(t, gdb, primeira)

	r := buscarOfertas(t, h, cookie, cliente)

	if r.Ofertas.Avista.Desconto > r.Posicao.Encargos {
		t.Errorf("desconto de %v passou dos encargos (%v) — voltou a incidir sobre o principal",
			r.Ofertas.Avista.Desconto, r.Posicao.Encargos)
	}
	if quer := r.Posicao.Saldo - r.Ofertas.Avista.Desconto; r.Ofertas.Avista.ValorTotal != quer {
		t.Errorf("valor à vista = %v, quer %v", r.Ofertas.Avista.ValorTotal, quer)
	}
}

func TestOfertaAcimaDaAlcadaVemTravadaEnaoEscondida(t *testing.T) {
	h, gdb := montarServidor(t)
	criarGerencia(t, h)
	cookieGerencia := logar(t, h, emailGerencia, senhaGerencia)

	primeira := semearDivida(t, gdb, "11222333000181", "CT-1", "APR", 1800, 70)
	cliente := clienteDaDivida(t, gdb, primeira)

	// Aprendiz tem alçada de 20%, e a política da faixa 61-90 concede 20% à
	// vista — igual, então nada travado. Coordenação vê o mesmo.
	cookieAprendiz := criarOperador(t, h, cookieGerencia, "apr@arcom.com.br", "aprendiz", "APR")
	r := buscarOfertas(t, h, cookieAprendiz, cliente)
	if r.AlcadaMaxima != 20 {
		t.Errorf("alçada do aprendiz = %v, quer 20", r.AlcadaMaxima)
	}
	if r.Ofertas.Avista.DescontoPct > r.AlcadaMaxima {
		t.Errorf("ofereceu %v%% para quem tem alçada de %v%%", r.Ofertas.Avista.DescontoPct, r.AlcadaMaxima)
	}
}

func TestOfertasDeClienteForaDaCarteiraRespondem404(t *testing.T) {
	// Mesmo motivo do resto do sistema: 403 confirmaria que o CNPJ existe na
	// carteira de outro analista.
	h, gdb := montarServidor(t)
	criarGerencia(t, h)
	cookieGerencia := logar(t, h, emailGerencia, senhaGerencia)

	doBruno := semearDivida(t, gdb, "33333333333", "CT-BRUNO", "BRUNO", 900, 70)
	cliente := clienteDaDivida(t, gdb, doBruno)

	cookieAna := criarOperador(t, h, cookieGerencia, "ana2@arcom.com.br", "analista", "ANA")
	rec := chamar(t, h, http.MethodGet, "/api/v1/carteira/clientes/"+cliente.String()+"/ofertas", nil, cookieAna)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, quer 404 (%s)", rec.Code, rec.Body.String())
	}
}

func TestAcordoCobreTodosOsTitulosDoCNPJ(t *testing.T) {
	// Se um título coberto ficasse "aberto", ele voltaria para a régua cobrando
	// quem já fechou acordo.
	h, gdb := montarServidor(t)
	criarGerencia(t, h)
	cookie := logar(t, h, emailGerencia, senhaGerencia)

	primeira := semearDivida(t, gdb, "11222333000181", "CT-1", "", 1000, 40)
	cliente := clienteDaDivida(t, gdb, primeira)
	semearDividaDoCliente(t, gdb, cliente, "CT-2", 500, 50, 70)
	semearDividaDoCliente(t, gdb, cliente, "CT-3", 300, 30, 45)

	rec := chamar(t, h, http.MethodPost, "/api/v1/acordos", map[string]any{
		"clienteId": cliente, "tipoPagamento": "pix", "descontoPct": 10, "parcelas": 1,
	}, cookie)
	if rec.Code != http.StatusCreated {
		t.Fatalf("fechar acordo: status %d (%s)", rec.Code, rec.Body.String())
	}

	var cobertos, abertos int64
	if err := gdb.Raw(`SELECT count(*) FROM acordo_dividas`).Scan(&cobertos).Error; err != nil {
		t.Fatalf("contar cobertura: %v", err)
	}
	if cobertos != 3 {
		t.Errorf("títulos cobertos = %d, quer 3", cobertos)
	}
	if err := gdb.Raw(`SELECT count(*) FROM dividas WHERE cliente_id = ? AND status = 'aberto'`, cliente).
		Scan(&abertos).Error; err != nil {
		t.Fatalf("contar abertos: %v", err)
	}
	if abertos != 0 {
		t.Errorf("%d título(s) ficaram abertos depois do acordo — voltariam para a régua", abertos)
	}
}

func TestOfertasMostramOAcordoEmVigorEmVezDeVazio(t *testing.T) {
	// Depois de fechar, todos os títulos viram "negociado" e a posição aberta
	// fica vazia. Sem devolver o acordo, a mesa diria "sem título em atraso"
	// logo depois de o analista fechar um — o que pareceria acordo não gravado.
	h, gdb := montarServidor(t)
	criarGerencia(t, h)
	cookie := logar(t, h, emailGerencia, senhaGerencia)

	primeira := semearDivida(t, gdb, "11222333000181", "CT-1", "", 1000, 40)
	cliente := clienteDaDivida(t, gdb, primeira)
	semearDividaDoCliente(t, gdb, cliente, "CT-2", 500, 50, 45)

	// Antes: posição aberta, sem acordo.
	rec := chamar(t, h, http.MethodGet, "/api/v1/carteira/clientes/"+cliente.String()+"/ofertas", nil, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("antes: status %d (%s)", rec.Code, rec.Body.String())
	}
	var antes struct {
		Posicao struct{ Titulos int } `json:"posicao"`
		Acordo  *struct{}             `json:"acordoAtivo"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &antes); err != nil {
		t.Fatalf("ler antes: %v", err)
	}
	if antes.Posicao.Titulos != 2 || antes.Acordo != nil {
		t.Fatalf("antes do acordo: títulos %d, acordo %v", antes.Posicao.Titulos, antes.Acordo)
	}

	if rec := chamar(t, h, http.MethodPost, "/api/v1/acordos", map[string]any{
		"clienteId": cliente, "tipoPagamento": "pix", "descontoPct": 10, "parcelas": 1,
	}, cookie); rec.Code != http.StatusCreated {
		t.Fatalf("fechar acordo: status %d (%s)", rec.Code, rec.Body.String())
	}

	// Depois: 200 com o acordo, e não 404 de "sem posição".
	rec = chamar(t, h, http.MethodGet, "/api/v1/carteira/clientes/"+cliente.String()+"/ofertas", nil, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("depois: status %d, quer 200 (%s)", rec.Code, rec.Body.String())
	}
	var depois struct {
		Acordo *struct {
			Titulos    int     `json:"titulos"`
			ValorTotal float64 `json:"valorTotal"`
			Status     string  `json:"status"`
		} `json:"acordoAtivo"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &depois); err != nil {
		t.Fatalf("ler depois: %v", err)
	}
	if depois.Acordo == nil {
		t.Fatal("depois de fechar, a resposta tem que trazer o acordo em vigor")
	}
	if depois.Acordo.Titulos != 2 {
		t.Errorf("títulos cobertos = %d, quer 2", depois.Acordo.Titulos)
	}
	if depois.Acordo.Status != "ativo" {
		t.Errorf("status = %q, quer ativo", depois.Acordo.Status)
	}
}

func TestFecharAcordoDeClienteForaDaCarteiraRespondem404(t *testing.T) {
	// A escrita tem o mesmo recorte da leitura: trocar o clienteId no corpo não
	// fecha acordo na carteira de outro analista.
	h, gdb := montarServidor(t)
	criarGerencia(t, h)
	cookieGerencia := logar(t, h, emailGerencia, senhaGerencia)

	doBruno := semearDivida(t, gdb, "33333333333", "CT-BRUNO", "BRUNO", 900, 70)
	cliente := clienteDaDivida(t, gdb, doBruno)

	cookieAna := criarOperador(t, h, cookieGerencia, "ana3@arcom.com.br", "analista", "ANA")
	rec := chamar(t, h, http.MethodPost, "/api/v1/acordos", map[string]any{
		"clienteId": cliente, "tipoPagamento": "pix", "descontoPct": 5, "parcelas": 1,
	}, cookieAna)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, quer 404 (%s)", rec.Code, rec.Body.String())
	}

	var quantos int64
	if err := gdb.Raw(`SELECT count(*) FROM acordos`).Scan(&quantos).Error; err != nil {
		t.Fatalf("contar: %v", err)
	}
	if quantos != 0 {
		t.Errorf("gravou %d acordo(s) fora do escopo", quantos)
	}
}
