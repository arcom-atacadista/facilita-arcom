// Comando de diagnóstico que responde, em dez segundos e sem enviar nada,
// se o disparo pela Cloud API da Meta vai funcionar.
//
// # POR QUE ISTO EXISTE
//
// O canal de envio está pronto no código, mas ele só funciona se quatro
// coisas estiverem certas do lado da Meta ao mesmo tempo: o token vale, o
// número está mesmo na Cloud API (e não preso a um BSP), os templates estão
// aprovados, e a contagem de marcadores de cada template bate com o que a
// régua manda. Errar qualquer uma delas dá o mesmo sintoma no worker — uma
// linha de erro genérica no log, no meio de um disparo real.
//
// Este comando confere as quatro antes de a primeira mensagem sair, lendo o
// que a Meta responde e comparando com o que a tabela campanhas espera.
//
// NÃO IMPRIME O TOKEN, em nenhuma saída — nem mascarado. Rode você mesmo e
// reporte o resultado; a credencial não precisa sair da sua máquina.
//
//	WHATSAPP_PHONE_NUMBER_ID=... WHATSAPP_ACCESS_TOKEN=... \
//	  WHATSAPP_WABA_ID=... DATABASE_URL=postgres://app@localhost:5432/app \
//	  go run ./cmd/checar-whatsapp
//
// Com -enviar, manda UMA mensagem de verdade para TELEFONE_TESTE, montada
// pelo mesmo código que a régua usa. É o único jeito de provar o caminho
// inteiro; por isso exige a flag e um número explícito.
//
//	TELEFONE_TESTE=34999998888 go run ./cmd/checar-whatsapp -enviar
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // registra o driver "pgx" pro database/sql

	"facilitaarcom/internal/disparo"
)

func main() {
	enviar := flag.Bool("enviar", false,
		"manda uma mensagem real para TELEFONE_TESTE usando o primeiro template aprovado")
	flag.Parse()

	r := &relatorio{}
	executar(context.Background(), r, *enviar)
	r.fechar()

	if r.falhas > 0 {
		os.Exit(1)
	}
}

// --- relatório ---

// relatorio acumula o resultado das checagens. Cada uma imprime na hora (a
// primeira é a mais lenta e é bom ver que está andando) e o total decide o
// código de saída, para o comando servir num script de implantação.
type relatorio struct {
	falhas  int
	avisos  int
	pulados int
}

func (r *relatorio) ok(titulo string, linhas ...string) { r.linha("OK    ", titulo, linhas) }

func (r *relatorio) falha(titulo string, linhas ...string) {
	r.falhas++
	r.linha("FALHA ", titulo, linhas)
}

func (r *relatorio) aviso(titulo string, linhas ...string) {
	r.avisos++
	r.linha("AVISO ", titulo, linhas)
}

func (r *relatorio) pulado(titulo string, linhas ...string) {
	r.pulados++
	r.linha("PULADO", titulo, linhas)
}

func (r *relatorio) linha(marca, titulo string, linhas []string) {
	fmt.Printf("[%s] %s\n", marca, titulo)
	for _, l := range linhas {
		fmt.Printf("           %s\n", l)
	}
}

func (r *relatorio) fechar() {
	fmt.Println()
	switch {
	case r.falhas > 0:
		fmt.Printf("%d falha(s), %d aviso(s), %d checagem(ns) pulada(s). "+
			"O disparo NÃO vai funcionar assim.\n", r.falhas, r.avisos, r.pulados)
	case r.pulados > 0:
		fmt.Printf("Nenhuma falha, %d aviso(s), mas %d checagem(ns) não rodaram — "+
			"o diagnóstico está incompleto.\n", r.avisos, r.pulados)
	default:
		fmt.Printf("Tudo conferido, %d aviso(s). O caminho de envio está de pé.\n", r.avisos)
	}
}

// --- checagens ---

func executar(ctx context.Context, r *relatorio, enviarDeVerdade bool) {
	idNumero := os.Getenv("WHATSAPP_PHONE_NUMBER_ID")
	token := os.Getenv("WHATSAPP_ACCESS_TOKEN")
	versao := disparo.VersaoAPIOu(os.Getenv("WHATSAPP_API_VERSION"))

	base := os.Getenv("WHATSAPP_BASE_URL")
	if base == "" {
		base = disparo.URLBaseMeta
	}
	base = strings.TrimRight(base, "/")

	// 1. Credenciais. Sem estas duas não há o que checar.
	var faltando []string
	if idNumero == "" {
		faltando = append(faltando, "WHATSAPP_PHONE_NUMBER_ID")
	}
	if token == "" {
		faltando = append(faltando, "WHATSAPP_ACCESS_TOKEN")
	}
	if len(faltando) > 0 {
		r.falha("credenciais de envio",
			"faltam: "+strings.Join(faltando, ", "),
			"o Phone Number ID está em developers.facebook.com > seu app > WhatsApp > Configuração da API")
		return
	}
	r.ok("credenciais de envio presentes", "versão da Graph API: "+versao)

	// As três checagens de webhook não dependem da Meta responder: são de
	// configuração, e a falta delas só apareceria quando um cliente
	// respondesse e ninguém visse.
	checarWebhook(r)

	cliente := &http.Client{Timeout: 20 * time.Second}

	// 2. O token vale, e vale para qual número.
	numero := checarNumero(ctx, r, cliente, base, versao, idNumero, token)

	// 3. Os templates que a régua espera existem, estão aprovados e têm a
	//    quantidade certa de marcadores.
	aprovado := checarTemplates(ctx, r, cliente, base, versao, token)

	// 4. Envio real, só sob pedido explícito.
	if enviarDeVerdade {
		enviarTeste(ctx, r, base, versao, idNumero, token, aprovado, numero)
	}
}

func checarWebhook(r *relatorio) {
	segredo := os.Getenv("WHATSAPP_APP_SECRET")
	verify := os.Getenv("WHATSAPP_VERIFY_TOKEN")
	appURL := os.Getenv("APP_URL")

	if segredo == "" || verify == "" {
		r.aviso("webhook desligado",
			"sem WHATSAPP_APP_SECRET e WHATSAPP_VERIFY_TOKEN a rota do webhook nem é montada",
			"consequência: dá para DISPARAR, mas resposta de cliente não chega, e a mesa fica vazia")
	} else {
		r.ok("webhook configurado",
			"cadastre na Meta: {APP_URL}/api/v1"+conversaCaminhoWebhook,
			"assine o campo `messages` — é o que traz a resposta do cliente")
	}

	switch u, err := url.Parse(appURL); {
	case appURL == "":
		r.aviso("APP_URL não definida",
			"o link de negociação sai como http://localhost:8080 e chega quebrado no cliente")
	case err != nil || u.Scheme != "https":
		r.aviso("APP_URL não é https: "+appURL,
			"a Meta só entrega webhook em https, e o link vai aberto no celular do cliente")
	default:
		r.ok("APP_URL pública: " + appURL)
	}
}

// conversaCaminhoWebhook repete o valor de conversa.CaminhoWebhook. Importar o
// pacote inteiro aqui puxaria gorm e o serviço de conversa para um comando que
// só faz chamadas HTTP; a constante é uma string estável e o teste de rotas
// guarda o caminho de verdade.
const conversaCaminhoWebhook = "/webhooks/whatsapp"

type respostaNumero struct {
	DisplayPhoneNumber     string `json:"display_phone_number"`
	VerifiedName           string `json:"verified_name"`
	QualityRating          string `json:"quality_rating"`
	CodeVerificationStatus string `json:"code_verification_status"`
	PlatformType           string `json:"platform_type"`
	Throughput             struct {
		Level string `json:"level"`
	} `json:"throughput"`
}

func checarNumero(ctx context.Context, r *relatorio, c *http.Client,
	base, versao, idNumero, token string) *respostaNumero {

	alvo := fmt.Sprintf("%s/%s/%s?fields=display_phone_number,verified_name,"+
		"quality_rating,code_verification_status,platform_type,throughput", base, versao, idNumero)

	var n respostaNumero
	if err := pegar(ctx, c, alvo, token, &n); err != nil {
		r.falha("a Meta recusou a consulta do número", explicar(err),
			"código 190 é token inválido ou expirado — o token temporário do painel dura 24h;",
			"para produção gere um token de Usuário do Sistema, que não expira")
		return nil
	}

	detalhes := []string{
		"número: " + n.DisplayPhoneNumber + "  (" + n.VerifiedName + ")",
		"qualidade: " + ouTraco(n.QualityRating) + "   verificação: " + ouTraco(n.CodeVerificationStatus),
	}
	if n.Throughput.Level != "" {
		detalhes = append(detalhes, "capacidade: "+n.Throughput.Level)
	}
	r.ok("token válido, e controla este número", detalhes...)
	r.aviso("confirme que o número acima é o de TESTE",
		"se for o número que hoje atende pela Nines, o disparo daqui sai para cliente real")

	// platform_type é o que separa "meu app fala com este número" de "este
	// número está registrado em outro lugar". Um número ainda em poder do BSP
	// não devolve CLOUD_API, e o envio falharia sem dizer por quê.
	if n.PlatformType != "" && n.PlatformType != "CLOUD_API" {
		r.falha("o número não está na Cloud API: "+n.PlatformType,
			"enquanto ele estiver registrado em outra plataforma (o BSP atual, por exemplo),",
			"este app não envia por ele — é preciso migrar o número ou usar outro para o teste")
	}
	return &n
}

type templateDaMeta struct {
	Nome       string `json:"name"`
	Status     string `json:"status"`
	Idioma     string `json:"language"`
	Categoria  string `json:"category"`
	Componente []struct {
		Tipo  string `json:"type"`
		Texto string `json:"text"`
	} `json:"components"`
}

type respostaTemplates struct {
	Dados  []templateDaMeta `json:"data"`
	Paging struct {
		Next string `json:"next"`
	} `json:"paging"`
}

// campanha é o que a régua espera do template, lido da tabela.
type campanha struct {
	nome string
	// texto é o template em português com nossos marcadores ({nome}, {valor},
	// ...). Não é o que viaja para a Meta — serve para mostrar na tela o que o
	// cliente vai ler, e é o mesmo texto que a régua grava no histórico.
	texto      string
	meta       string
	idioma     string
	parametros []string
	ativo      bool
}

var marcadorMeta = regexp.MustCompile(`\{\{\s*(\d+)\s*\}\}`)

func checarTemplates(ctx context.Context, r *relatorio, c *http.Client,
	base, versao, token string) *disparo.Mensagem {

	waba := os.Getenv("WHATSAPP_WABA_ID")
	campanhas, err := lerCampanhas(ctx)

	switch {
	case waba == "" && err != nil:
		r.pulado("conferência dos templates",
			"defina WHATSAPP_WABA_ID (o id da conta do WhatsApp Business, em Configurações "+
				"do Business > Contas do WhatsApp) e DATABASE_URL")
		return nil
	case waba == "":
		r.pulado("conferência dos templates",
			"defina WHATSAPP_WABA_ID — sem ele não dá para listar o que a Meta aprovou",
			"a régua espera: "+nomesEsperados(campanhas))
		return nil
	case err != nil:
		r.pulado("conferência dos templates",
			"não consegui ler a tabela campanhas: "+err.Error(),
			"defina DATABASE_URL para eu comparar o aprovado com o que a régua manda")
		return nil
	}

	alvo := fmt.Sprintf("%s/%s/%s/message_templates?limit=200", base, versao, waba)
	var resp respostaTemplates
	if err := pegar(ctx, c, alvo, token, &resp); err != nil {
		r.falha("a Meta recusou a lista de templates", explicar(err),
			"confira se WHATSAPP_WABA_ID é o id da conta do WhatsApp Business (não o do app, "+
				"nem o do número)")
		return nil
	}
	if resp.Paging.Next != "" {
		r.aviso("a Meta tem mais de 200 templates nesta conta",
			"só olhei os 200 primeiros; um template da régua pode ter ficado de fora")
	}

	porChave := map[string]templateDaMeta{}
	for _, t := range resp.Dados {
		porChave[t.Nome+"|"+t.Idioma] = t
	}

	var primeiroAprovado *disparo.Mensagem

	for _, cmp := range campanhas {
		if !cmp.ativo {
			continue
		}
		if cmp.meta == "" {
			r.falha("campanha "+cmp.nome+" sem template_meta",
				"a régua não tem o nome aprovado para mandar, e o disparo para nesta faixa")
			continue
		}

		t, achou := porChave[cmp.meta+"|"+cmp.idioma]
		if !achou {
			r.falha(fmt.Sprintf("template %q (%s) não existe nesta conta", cmp.meta, cmp.idioma),
				"campanha: "+cmp.nome,
				"submeta em Gerenciador do WhatsApp > Modelos de mensagem, categoria Utilidade,",
				"com "+fmt.Sprint(len(cmp.parametros))+" variáveis nesta ordem: "+
					strings.Join(cmp.parametros, ", "))
			continue
		}

		if t.Status != "APPROVED" {
			r.falha(fmt.Sprintf("template %q está %s", cmp.meta, t.Status),
				"campanha: "+cmp.nome,
				"enquanto não estiver APPROVED, todo disparo desta faixa é recusado (código 132001)")
			continue
		}

		// A categoria decide o preço. Utility custa cerca de um nono de
		// marketing no Brasil, e a Meta reclassifica sozinha lendo o texto.
		if t.Categoria != "" && t.Categoria != "UTILITY" {
			r.aviso(fmt.Sprintf("template %q foi aprovado como %s, não UTILITY", cmp.meta, t.Categoria),
				"marketing custa cerca de 9x utility por mensagem — reescreva o texto sem",
				"linguagem de oferta e peça reclassificação")
		}

		corpo := ""
		for _, comp := range t.Componente {
			if strings.EqualFold(comp.Tipo, "BODY") {
				corpo = comp.Texto
			}
		}
		esperados, aprovadosN := len(cmp.parametros), maiorMarcador(corpo)
		if esperados != aprovadosN {
			r.falha(fmt.Sprintf("template %q tem %d marcador(es), e a régua manda %d valor(es)",
				cmp.meta, aprovadosN, esperados),
				"campanha: "+cmp.nome,
				"a régua manda, nesta ordem: "+strings.Join(cmp.parametros, ", "),
				"a Meta recusa o envio inteiro com código 132000 quando a contagem não bate")
			continue
		}

		r.ok(fmt.Sprintf("template %q aprovado, %d marcadores, categoria %s",
			cmp.meta, aprovadosN, ouTraco(t.Categoria)),
			"campanha: "+cmp.nome)

		if primeiroAprovado == nil {
			m, err := montarExemplo(cmp)
			if err != nil {
				r.falha("campanha " + cmp.nome + ": " + err.Error())
				continue
			}
			primeiroAprovado = m
		}
	}

	return primeiroAprovado
}

// montarExemplo monta a mensagem pelo MESMO caminho da régua — ValoresNaOrdem
// sobre as variáveis da campanha. Um teste que montasse o payload à mão
// provaria que a Meta aceita alguma coisa, não que aceita a nossa.
func montarExemplo(cmp campanha) (*disparo.Mensagem, error) {
	v := disparo.Variaveis{
		Tratamento:   "Supermercado Boa Compra",
		NomeCompleto: "Supermercado Boa Compra LTDA",
		Contrato:     "CTR-TESTE-0001",
		Dias:         42,
		Valor:        84210.00,
		Link:         appURLOuPadrao() + "/negociar/token-de-teste",
	}
	valores, err := disparo.ValoresNaOrdem(cmp.parametros, v)
	if err != nil {
		return nil, err
	}
	return &disparo.Mensagem{
		Texto:      disparo.MontarMensagem(cmp.texto, v),
		Template:   cmp.meta,
		Idioma:     cmp.idioma,
		Parametros: valores,
	}, nil
}

func enviarTeste(ctx context.Context, r *relatorio, base, versao, idNumero, token string,
	exemplo *disparo.Mensagem, numero *respostaNumero) {

	if exemplo == nil {
		r.falha("envio de teste não rodou",
			"nenhum template aprovado para mandar — resolva as falhas acima primeiro")
		return
	}

	destino := disparo.TelefoneComDDI(os.Getenv("TELEFONE_TESTE"))
	if !disparo.TelefoneValido(destino) {
		r.falha("envio de teste não rodou",
			"defina TELEFONE_TESTE com o seu número (ex.: 34999998888)",
			"esta é a trava que impede o teste de alcançar um cliente")
		return
	}
	if numero != nil && somenteDigitos(numero.DisplayPhoneNumber) == destino {
		r.falha("envio de teste não rodou",
			"TELEFONE_TESTE é o próprio número remetente — a Meta recusa mandar para si mesmo")
		return
	}

	canal, err := disparo.NovoCanalMeta(disparo.ConfigMeta{
		IDNumero: idNumero, Token: token, VersaoAPI: versao, BaseURL: base,
	})
	if err != nil {
		r.falha("envio de teste não rodou", err.Error())
		return
	}

	m := *exemplo
	m.Telefone = destino

	wamid, err := canal.Enviar(ctx, m)
	if err != nil {
		r.falha("a Meta recusou o envio de teste", explicar(err),
			"template: "+m.Template+"  ("+m.Idioma+")")
		return
	}
	r.ok("mensagem entregue à Meta",
		"template: "+m.Template,
		"id da mensagem: "+wamid,
		"texto esperado: "+m.Texto,
		"confira o aparelho. Se chegou, o caminho inteiro funciona;",
		"responda a mensagem para abrir a janela de 24h e ver a conversa na mesa")
}

// --- acesso ao banco ---

func lerCampanhas(ctx context.Context) ([]campanha, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, errors.New("DATABASE_URL não definida")
	}

	bd, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	defer bd.Close()

	linhas, err := bd.QueryContext(ctx, `
		SELECT nome, template, coalesce(template_meta, ''), idioma, parametros, ativo
		  FROM campanhas ORDER BY faixa_min`)
	if err != nil {
		return nil, err
	}
	defer linhas.Close()

	var todas []campanha
	for linhas.Next() {
		var c campanha
		var params []byte
		if err := linhas.Scan(&c.nome, &c.texto, &c.meta, &c.idioma, &params, &c.ativo); err != nil {
			return nil, err
		}
		if len(params) > 0 {
			if err := json.Unmarshal(params, &c.parametros); err != nil {
				return nil, fmt.Errorf("campanha %s: parâmetros ilegíveis: %w", c.nome, err)
			}
		}
		todas = append(todas, c)
	}
	if err := linhas.Err(); err != nil {
		return nil, err
	}
	if len(todas) == 0 {
		return nil, errors.New("nenhuma campanha na tabela — rodou as migrations?")
	}
	return todas, nil
}

// --- utilidades ---

// pegar faz um GET autenticado na Graph API. O token vai no cabeçalho, nunca
// na URL: URL entra em log de proxy e em histórico de shell.
func pegar(ctx context.Context, c *http.Client, alvo, token string, destino any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, alvo, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	corpo, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		var falha struct {
			Erro struct {
				Message string `json:"message"`
				Code    int    `json:"code"`
			} `json:"error"`
		}
		_ = json.Unmarshal(corpo, &falha)
		return fmt.Errorf("http %d, código %d: %s",
			resp.StatusCode, falha.Erro.Code, falha.Erro.Message)
	}
	return json.Unmarshal(corpo, destino)
}

// explicar mostra o erro da Meta sem deixar escapar o token: ErroMeta já
// guarda só status, código e mensagem, e a URL nunca entra na mensagem.
func explicar(err error) string {
	var meta *disparo.ErroMeta
	if errors.As(err, &meta) {
		return fmt.Sprintf("http %d, código %d: %s", meta.Status, meta.Codigo, meta.Mensagem)
	}
	return err.Error()
}

// maiorMarcador devolve o maior {{n}} do corpo, e não a contagem: um template
// que usa {{1}} duas vezes tem dois marcadores e um só parâmetro.
func maiorMarcador(corpo string) int {
	maior := 0
	for _, achado := range marcadorMeta.FindAllStringSubmatch(corpo, -1) {
		var n int
		if _, err := fmt.Sscanf(achado[1], "%d", &n); err == nil && n > maior {
			maior = n
		}
	}
	return maior
}

func nomesEsperados(campanhas []campanha) string {
	if len(campanhas) == 0 {
		return "(não consegui ler a tabela campanhas)"
	}
	var nomes []string
	for _, c := range campanhas {
		if c.meta != "" {
			nomes = append(nomes, c.meta)
		}
	}
	return strings.Join(nomes, ", ")
}

func appURLOuPadrao() string {
	if v := os.Getenv("APP_URL"); v != "" {
		return strings.TrimRight(v, "/")
	}
	return "http://localhost:8080"
}

var naoDigito = regexp.MustCompile(`\D`)

func somenteDigitos(s string) string { return naoDigito.ReplaceAllString(s, "") }

func ouTraco(v string) string {
	if v == "" {
		return "—"
	}
	return v
}
