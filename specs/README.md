# Meu Projeto ARCOM

> Novo por aqui? Leia [`system-design/LEIA-PRIMEIRO.md`](system-design/LEIA-PRIMEIRO.md) primeiro.

## Rodar

```bash
docker compose up --build
```

Abra http://localhost:8080

## Estrutura

- `frontend/` — React + Vite (Tailwind + shadcn)
- `backend/` — Go (API sob `/api/v1`, `/api/health`, `/api/ready`)
- `docker-compose.yml` — sobe tudo (frontend, backend, `migrate`, postgres —
  redis só se o projeto tiver adicionado, não vem por padrão)

Padrões e o que pode/não pode: veja `system-design/padroes/`.
