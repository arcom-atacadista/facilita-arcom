# 04 — Docker: o modelo, o que pode/não pode, como rodar e como ir pra produção

Aqui está como o projeto vira containers, como você sobe em desenvolvimento e
como sobe em produção de um jeito seguro.

## Como rodar (desenvolvimento)

Na raiz do projeto:

```bash
docker compose up --build
```

Isso builda e sobe frontend, backend, `migrate` (aplica as migrations e sai) e
(se houver) postgres. Redis não vem por padrão — só existe se alguém pediu
pra adicionar (ver `01-stack-permitida.md`). Abra **http://localhost:8080**. Pra parar:
`Ctrl+C`. Pra derrubar e limpar: `docker compose down` (adicione `-v` pra
apagar os dados do banco).

## O modelo Docker

- **frontend** — build multi-stage: `node:22-alpine` faz `pnpm install` +
  `pnpm run build`; o resultado é servido por
  `nginxinc/nginx-unprivileged:1.27-alpine` na porta 8080 (publicada em 8080).
  O proxy de `/api` pro backend está no `nginx.conf.template` (envsubst da
  variável `API_TARGET` em runtime). Os headers de segurança do conteúdo
  estático ficam em `nginx-headers-seguranca.conf`, incluído por `location`
  (ver `02-frontend.md`) — não em `/api/`, onde quem manda os headers é o
  próprio backend.
- **backend** — Go: build em `golang:1.25-alpine`, runtime em `alpine:3.20`.
  Escuta na porta **3000**, exposta só pra rede interna do compose. O
  healthcheck consulta `GET /api/ready` (não `/api/health` — ver
  `09-contrato-api.md`).
- **migrate** — mesma imagem do backend, comando `./migrate`: aplica as
  migrations do Postgres (goose) e sai (código 0 = sucesso). O `backend` só
  sobe depois que este serviço termina (`service_completed_successfully`) —
  nunca migration rodando dentro do boot do servidor.
- **postgres** — `postgres:16-alpine`, com volume `pgdata` pros dados.
- **redis** (opt-in, não vem no template base) — `redis:7-alpine`, só se o
  projeto precisar (ver `01-stack-permitida.md`). Sem volume por padrão: roda
  só em memória, sem persistência (`--save ""`) — não precisa de disco.

Só o **frontend** publica porta pro seu computador (8080). O backend, o banco e
o cache conversam pela rede interna do compose — o front alcança o backend pelo
nome do serviço (`backend:3000`).

## Regras de build que NÃO são opcionais

Estas três já vêm no template e não devem ser removidas — cada uma existe porque
a falta dela já quebrou (ou vazou) projeto de verdade:

1. **Versão do pnpm pinada.** O `package.json` do frontend tem
   `"packageManager": "pnpm@<versão>"` e o `Dockerfile` faz
   `corepack prepare pnpm@<versão> --activate`. Sem isso o corepack baixa a
   pnpm "latest", que exige um Node mais novo que o `node:20-alpine` e **quebra
   o build** (erro `ERR_UNKNOWN_BUILTIN_MODULE: node:sqlite`). A versão do pnpm
   tem que casar com o `lockfileVersion` do `pnpm-lock.yaml`.
2. **`.dockerignore` no `frontend/` e no `backend/`.** O build faz `COPY . .`;
   sem o `.dockerignore`, o `.env` (com chaves reais), `node_modules`, `dist` e
   `.git` entram na imagem — **vazamento de segredo** e imagem gigante. Sempre
   ignore, no mínimo: `.env`, `.env.*`, `node_modules`, `dist`, binários
   compilados e logs.
3. **Container roda como usuário non-root.** Frontend: build com `USER node`
   (com `COREPACK_HOME` dentro da imagem, pra não precisar baixar o pnpm de
   novo em runtime); runtime já é non-root de fábrica na imagem
   `nginx-unprivileged`. Backend com um usuário criado no Dockerfile
   (`adduser`). Se o app for comprometido, o atacante não é root dentro do
   container.

## O que PODE no docker-compose

- publicar a porta do **frontend** (`ports: ["8080:8080"]`)
- usar as imagens do allowlist (`node:22-alpine`,
  `nginxinc/nginx-unprivileged:1.27-alpine`, `golang:1.25-alpine`,
  `alpine:3.20`, `postgres:16-alpine`, `redis:7-alpine`)
- volumes nomeados pros dados do banco/cache
- em **produção**, um segundo arquivo de override (`docker-compose.prod.yml`) —
  ver a seção de produção abaixo

## O que NÃO PODE

- imagem fora do allowlist
- `privileged`, `cap_add`, `network_mode: host`, `pid: host`
- bind mount de caminho do host na forma `- /caminho/do/host:/dentro` (exceto a
  persistência de dados de produção descrita abaixo, que é declarada como
  **volume nomeado** com `driver_opts`, não como bind direto)
- `build.context` fora de `./frontend` ou `./backend`
- alterar o `.npmrc` (tem que manter `ignore-scripts=true`)
- `package-lock.json` ou `yarn.lock` no repo (só `pnpm-lock.yaml`)

## Variáveis de ambiente

Pra rodar local já vêm com valor padrão (funciona sem configurar nada). Pra
mudar, crie um arquivo `.env` na raiz:

```
PROJECT_NAME=meu-projeto
POSTGRES_PASSWORD=umaSenhaForte
```

Segredos de verdade (chave de API de terceiros etc.) nunca vão no frontend nem
commitados — em produção eles vêm do `.env.prod` (abaixo), guardado só na infra.

---

## Produção

O `docker compose up --build` puro é pro seu computador. Pra produção, use um
**arquivo de override** (`docker-compose.prod.yml`) junto com o base — assim o
dev continua simples e a produção fica endurecida sem duplicar arquivo.

```bash
docker compose -f docker-compose.yml -f docker-compose.prod.yml \
  --env-file .env.prod up -d --build
```

O que o override de produção deve fazer (e por quê):

- **Secrets sem fallback.** Em vez de `${JWT_SECRET:-dev-secret}`, use
  `${JWT_SECRET:?defina no .env.prod}`. Se faltar uma chave, o `up` **recusa a
  subir** em vez de rodar com valor de desenvolvimento por engano.
- **`restart: always`** (em vez de `unless-stopped`) — os serviços voltam
  sozinhos depois de um reboot do host.
- **Log rotacionado** (`json-file`, `max-size`/`max-file`) — pra não encher o
  disco do host.
- **Limites de CPU/memória** por serviço — pra um serviço com vazamento não
  derrubar os outros.
- **Publicar a porta só onde precisa.** Se o reverse proxy roda no mesmo host,
  publique em `127.0.0.1` (ex.: `"127.0.0.1:8080:8080"`), nunca aberto pra
  internet sem TLS.
- **Use `ports: !override` no serviço `frontend` do override.** Sem essa tag,
  o Compose funde a lista de `ports` do override com a do base em vez de
  substituir — o `"8080:8080"` (0.0.0.0) do `docker-compose.yml` continua
  valendo, e a publicação em `127.0.0.1` do override tenta abrir a mesma
  porta de novo por cima. A segunda falha sempre com "address already in
  use", mesmo sem nenhum processo externo ocupando a porta — o frontend nunca
  sobe em produção. `!override` (Compose ≥2.24) força a substituição.

### TLS / domínio

O compose **não termina TLS**. Em produção, um reverse proxy (Caddy, Traefik,
Nginx ou o load balancer que a infra já usa) fica na frente, cuida do
certificado HTTPS e repassa HTTP pra porta publicada do frontend. Decida com a
infra qual proxy usar e pra onde apontar o domínio — é esse proxy externo quem
decide qual domínio é aceito; o nginx do frontend (dentro do compose) não
precisa replicar essa checagem.

### Persistência dos dados em disco de verdade

Em produção os dados do Postgres/Redis não devem ficar num volume anônimo no
disco de sistema — devem ir pro storage/array que a infra faz backup. Declare os
volumes nomeados apontando pro caminho do host via `driver_opts` (continua sendo
"volume nomeado", que é o permitido):

```yaml
volumes:
  pgdata:
    driver: local
    driver_opts:
      type: none
      o: bind
      device: ${DATA_DIR:-/array/<projeto>/data}/postgres
```

Antes do primeiro `up`, o diretório do Postgres precisa existir e ter o dono
certo (senão ele recusa a inicializar), usando o UID da imagem oficial
(`postgres:16-alpine` = `70:70`).

### Teto de disco 100% (Postgres — Redis normalmente não precisa)

O Docker **não** limita tamanho de volume (`deploy.resources` só cobre
CPU/memória). Pra um teto de disco de verdade, cada pasta de dados ganha um
**filesystem próprio de tamanho fixo** — quando enche, o kernel barra a escrita
e não passa do tamanho. O template traz um script que faz isso (imagem loopback
ext4) e já cria os diretórios com o dono certo. Rode uma vez, como root:

```bash
sudo ./scripts/setup-storage-prod.sh
```

Isso já cobre o caso comum: Redis (se o projeto tiver) roda **sem
persistência** por padrão (`--save ""`, ver `01-stack-permitida.md`) — não
grava nada em disco, não precisa de teto de disco nenhum, só do teto de
**memória** via `--maxmemory` no `docker-compose.yml` (imposto pelo próprio
Redis). Só no caso raro de o projeto exigir Redis durável (persistência
ligada de propósito) é que existe algo em disco pra limitar:

```bash
sudo COM_REDIS=1 PG_SIZE=10G REDIS_SIZE=2G ./scripts/setup-storage-prod.sh
```

Nesse caso (UID da imagem oficial `redis:7-alpine` = `999:1000`), a
memória de cada container também tem teto via `deploy.resources.limits.memory`
no override de produção.

### `.env.prod`

Fica **só no host de produção**, nunca commitado (adicione ao `.gitignore`).
Use o mesmo `.env.example` da raiz como modelo: copie pra `.env.prod` e
preencha todos os valores com segredos reais — não existe um exemplo separado
pra produção, é o mesmo arquivo, só que sem fallback. Gere segredos novos
(`openssl rand -base64 48`) — nunca reuse os de desenvolvimento.

### Build no host vs. registry

Pra projetos pequenos, buildar no próprio host de produção (`up -d --build`)
está de bom tamanho. Se o projeto crescer, o padrão é buildar uma vez numa
máquina/CI, empurrar a imagem pronta pra um Docker registry e o host de produção
só dar `pull` — mas isso é decisão de infra, não obrigatório pelo padrão.
