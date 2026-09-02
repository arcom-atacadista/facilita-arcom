---
name: deploy-producao
description: Runbook pra subir o projeto em produção com o override de produção, storage com teto de disco e secrets. Use quando o usuário quiser fazer deploy, ir pra produção ou configurar o servidor.
---

# Deploy de produção

Detalhes completos em `padroes/04-docker-deploy.md`. Resumo do runbook:

## Uma vez, no host de produção (como root)

1. **Storage com teto de disco 100%** (cria os diretórios, dono certo e o teto):
   ```bash
   sudo ./scripts/setup-storage-prod.sh          # sem Redis, é só rodar assim (padrão); COM_REDIS=1 se o projeto usa
   # tamanhos: sudo PG_SIZE=10G REDIS_SIZE=2G ./scripts/setup-storage-prod.sh
   ```
2. **Secrets**: copie `.env.example` pra `.env.prod` e preencha com valores reais
   (gere novos: `openssl rand -base64 48`). Nunca comite; nunca reuse os de dev.

## Subir

```bash
docker compose -f docker-compose.yml -f docker-compose.prod.yml \
  --env-file .env.prod up -d --build
```

## Checar

- `docker compose ps` — `migrate` com `Exited (0)`, os demais `healthy`.
- `curl localhost:8080/api/ready` — 200 (confirma Postgres/Redis respondendo).
- `./scripts/verificar.sh` — varredura de lint/segurança antes de considerar
  o deploy pronto (ver `padroes/11-guards.md`).
- O reverse proxy/TLS fica FORA do compose (nginx/Caddy/LB da infra) apontando
  pra porta publicada do frontend (`127.0.0.1:8080` por padrão) — esse proxy
  externo é quem decide qual domínio é aceito.

## Nunca

- Não publique porta aberta pra internet sem TLS na frente.
- Não deixe secret com fallback em produção (o `${VAR:?}` recusa subir se faltar).
- Não builde no host se o projeto crescer — aí passa a buildar no CI e o host só
  dá `pull` (decisão de infra).
