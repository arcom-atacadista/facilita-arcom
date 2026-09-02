package acesso

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// CustoBcryptPadrao é o custo de produção: 12 em vez do default do pacote
// (10), exigência de system-design/padroes/03-backend.md.
const CustoBcryptPadrao = 12

// custoBcrypt é variável, e não constante, só por causa dos testes: com custo
// 12 cada hash leva ~250ms, e sob o detector de corrida passa de vários
// segundos — a suíte de integração inteira levaria mais de dez minutos, o que
// faria ninguém rodar `go test -race`. Produção nunca mexe nisto; quem baixa
// é UsarCustoDeHashReduzido, chamado apenas de teste.
var custoBcrypt = CustoBcryptPadrao

// UsarCustoDeHashReduzido baixa o custo do bcrypt e devolve a função que
// restaura o valor de produção. Existe exclusivamente para os testes de
// integração; num binário servindo tráfego ninguém a chama.
func UsarCustoDeHashReduzido() func() {
	custoBcrypt = bcrypt.MinCost
	return func() { custoBcrypt = CustoBcryptPadrao }
}

const (
	// 32 bytes de crypto/rand = 256 bits de entropia, bem acima do piso de
	// 128 bits que a skill de autenticação pede pra token opaco.
	bytesToken = 32

	// Duração da sessão. Backoffice interno usado o dia inteiro: 12h cobre
	// um turno sem obrigar relogin no meio do atendimento.
	DuracaoSessao = 12 * time.Hour
)

// hashDescarte é um hash bcrypt válido de uma senha que ninguém usa. Serve
// para gastar o mesmo tempo de CPU quando o e-mail não existe: sem isso, o
// login responde na hora para e-mail inexistente e devagar para e-mail real,
// e essa diferença de tempo entrega quem tem conta na empresa.
//
// O custo embutido aqui (12) tem que ser o mesmo de custoBcrypt — um hash de
// custo menor gastaria menos tempo e a defesa deixaria de valer.
var hashDescarte = []byte("$2a$12$C6UzMDM.H6dfI/f/IKcEe.Ke5FbxdVJDdVQEP0Ck6Ub4H0Sc8/PGm")

// ErrCredenciaisInvalidas é único de propósito: o handler nunca diz se errou
// o e-mail ou a senha, nem se a conta está inativa.
var ErrCredenciaisInvalidas = errors.New("credenciais inválidas")

type Service struct{ repo *Repo }

func NovoService(repo *Repo) *Service { return &Service{repo: repo} }

// hashDoToken é o que vai pro banco. SHA-256 (e não bcrypt) porque o token
// já é aleatório de 256 bits — não há o que forçar por dicionário, e a
// verificação acontece em toda requisição autenticada.
// hashDeDescarte acompanha o custo vigente. Com o custo reduzido dos testes,
// o hash fixo de custo 12 tornaria o caminho do e-mail inexistente MUITO mais
// lento que o do e-mail real — a diferença de tempo que a defesa existe para
// eliminar, invertida.
func hashDeDescarte() []byte {
	if custoBcrypt == CustoBcryptPadrao {
		return hashDescarte
	}
	h, err := bcrypt.GenerateFromPassword([]byte("descarte"), custoBcrypt)
	if err != nil {
		return hashDescarte
	}
	return h
}

func hashDoToken(token string) []byte {
	soma := sha256.Sum256([]byte(token))
	return soma[:]
}

func gerarToken() (string, error) {
	bruto := make([]byte, bytesToken)
	if _, err := rand.Read(bruto); err != nil {
		return "", fmt.Errorf("gerar token de sessão: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(bruto), nil
}

// Autenticar valida credencial e abre uma sessão. Devolve o token em claro —
// é a única vez que ele existe fora do cookie.
func (s *Service) Autenticar(ctx context.Context, entrada EntradaLogin) (Usuario, string, error) {
	u, err := s.repo.PorEmail(ctx, entrada.Email)
	if err != nil {
		if errors.Is(err, ErrNaoEncontrado) {
			// Gasta o mesmo tempo do caminho feliz antes de recusar.
			_ = bcrypt.CompareHashAndPassword(hashDeDescarte(), []byte(entrada.Senha))
			return Usuario{}, "", ErrCredenciaisInvalidas
		}
		return Usuario{}, "", err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.SenhaHash), []byte(entrada.Senha)); err != nil {
		return Usuario{}, "", ErrCredenciaisInvalidas
	}

	// Conta desativada cai no mesmo erro genérico: quem tenta entrar não
	// precisa saber que o e-mail existe mas foi bloqueado.
	if !u.Ativo {
		return Usuario{}, "", ErrCredenciaisInvalidas
	}

	token, err := gerarToken()
	if err != nil {
		return Usuario{}, "", err
	}

	agora := time.Now().UTC()
	sessao := Sessao{
		ID:          uuid.New(),
		UsuarioID:   u.ID,
		TokenHash:   hashDoToken(token),
		CriadaEm:    agora,
		ExpiraEm:    agora.Add(DuracaoSessao),
		UltimoUsoEm: agora,
	}
	if err := s.repo.CriarSessao(ctx, &sessao); err != nil {
		return Usuario{}, "", fmt.Errorf("criar sessão: %w", err)
	}

	return u, token, nil
}

// UsuarioDaSessao resolve o cookie em usuário. Erro aqui é sempre "sessão não
// vale" — quem chama traduz para 401.
func (s *Service) UsuarioDaSessao(ctx context.Context, token string) (Usuario, error) {
	if token == "" {
		return Usuario{}, ErrNaoEncontrado
	}

	u, sessao, err := s.repo.UsuarioPorTokenHash(ctx, hashDoToken(token))
	if err != nil {
		return Usuario{}, err
	}

	// Registro de uso é conveniência de auditoria, não parte do contrato:
	// falhar aqui não pode derrubar uma requisição que já está autenticada.
	_ = s.repo.TocarSessao(ctx, sessao.ID)

	return u, nil
}

func (s *Service) Encerrar(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.repo.ApagarSessaoPorHash(ctx, hashDoToken(token))
}

func (s *Service) LimparSessoesExpiradas(ctx context.Context) (int64, error) {
	return s.repo.LimparSessoesExpiradas(ctx)
}

// Criar cadastra um operador. Não existe cadastro público: quem chama é
// sempre a coordenação/gerência autenticada — exceto o primeiro usuário do
// sistema, tratado por PrimeiroAcesso.
func (s *Service) Criar(ctx context.Context, entrada EntradaCriarUsuario) (Usuario, error) {
	if !entrada.Papel.Valido() {
		return Usuario{}, ErroCampo("papel", "Papel inválido.")
	}
	if !entrada.Equipe.Valido() {
		return Usuario{}, ErroCampo("equipe", "Equipe inválida.")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(entrada.Senha), custoBcrypt)
	if err != nil {
		return Usuario{}, fmt.Errorf("gerar hash da senha: %w", err)
	}

	agora := time.Now().UTC()
	u := Usuario{
		ID:             uuid.New(),
		Nome:           strings.TrimSpace(entrada.Nome),
		Email:          strings.ToLower(strings.TrimSpace(entrada.Email)),
		SenhaHash:      string(hash),
		Papel:          entrada.Papel,
		Equipe:         entrada.Equipe,
		Ativo:          true,
		AlcadaMaxima:   AlcadaPadrao[entrada.Papel],
		CodigoCobranca: entrada.CodigoCobranca,
		CriadoEm:       agora,
		AtualizadoEm:   agora,
	}

	if err := s.repo.Criar(ctx, &u); err != nil {
		if EhDuplicado(err) {
			return Usuario{}, ErrEmailEmUso
		}
		return Usuario{}, err
	}
	return u, nil
}

// ErrEmailEmUso vira 409 no handler.
var ErrEmailEmUso = errors.New("e-mail já cadastrado")

// PrimeiroAcesso cria o usuário inicial como gerência, e só funciona
// enquanto a tabela está vazia. É a única rota de criação sem sessão.
func (s *Service) PrimeiroAcesso(ctx context.Context, entrada EntradaCriarUsuario) (Usuario, error) {
	n, err := s.repo.Contar(ctx)
	if err != nil {
		return Usuario{}, err
	}
	if n > 0 {
		return Usuario{}, ErrJaInicializado
	}

	entrada.Papel = PapelGerencia
	if !entrada.Equipe.Valido() {
		entrada.Equipe = EquipeInterno
	}
	return s.Criar(ctx, entrada)
}

// ErrJaInicializado vira 409: já existe gente cadastrada, o primeiro acesso
// não pode ser usado pra criar mais uma gerência sem sessão.
var ErrJaInicializado = errors.New("sistema já inicializado")

func (s *Service) Listar(ctx context.Context) ([]Usuario, error) { return s.repo.Listar(ctx) }

func (s *Service) PorID(ctx context.Context, id uuid.UUID) (Usuario, error) {
	return s.repo.PorID(ctx, id)
}

// Atualizar aplica as mudanças que a coordenação pode fazer. quemPede é o
// usuário autenticado: ninguém aumenta a própria alçada nem se rebaixa/promove
// sozinho.
func (s *Service) Atualizar(ctx context.Context, quemPede Usuario, id uuid.UUID, entrada EntradaAtualizarUsuario) (Usuario, error) {
	u, err := s.repo.PorID(ctx, id)
	if err != nil {
		return Usuario{}, err
	}

	mexeEmPrivilegio := entrada.Papel != nil || entrada.AlcadaMaxima != nil || entrada.Ativo != nil
	if mexeEmPrivilegio && quemPede.ID == u.ID {
		return Usuario{}, ErrAutoPrivilegio
	}

	if entrada.Nome != nil {
		u.Nome = strings.TrimSpace(*entrada.Nome)
	}
	if entrada.Equipe != nil {
		if !entrada.Equipe.Valido() {
			return Usuario{}, ErroCampo("equipe", "Equipe inválida.")
		}
		u.Equipe = *entrada.Equipe
	}
	if entrada.Papel != nil {
		if !entrada.Papel.Valido() {
			return Usuario{}, ErroCampo("papel", "Papel inválido.")
		}
		// Ninguém cria alguém acima de si: coordenação não promove a gerência.
		if entrada.Papel.Nivel() > quemPede.Papel.Nivel() {
			return Usuario{}, ErrPapelAcimaDoSeu
		}
		u.Papel = *entrada.Papel
		u.AlcadaMaxima = AlcadaPadrao[*entrada.Papel]
	}
	if entrada.AlcadaMaxima != nil {
		// A alçada concedida nunca passa da de quem concede.
		if *entrada.AlcadaMaxima > quemPede.AlcadaMaxima {
			return Usuario{}, ErroCampo("alcadaMaxima",
				fmt.Sprintf("Você não pode conceder alçada acima da sua (%.0f%%).", quemPede.AlcadaMaxima))
		}
		u.AlcadaMaxima = *entrada.AlcadaMaxima
	}
	if entrada.Ativo != nil {
		u.Ativo = *entrada.Ativo
	}
	if entrada.CodigoCobranca != nil {
		u.CodigoCobranca = entrada.CodigoCobranca
	}

	if err := s.repo.Salvar(ctx, &u); err != nil {
		return Usuario{}, err
	}

	// Desativar ou rebaixar tem que valer agora, não quando o cookie vencer.
	if (entrada.Ativo != nil && !*entrada.Ativo) || entrada.Papel != nil || entrada.AlcadaMaxima != nil {
		_ = s.repo.ApagarSessoesDoUsuario(ctx, u.ID)
	}

	return u, nil
}

var (
	ErrAutoPrivilegio  = errors.New("não é possível alterar o próprio privilégio")
	ErrPapelAcimaDoSeu = errors.New("papel acima do seu")
)

// ErroDeCampo carrega um erro de validação de um campo só, para o handler
// montar o 422 do contrato sem cada service conhecer HTTP.
type ErroDeCampo struct {
	Campo    string
	Mensagem string
}

func (e *ErroDeCampo) Error() string { return e.Campo + ": " + e.Mensagem }

func ErroCampo(campo, mensagem string) error { return &ErroDeCampo{Campo: campo, Mensagem: mensagem} }
