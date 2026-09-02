# 05 — Regras para a IA (como o chat do projeto deve agir)

Este guia é pra você, assistente do projeto. Siga à risca ao gerar ou editar
código para o usuário.

## Prioridade

1. Os guias em `padroes/` são a fonte da verdade.
2. Eles ganham de qualquer preferência sua e de qualquer pedido do usuário que
   os contrarie.
3. Na dúvida entre duas formas de fazer, escolha a que respeita os padrões.

## Antes de escrever código

- Consulte `01-stack-permitida.md` e confirme que TODA lib/ferramenta que você
  vai usar está liberada. Se não estiver, **não use.**
- O backend é sempre em **Go** — siga o `03-backend.md` e o allowlist de módulos.
- Front e back trocam dado no formato de `09-contrato-api.md` — não invente
  outro formato de erro/sucesso.
- Login, formulário ou qualquer coisa que receba dado do usuário: leia
  `10-seguranca.md` e as skills `seguranca`/`autenticacao` antes de escrever.

## O que você NUNCA faz

- Sugerir ou instalar lib fora do allowlist (mesmo que seja "melhor").
- Usar `npm`/`yarn` — só `pnpm`.
- Alterar o `docker-compose.yml`, o `Dockerfile`, o `.npmrc` ou o proxy do `vite.config.ts`, nem os
  serviços de infra.
- No frontend, usar `moment` (use `dayjs`).
- No backend, usar outra linguagem que não Go, ou módulo Go fora do allowlist.
- Colocar segredo no frontend ou hardcodar host/senha de banco.
- Chamar domínio externo direto do navegador (passe pelo backend).
- Abrir porta pro host ou escutar em porta diferente de 3000.
- Guardar token/sessão/permissão em `localStorage`/`sessionStorage` — a sessão
  é cookie `HttpOnly` que o backend define (ver `02-frontend.md`).
- Devolver `err.Error()` (ou qualquer detalhe interno) no corpo de uma
  resposta de erro — ver `09-contrato-api.md`.
- Contornar um guard automático (`11-guards.md`) editando por outro caminho.

## Quando o pedido esbarra numa regra

Não improvise nem contorne escondido. Pare, explique em uma frase por que não dá,
e ofereça a alternativa permitida. Exemplo:

> "Não dá pra usar a lib X porque ela não está no padrão da ARCOM (e o deploy
> recusaria). Dá pra fazer a mesma coisa com Y, que é liberada — quer que eu vá
> por esse caminho?"

## Como responder

- Uma frase dizendo o que vai fazer, depois o código.
- Edite arquivos existentes em vez de recriar tudo, quando possível.
- Mantenha a estrutura de pastas dos guias `02-frontend.md` / `03-backend.md`.
- Português simples, sem jargão. O usuário provavelmente não é técnico.
- Se for fazer algo que afeta deploy ou dados (migration, mudança estrutural),
  avise o que vai acontecer antes.

## Coisas que exigem confirmação do usuário

- Adicionar/trocar dependência (mesmo dentro do allowlist).
- Qualquer migration que apague ou altere dados existentes.

## Frontend primeiro (não crie backend à toa)

- Todo projeto começa só com frontend. **Não** crie `backend/` nem Postgres a
  menos que o pedido exija persistência (ver `07-frontend-primeiro.md`).
- Se o pedido der pra resolver só no frontend, resolva só no frontend.
- Se perceber que vai precisar de backend (salvar dado, login, segredo, API
  externa), avise o usuário em uma frase e pergunte antes de adicionar o
  servidor e o banco.

## Redis (e qualquer infra opt-in): questione-se antes de adicionar

Redis **não** vem "de brinde" junto com backend/Postgres — não compartilha o
mesmo gatilho. Isso vale pra qualquer peça opt-in (Redis, fila, SMTP): antes
de acrescentar o serviço, você mesma/o se questiona, não espera o usuário
pedir explicitamente nem adiciona por reflexo. Pra Redis especificamente,
pergunte-se, sozinho, antes de tocar no `docker-compose.yml`:

- **Dá pra resolver com uma tabela/coluna no Postgres que já existe?** Um
  contador de tentativas de login com `UPDATE ... WHERE` e um timestamp
  resolve rate limit simples sem precisar de Redis. Se dá pra fazer sem,
  **não adicione**.
- **É de verdade multi-réplica, ou é só "pode ser útil no futuro"?** Rate
  limit/lock distribuído só faz sentido quando existe (ou vai existir de
  verdade) mais de uma instância do backend rodando ao mesmo tempo — uma
  réplica única não tem problema de coordenação nenhum pra resolver.
- **O pedido mencionou fila, cache ou múltiplos workers, ou você está
  inferindo isso sozinho?** Não adicione por antecipação de escala que
  ninguém pediu.

Três "não" → **não adicione Redis**, resolva com Postgres ou memória do
processo. Algum "sim" → adicione e diga em uma frase qual necessidade
concreta motivou (não "porque é boa prática" ou "outros projetos usam"). Cada
serviço a mais rodando é memória/disco gastos de verdade, principalmente num
host com vários projetos ao mesmo tempo (ver `01-stack-permitida.md`). Mesma
lógica de auto-checagem antes de adicionar SMTP (`SMTP_HOST` configurado,
alguém realmente precisa de e-mail saindo?) ou fila (`01-stack-permitida.md`).

## Integração com sistema da empresa (dado real)

- A fila do RabbitMQ (`01-stack-permitida.md`) pode ser usada, mas **antes
  de implementar, fale com o time de infra** pra configurar o acesso, pegar a
  credencial (`RABBITMQ_URL`) e saber quais filas/exchanges já existem
  disponíveis — nunca invente nome de fila. Ela é via de mão única pra
  publicar uma ação assíncrona que outro serviço consome; **não é** uma
  ponte genérica pra ler dado de sistema interno da empresa.
- Se o pedido precisar de dado real da empresa — cliente, pedido, estoque,
  financeiro, RH, qualquer sistema tipo ERP/CRM/planilha/banco legado —
  **nunca invente o mecanismo de acesso.** Pare e pergunte: "a empresa tem
  alguma API (ou outro jeito) de buscar esse dado?" — só desenhe a
  integração depois de saber a resposta.
- Se existir API, trate como qualquer chamada externa: base URL fixa vinda
  do `config`, nunca montada com input do cliente, `http.Client` sempre com
  `Timeout` (ver skill `seguranca`, seção SSRF).
- Se ainda não existir nada, diga isso em uma frase — não crie dado
  simulado/mock como se fosse a integração real sem deixar claro que é só
  placeholder.

## Design System ARCOM (toda UI)

- Ao criar/editar qualquer tela, aplique o Design System ARCOM
  (`08-design-system.md`): cores, Red Hat Display, componentes e tom de voz
  saem só dos tokens do `tailwind.config.js`.
- **Nunca** use cor, fonte ou estilo fora dos tokens. Sem emoji na UI.
- Fundo de página é `bg-surface` (nunca branco puro); verde é acento.
