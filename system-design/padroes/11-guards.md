# 11 — Guards (o que é bloqueado automaticamente)

Os guias em `padroes/` são a fonte da verdade, mas um guia só é lido se
alguém (ou a IA) lembrar de ler. Os **guards** existem pra não depender disso:
são hooks do Claude Code (`.claude/settings.json`) que rodam antes/depois de
cada edição e **recusam** o que contraria o padrão, na hora — não em revisão
depois.

Dois níveis, por causa de latência:

| Nível | Quando roda | O que faz |
|---|---|---|
| **Hooks** (`scripts/guards/*.js`) | a cada `Edit`/`Write`/`Bash`, em milissegundos | grep/checagem rápida — bloqueia na hora |
| **`scripts/verificar.sh`** | sob demanda (ou pela skill `revisar-seguranca`) | varredura completa: `golangci-lint`, `govulncheck`, `eslint`, `tsc` |

Um hook não pode rodar `govulncheck` a cada tecla — ficaria inutilizável.
Por isso o hook pega o **padrão óbvio e rápido de detectar**; a varredura
completa (mais lenta, mais precisa) fica pra quando o código já existe.

## O que os hooks bloqueiam

**`proteger-arquivo.js`** (antes de `Edit`/`Write`/`MultiEdit`) — recusa editar:
`.npmrc`, `docker-compose.yml`/`docker-compose.prod.yml`, `Dockerfile`
(frontend/backend), `nginx.conf.template` e `nginx-headers-seguranca.conf`
(frontend), o bloco de proxy do `vite.config.ts`, `pnpm-lock.yaml`, `go.sum`.
Esses arquivos são "base pronta" (ver `00-visao-geral.md`) — editar sem
querer quebra o build de todo mundo.

**`bloquear-comando.js`** (antes de rodar `Bash`) — recusa: `npm install`/
`yarn add` (só `pnpm`), `curl ... | sh` (executa script baixado sem revisar),
`chmod 777`, `git add`/`git commit` incluindo `.env`/`.env.prod`,
`git commit --no-verify`.

**`verificar-segredo.js`** (depois de `Edit`/`Write`) — acusa: variável
`VITE_*` com nome `KEY`/`SECRET`/`TOKEN`/`PASSWORD` (vai pro bundle público,
ver `01-stack-permitida.md`), string com formato de chave de API hardcoded,
segredo escrito num `.env` que está versionado.

**`verificar-codigo.js`** (depois de `Edit`/`Write`) — assinaturas por
linguagem, uma por padrão real encontrado nesta revisão:

- **Go:** `fmt.Sprintf`/concatenação dentro de `Raw`/`Exec`/`Where` (SQLi) ·
  `math/rand` gerando token/senha/id (use `crypto/rand`) ·
  `==` comparando token/HMAC (use `crypto/subtle.ConstantTimeCompare`) ·
  `http.ListenAndServe(...)` direto (sem `http.Server`/timeouts) ·
  `.Updates(...)` sem `Select(...)`/DTO explícito (mass assignment) ·
  `err.Error()` escrito na resposta HTTP (vazamento) ·
  `http.Client{}` sem `Timeout`.
- **Frontend:** `dangerouslySetInnerHTML` fora do componente único de
  sanitização · `localStorage`/`sessionStorage` (sessão é cookie, ver
  `02-frontend.md`) · URL absoluta de servidor em vez de `/api/...`.

Rota fora de grupo com auth, rate limit fail-open no Redis, log sem máscara,
cor/fonte fora dos tokens e emoji na UI **não** têm grep automatizado ainda —
são achados que a skill `revisar-seguranca` cobre na varredura manual, mas o
hook rápido de `Edit`/`Write` não pega sozinho. Se um desses virar guard de
verdade, mova a entrada daqui pra cima.

**`verificar-dependencia.js`** (depois de mexer em `package.json`/`go.mod`) —
acusa lib fora do `01-stack-permitida.md`.

Cada achado sai no mesmo formato usado pela skill `revisar-seguranca`
(`id`, `criticidade`, `arquivo`, `linha`, `codigo`, `correção`) — consistência
entre o que bloqueia na hora e o que a auditoria completa relata depois.

## Rodar a varredura completa manualmente

```bash
./system-design/scripts/verificar.sh
```

Roda tudo (backend: `go vet`, `golangci-lint`, `govulncheck`, `go test -race`;
frontend: `eslint`, `tsc --noEmit`; mais os mesmos greps dos hooks) e imprime
um relatório. Ferramenta que não está instalada gera **aviso**, não erro — a
pessoa não-técnica não pode ficar travada por falta de `golangci-lint` local
(o Docker/CI, quando existir, sempre tem).

## Quando um guard bloqueia

Pare, explique o motivo em uma frase (qual regra, por quê) e ofereça o
caminho permitido — a mesma postura de `05-regras-para-a-ia.md`. Nunca
contorne o guard escondido (ex.: editando o arquivo por outro caminho).
