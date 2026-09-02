# Facilita ARCOM

Plataforma própria de crédito e cobrança da ARCOM S/A: gestão da carteira em
atraso (3 a 90 dias), régua de cobrança por faixa, disparo de mensagem e
negociação self-service pelo próprio cliente.

> Novo por aqui? Leia [`system-design/LEIA-PRIMEIRO.md`](system-design/LEIA-PRIMEIRO.md) primeiro.

## Rodar

```bash
cp .env.example .env
docker compose up --build
```

Abra http://localhost:8080

## Estrutura

- `frontend/` — React + Vite (Tailwind + shadcn), Design System ARCOM
- `backend/` — Go (API sob `/api/v1`, `/api/health`, `/api/ready`)
- `docker-compose.yml` — sobe tudo (frontend, backend, `migrate`, postgres)
- `system-design/` — o Padrão ARCOM (regras, design system, privacidade) — travado

Padrões e o que pode/não pode: veja `system-design/padroes/`.

## Integrações

- **Gateway ARCOM** (`https://kabana-api.arcom.com.br`) — origem dos débitos
  reais e do histórico de disparos da Nines. Somente leitura. A `X-API-Key`
  é provisionada pelo time de TI em produção (`GATEWAY_ARCOM_API_KEY`); sem
  ela a chamada responde 401, o que é esperado em dev.
- **Canal de envio de mensagem** — ainda não definido. A camada de disparo
  está pronta atrás de uma porta de saída (`internal/disparo`), hoje com a
  implementação de registro-apenas. Ver `backend/internal/disparo/canal.go`.
