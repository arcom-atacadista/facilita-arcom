# ARCOM Starter Project

**Kit padrão da ARCOM para qualquer pessoa criar um app, mesmo sem saber programar.**

Este repositório reúne tudo que você precisa pra pedir a um assistente de IA
(o Claude) que construa um projeto pra você — um site, um painel interno, uma
calculadora, um formulário — já com a identidade visual da ARCOM, dentro das
regras técnicas da empresa e respeitando as políticas de uso de IA.

Você não precisa entender de código pra usar isso. Este guia explica, em
português simples, o que tem aqui dentro e o passo a passo pra criar o seu
projeto.

---

## 1. O que tem dentro deste repositório

O repositório tem três partes. Você não precisa ler tudo — só precisar saber
onde procurar cada coisa.

| Pasta | O que é | Quando você usa |
|---|---|---|
| `design/` | Identidade visual da ARCOM: cores, logo, fontes, tom de voz | Quando quiser entender ou consultar a marca |
| `specs/` | As regras técnicas que a IA segue para construir o projeto | Você não precisa ler linha por linha — é o "manual de instruções" que o Claude usa nos bastidores |
| `privacy/` | Termos de uso, privacidade e o guia de boas práticas de IA da ARCOM | Leitura obrigatória antes de usar IA com dados da empresa |

Dentro de `specs/`, o arquivo mais importante pra você é o
`specs/LEIA-PRIMEIRO.md` — ele explica a estrutura completa em mais detalhe.
Mas pra começar, os únicos dois arquivos que você realmente precisa **copiar**
são:

- `specs/INSTRUCOES-DO-PROJETO.md`
- Todos os arquivos da pasta `specs/` (você vai enviá-los, não precisa ler)

---

## 2. Antes de começar: os termos de uso

Antes de criar qualquer projeto, leia o conteúdo da pasta `privacy/`. Ela
reúne as regras da ARCOM sobre:

- o que pode e o que não pode ser digitado numa ferramenta de IA (dados de
  cliente, senha, informação financeira, etc.);
- privacidade e proteção de dados;
- boas práticas no uso de IA no dia a dia de trabalho.

Isso vale tanto pra esse projeto quanto pra qualquer outra conversa que você
tiver com IA na empresa. Na dúvida sobre se pode colocar uma informação no
chat, não coloque — pergunte antes ao time responsável.

---

## 3. Passo a passo: como criar o seu projeto

Você vai usar o **Claude** (claude.ai) no formato **Projeto**. É como uma
pasta de trabalho onde o assistente já sabe as regras da ARCOM antes mesmo de
você pedir a primeira tela.

### Passo 1 — Crie um Projeto novo no Claude
No claude.ai, vá em **Projetos → Novo projeto** e dê um nome (ex.: "App de
Pedidos ARCOM").

### Passo 2 — Conecte este repositório do GitHub ao Project knowledge

Dentro do seu Projeto, procure a seção **Project knowledge** (geralmente na
coluna da direita). É ali que o Claude guarda o material que vai consultar
antes de construir qualquer coisa pra você.

1. Clique no botão **+** da seção Project knowledge.
2. No menu que abrir, escolha a opção **GitHub**. Se for a primeira vez, o
   Claude vai pedir pra você autorizar o acesso à sua conta do GitHub —
   confirme a autorização.
3. Na busca, digite o nome do repositório (`arcom-atacadista/starter`) ou
   cole a URL dele, e clique no resultado pra abrir.
4. Uma lista de pastas e arquivos vai aparecer, já com tudo marcado. Deixe
   selecionadas as pastas `design/`, `privacy/`, `specs/` e o `README.md`.
5. Clique em **Add files**.

Pronto — os arquivos do repositório agora fazem parte do Project knowledge, e
é esse material que ensina o Claude a seguir os padrões da ARCOM (visual,
técnico e de segurança) em tudo o que ele construir para você.

> **Dica:** se o repositório for atualizado no futuro (nova versão do
> padrão), volte nessa mesma seção e use o ícone de **sincronizar** pra trazer
> as mudanças mais recentes, sem precisar montar o projeto de novo.

### Passo 3 — Cole as instruções no campo "Instruções"
Abra o arquivo `specs/INSTRUCOES-DO-PROJETO.md`, copie todo o conteúdo dele e
cole no campo **Instruções** do seu Projeto (o campo de configuração, não o
chat). É esse texto que diz ao Claude: "siga sempre os padrões da ARCOM,
mesmo que eu peça algo diferente sem saber."

### Passo 4 — Converse com o chat
A partir daqui, é só pedir o que você precisa em linguagem normal, como se
estivesse explicando pra uma pessoa da equipe. Alguns exemplos:

- "Cria uma landing page pra apresentar o time de vendas"
- "Faz um formulário de cadastro de fornecedor com nome, CNPJ e telefone"
- "Monta uma calculadora de frete por peso e distância"
- "Cria um painel simples mostrando os pedidos do dia"

O Claude vai perguntar o que faltar e avisar quando algum pedido não for
permitido pelas regras da ARCOM — nesses casos, ele sempre vai te oferecer uma
alternativa que funciona.

### Passo 5 — Peça pra subir o projeto
Quando o projeto estiver pronto, peça ajuda ao time de infraestrutura pra
publicá-lo. Todo projeto criado nesse padrão sobe com um único comando
(`docker compose up --build`), então essa etapa é rápida.

---

## 4. O que já vem garantido, sem você precisar pedir

Você não precisa entender nada disso — é só pra você saber que está protegido:

- **Visual sempre da ARCOM**: cores, fonte e estilo seguem automaticamente o
  Design System da empresa (`design/ARCOM-Design-System.md`). Nenhuma tela sai
  fora do padrão.
- **Segurança básica**: senha, chave de acesso e dado sensível nunca ficam
  expostos no projeto. Login, formulário e qualquer parte que lida com dado de
  usuário seguem um checklist de segurança interno.
- **Tecnologia padronizada**: o Claude só usa as ferramentas que a ARCOM já
  aprovou. Isso evita depender de bibliotecas desconhecidas, instáveis ou fora
  de suporte.
- **Simplicidade primeiro**: todo projeto começa o mais simples possível (só
  a parte visual). Um banco de dados ou servidor só entra quando o seu pedido
  realmente exigir isso — e o chat avisa antes de adicionar essa
  complexidade.

---

## 5. Perguntas comuns

**Preciso saber programar?**
Não. Você descreve o que quer em português normal e o Claude escreve o código.

**Posso pedir qualquer coisa?**
Quase tudo. Se o pedido esbarrar numa regra da empresa (por exemplo, usar uma
ferramenta não aprovada), o chat explica o motivo em uma frase e sugere o
caminho permitido.

**E se eu quiser mudar o projeto depois de pronto?**
Sem problema — volte na mesma conversa do Projeto e peça o ajuste. O Claude
edita o que já existe em vez de recriar do zero.

**Onde peço ajuda se algo não funcionar?**
Fale com o time de infraestrutura/tecnologia da ARCOM. Eles ajudam a publicar
o projeto e resolver qualquer travamento técnico.

**Os arquivos deste repositório mudam com o tempo?**
Sim — o padrão evolui (a versão atual está em `specs/VERSAO.md`). Quando
houver uma atualização, substitua os arquivos antigos da pasta `specs/` no
Conhecimento do seu Projeto pelos novos, e recole o `INSTRUCOES-DO-PROJETO.md`
se ele tiver mudado.

