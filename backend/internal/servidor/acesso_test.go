package servidor_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/pressly/goose/v3"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	_ "github.com/jackc/pgx/v5/stdlib"

	"facilitaarcom/internal/acesso"
	"facilitaarcom/internal/config"
	"facilitaarcom/internal/db"
	"facilitaarcom/internal/servidor"
)

// Estes testes precisam de um Postgres de verdade — as regras que importam
// (índice único de e-mail, sessão que morre ao desativar o usuário) moram no
// banco, e um fake não as exerceria. Sem DATABASE_URL_TESTE o pacote pula,
// para `go test ./...` continuar rodando em máquina sem banco.
func bancoDeTeste(t *testing.T) *gorm.DB {
	t.Helper()

	url := os.Getenv("DATABASE_URL_TESTE")
	if url == "" {
		t.Skip("DATABASE_URL_TESTE não definida — pulando testes que precisam de Postgres")
	}

	sqlDB, err := sql.Open("pgx", url)
	if err != nil {
		t.Fatalf("abrir postgres de teste: %v", err)
	}

	// Uma conexão só: o lock consultivo abaixo é por conexão, e com um pool
	// o driver poderia entregar outra conexão (sem o lock) para a consulta
	// seguinte.
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })

	// Estes testes limpam as tabelas e repovoam. Duas execuções ao mesmo
	// tempo no mesmo banco — dois terminais, ou uma execução do pacote junto
	// com um `go test ./...` — truncariam os dados uma da outra, e a falha
	// aparece como "duplicated key" ou violação de chave estrangeira em
	// testes que não têm nada a ver com isso.
	//
	// O lock consultivo serializa: a segunda execução espera a vez em vez de
	// corromper. É liberado quando a conexão fecha, no Cleanup acima.
	if _, err := sqlDB.Exec(`SELECT pg_advisory_lock(875123401)`); err != nil {
		t.Fatalf("obter lock do banco de teste: %v", err)
	}

	goose.SetBaseFS(db.Migrations)
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatalf("dialeto do goose: %v", err)
	}
	goose.SetLogger(goose.NopLogger())
	if err := goose.Up(sqlDB, "migrations"); err != nil {
		t.Fatalf("aplicar migrations: %v", err)
	}

	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("abrir gorm: %v", err)
	}

	// bcrypt cost 12 é o de produção; aqui ele só faria a suíte demorar (sob
	// o detector de corrida, minutos por teste). O que estes testes exercitam
	// é o fluxo de sessão, não a força do hash.
	t.Cleanup(acesso.UsarCustoDeHashReduzido())

	// Cada teste começa com as tabelas de gente vazias. RESTART IDENTITY
	// CASCADE limpa também o que referencia usuários.
	if err := gdb.Exec("TRUNCATE usuarios, sessoes, clientes, dividas, disparos, acordos CASCADE").Error; err != nil {
		t.Fatalf("limpar tabelas: %v", err)
	}
	return gdb
}

func montarServidor(t *testing.T) (http.Handler, *gorm.DB) {
	t.Helper()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := &config.Config{Env: "development", Port: "3000", AppURL: "https://facilita.arcom.com.br"}
	gdb := bancoDeTeste(t)
	return servidor.Novo(cfg, log, gdb, nil), gdb
}

func servidorComBanco(t *testing.T) http.Handler {
	t.Helper()
	h, _ := montarServidor(t)
	return h
}

func chamar(t *testing.T, h http.Handler, metodo, caminho string, corpo any, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()

	var leitor io.Reader
	if corpo != nil {
		bruto, err := json.Marshal(corpo)
		if err != nil {
			t.Fatalf("serializar corpo: %v", err)
		}
		leitor = bytes.NewReader(bruto)
	}

	req := httptest.NewRequest(metodo, caminho, leitor)
	if corpo != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for _, c := range cookies {
		req.AddCookie(c)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func cookieDeSessao(t *testing.T, rec *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, c := range (&http.Response{Header: rec.Header()}).Cookies() {
		if c.Name == "sessao" {
			return c
		}
	}
	t.Fatal("resposta não trouxe o cookie de sessão")
	return nil
}

const (
	emailGerencia = "gerencia@arcom.com.br"
	senhaGerencia = "SenhaForte2026!"
)

// criarGerencia usa o primeiro acesso — a única rota que cria usuário sem
// sessão, e só enquanto a tabela está vazia.
func criarGerencia(t *testing.T, h http.Handler) {
	t.Helper()
	rec := chamar(t, h, http.MethodPost, "/api/v1/sessao/primeiro-acesso", map[string]string{
		"nome": "Gerência", "email": emailGerencia, "senha": senhaGerencia,
		"papel": "gerencia", "equipe": "interno",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("primeiro acesso: status %d (%s)", rec.Code, rec.Body.String())
	}
}

func logar(t *testing.T, h http.Handler, email, senha string) *http.Cookie {
	t.Helper()
	rec := chamar(t, h, http.MethodPost, "/api/v1/sessao", map[string]string{"email": email, "senha": senha})
	if rec.Code != http.StatusOK {
		t.Fatalf("login: status %d (%s)", rec.Code, rec.Body.String())
	}
	return cookieDeSessao(t, rec)
}

func TestLoginDefineCookieHttpOnlyESemTokenNoCorpo(t *testing.T) {
	h := servidorComBanco(t)
	criarGerencia(t, h)

	rec := chamar(t, h, http.MethodPost, "/api/v1/sessao", map[string]string{
		"email": emailGerencia, "senha": senhaGerencia,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, quer 200 (%s)", rec.Code, rec.Body.String())
	}

	c := cookieDeSessao(t, rec)
	if !c.HttpOnly {
		t.Error("cookie de sessão precisa ser HttpOnly — senão o JS da página lê o token")
	}
	if c.SameSite != http.SameSiteLaxMode {
		t.Errorf("SameSite = %v, quer Lax", c.SameSite)
	}
	if c.Path != "/api" {
		t.Errorf("Path = %q, quer /api", c.Path)
	}
	if c.Value == "" {
		t.Fatal("cookie veio sem valor")
	}

	// O token nunca pode aparecer no corpo: o frontend não deve ter como
	// guardá-lo em localStorage nem por acidente.
	if bytes.Contains(rec.Body.Bytes(), []byte(c.Value)) {
		t.Error("o token de sessão vazou no corpo da resposta de login")
	}
	// E nem o hash da senha.
	if bytes.Contains(rec.Body.Bytes(), []byte("senha")) {
		t.Errorf("resposta de login menciona senha: %s", rec.Body.String())
	}
}

func TestLoginRecusaSenhaErradaComMensagemGenerica(t *testing.T) {
	h := servidorComBanco(t)
	criarGerencia(t, h)

	casos := []struct {
		nome  string
		email string
		senha string
	}{
		{"senha errada", emailGerencia, "OutraSenhaQualquer1"},
		{"e-mail inexistente", "ninguem@arcom.com.br", senhaGerencia},
	}

	var mensagens []string
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			rec := chamar(t, h, http.MethodPost, "/api/v1/sessao", map[string]string{"email": c.email, "senha": c.senha})
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, quer 401 (%s)", rec.Code, rec.Body.String())
			}
			var p struct {
				Detail string `json:"detail"`
				Codigo string `json:"codigo"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
				t.Fatalf("corpo inesperado: %s", rec.Body.String())
			}
			mensagens = append(mensagens, p.Detail)
		})
	}

	// As duas respostas têm que ser idênticas — se diferirem, dá pra
	// descobrir quais e-mails têm conta na empresa.
	if len(mensagens) == 2 && mensagens[0] != mensagens[1] {
		t.Errorf("mensagens diferentes entre senha errada e e-mail inexistente: %q vs %q", mensagens[0], mensagens[1])
	}
}

func TestRotaPrivadaExigeSessao(t *testing.T) {
	h := servidorComBanco(t)
	criarGerencia(t, h)

	rec := chamar(t, h, http.MethodGet, "/api/v1/usuarios", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("sem cookie: status = %d, quer 401 (%s)", rec.Code, rec.Body.String())
	}

	cookie := logar(t, h, emailGerencia, senhaGerencia)
	rec = chamar(t, h, http.MethodGet, "/api/v1/usuarios", nil, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("com cookie: status = %d, quer 200 (%s)", rec.Code, rec.Body.String())
	}
}

func TestCookieForjadoNaoAutentica(t *testing.T) {
	h := servidorComBanco(t)
	criarGerencia(t, h)

	falso := &http.Cookie{Name: "sessao", Value: "token-inventado-que-nao-existe-no-banco"}
	rec := chamar(t, h, http.MethodGet, "/api/v1/usuarios", nil, falso)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, quer 401", rec.Code)
	}
}

func TestLogoutMataASessao(t *testing.T) {
	h := servidorComBanco(t)
	criarGerencia(t, h)
	cookie := logar(t, h, emailGerencia, senhaGerencia)

	rec := chamar(t, h, http.MethodPost, "/api/v1/sessao/logout", nil, cookie)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("logout: status = %d, quer 204 (%s)", rec.Code, rec.Body.String())
	}

	// O mesmo cookie não pode mais servir: a sessão saiu do banco, não só do
	// navegador.
	rec = chamar(t, h, http.MethodGet, "/api/v1/usuarios", nil, cookie)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("depois do logout: status = %d, quer 401", rec.Code)
	}
}

func TestPrimeiroAcessoSoFuncionaUmaVez(t *testing.T) {
	h := servidorComBanco(t)
	criarGerencia(t, h)

	rec := chamar(t, h, http.MethodPost, "/api/v1/sessao/primeiro-acesso", map[string]string{
		"nome": "Intruso", "email": "intruso@arcom.com.br", "senha": "SenhaForte2026!",
		"papel": "gerencia", "equipe": "interno",
	})
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, quer 409 — primeiro acesso não pode criar uma segunda gerência sem sessão (%s)", rec.Code, rec.Body.String())
	}
}

func TestAprendizNaoCriaUsuario(t *testing.T) {
	h := servidorComBanco(t)
	criarGerencia(t, h)
	cookieGerencia := logar(t, h, emailGerencia, senhaGerencia)

	const emailAprendiz, senhaAprendiz = "aprendiz@arcom.com.br", "SenhaForte2026!"
	rec := chamar(t, h, http.MethodPost, "/api/v1/usuarios", map[string]string{
		"nome": "Aprendiz", "email": emailAprendiz, "senha": senhaAprendiz,
		"papel": "aprendiz", "equipe": "interno",
	}, cookieGerencia)
	if rec.Code != http.StatusCreated {
		t.Fatalf("gerência criando aprendiz: status = %d (%s)", rec.Code, rec.Body.String())
	}

	cookieAprendiz := logar(t, h, emailAprendiz, senhaAprendiz)
	rec = chamar(t, h, http.MethodPost, "/api/v1/usuarios", map[string]string{
		"nome": "Outro", "email": "outro@arcom.com.br", "senha": "SenhaForte2026!",
		"papel": "analista", "equipe": "interno",
	}, cookieAprendiz)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("aprendiz criando usuário: status = %d, quer 403 (%s)", rec.Code, rec.Body.String())
	}
}

func TestDesativarUsuarioDerrubaSessaoNaHora(t *testing.T) {
	h := servidorComBanco(t)
	criarGerencia(t, h)
	cookieGerencia := logar(t, h, emailGerencia, senhaGerencia)

	const emailAnalista, senhaAnalista = "analista@arcom.com.br", "SenhaForte2026!"
	rec := chamar(t, h, http.MethodPost, "/api/v1/usuarios", map[string]string{
		"nome": "Analista", "email": emailAnalista, "senha": senhaAnalista,
		"papel": "analista", "equipe": "interno",
	}, cookieGerencia)
	if rec.Code != http.StatusCreated {
		t.Fatalf("criar analista: status = %d (%s)", rec.Code, rec.Body.String())
	}
	var criado struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &criado); err != nil {
		t.Fatalf("resposta de criação inesperada: %s", rec.Body.String())
	}

	cookieAnalista := logar(t, h, emailAnalista, senhaAnalista)
	if rec := chamar(t, h, http.MethodGet, "/api/v1/usuarios", nil, cookieAnalista); rec.Code != http.StatusOK {
		t.Fatalf("analista logado deveria listar: status = %d", rec.Code)
	}

	inativo := false
	rec = chamar(t, h, http.MethodPatch, "/api/v1/usuarios/"+criado.ID, map[string]any{"ativo": &inativo}, cookieGerencia)
	if rec.Code != http.StatusOK {
		t.Fatalf("desativar: status = %d (%s)", rec.Code, rec.Body.String())
	}

	// Sem isso, alguém desligado hoje continuaria dentro do sistema até o
	// cookie vencer sozinho.
	if rec := chamar(t, h, http.MethodGet, "/api/v1/usuarios", nil, cookieAnalista); rec.Code != http.StatusUnauthorized {
		t.Fatalf("depois de desativado: status = %d, quer 401", rec.Code)
	}
	if rec := chamar(t, h, http.MethodPost, "/api/v1/sessao", map[string]string{"email": emailAnalista, "senha": senhaAnalista}); rec.Code != http.StatusUnauthorized {
		t.Fatalf("login de usuário desativado: status = %d, quer 401", rec.Code)
	}
}

func TestNaoDaPraAumentarAPropriaAlcada(t *testing.T) {
	h := servidorComBanco(t)
	criarGerencia(t, h)
	cookie := logar(t, h, emailGerencia, senhaGerencia)

	rec := chamar(t, h, http.MethodGet, "/api/v1/sessao", nil, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("quem sou eu: status = %d (%s)", rec.Code, rec.Body.String())
	}
	var eu struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &eu); err != nil {
		t.Fatalf("resposta inesperada: %s", rec.Body.String())
	}

	alcada := 100.0
	rec = chamar(t, h, http.MethodPatch, "/api/v1/usuarios/"+eu.ID, map[string]any{"alcadaMaxima": &alcada}, cookie)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, quer 403 — ninguém mexe no próprio privilégio (%s)", rec.Code, rec.Body.String())
	}
}

func TestCoordenacaoNaoPromoveParaGerencia(t *testing.T) {
	h := servidorComBanco(t)
	criarGerencia(t, h)
	cookieGerencia := logar(t, h, emailGerencia, senhaGerencia)
	cookieCoord := criarOperador(t, h, cookieGerencia, "coord@arcom.com.br", "coordenacao", "")

	const emailAnalista, senhaAnalista = "analista2@arcom.com.br", "SenhaForte2026!"
	rec := chamar(t, h, http.MethodPost, "/api/v1/usuarios", map[string]string{
		"nome": "Analista", "email": emailAnalista, "senha": senhaAnalista,
		"papel": "analista", "equipe": "interno",
	}, cookieGerencia)
	if rec.Code != http.StatusCreated {
		t.Fatalf("criar analista: status = %d (%s)", rec.Code, rec.Body.String())
	}
	var criado struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &criado); err != nil {
		t.Fatalf("resposta inesperada: %s", rec.Body.String())
	}

	// Coordenação não promove ninguém acima do próprio papel.
	papel := "gerencia"
	rec = chamar(t, h, http.MethodPatch, "/api/v1/usuarios/"+criado.ID, map[string]any{"papel": &papel}, cookieCoord)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("promover para gerência: status = %d, quer 422 (%s)", rec.Code, rec.Body.String())
	}
}

func TestAlcadaConcedidaNaoPassaDaDeQuemConcede(t *testing.T) {
	h := servidorComBanco(t)
	criarGerencia(t, h)
	cookieGerencia := logar(t, h, emailGerencia, senhaGerencia)
	// Alçada padrão de coordenação é 100, então reduz pra expor o teto.
	cookieCoord := criarOperador(t, h, cookieGerencia, "coord2@arcom.com.br", "coordenacao", "")

	rec := chamar(t, h, http.MethodGet, "/api/v1/sessao", nil, cookieCoord)
	var coord struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &coord); err != nil {
		t.Fatalf("resposta inesperada: %s", rec.Body.String())
	}
	tetoReduzido := 40.0
	rec = chamar(t, h, http.MethodPatch, "/api/v1/usuarios/"+coord.ID, map[string]any{"alcadaMaxima": &tetoReduzido}, cookieGerencia)
	if rec.Code != http.StatusOK {
		t.Fatalf("reduzir teto da coordenação: status = %d (%s)", rec.Code, rec.Body.String())
	}
	cookieCoord = logar(t, h, "coord2@arcom.com.br", "SenhaForte2026!")

	rec = chamar(t, h, http.MethodPost, "/api/v1/usuarios", map[string]string{
		"nome": "Analista3", "email": "analista3@arcom.com.br", "senha": "SenhaForte2026!",
		"papel": "analista", "equipe": "interno",
	}, cookieCoord)
	var criado struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &criado); err != nil {
		t.Fatalf("resposta inesperada: %s", rec.Body.String())
	}

	acima := 60.0
	rec = chamar(t, h, http.MethodPatch, "/api/v1/usuarios/"+criado.ID, map[string]any{"alcadaMaxima": &acima}, cookieCoord)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("conceder alçada acima da própria: status = %d, quer 422 (%s)", rec.Code, rec.Body.String())
	}
}

func TestCriarUsuarioComPapelOuEquipeInvalidaDevolve422(t *testing.T) {
	h := servidorComBanco(t)
	criarGerencia(t, h)
	cookie := logar(t, h, emailGerencia, senhaGerencia)

	casos := []map[string]string{
		{"papel": "diretor", "equipe": "interno"},
		{"papel": "analista", "equipe": "matriz"},
	}
	for _, extra := range casos {
		corpo := map[string]string{"nome": "X", "email": "x@arcom.com.br", "senha": "SenhaForte2026!"}
		for k, v := range extra {
			corpo[k] = v
		}
		rec := chamar(t, h, http.MethodPost, "/api/v1/usuarios", corpo, cookie)
		if rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("%v: status = %d, quer 422 (%s)", extra, rec.Code, rec.Body.String())
		}
	}
}

// A rota de login é a que mais sofre força bruta — por isso tem rate limit
// próprio, bem mais apertado que o global (10/min por IP, contra 1200/min).
func TestLoginTemRateLimitPorIP(t *testing.T) {
	h := servidorComBanco(t)
	criarGerencia(t, h)

	corpo := map[string]string{"email": emailGerencia, "senha": "SenhaErrada1"}
	var bloqueouEm int
	for i := 1; i <= 10; i++ {
		rec := chamar(t, h, http.MethodPost, "/api/v1/sessao", corpo)
		if rec.Code == http.StatusTooManyRequests {
			bloqueouEm = i
			break
		}
	}
	if bloqueouEm == 0 {
		t.Fatal("10 tentativas em um minuto e nenhuma foi bloqueada")
	}
}
