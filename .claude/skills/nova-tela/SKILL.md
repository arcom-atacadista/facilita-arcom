---
name: nova-tela
description: Cria uma nova tela (página) no frontend seguindo o Padrão ARCOM e o Design System. Use quando o usuário pedir uma tela, página, formulário ou view nova.
---

# Nova tela (frontend)

Antes de escrever, leia `padroes/02-frontend.md`, `padroes/08-design-system.md`
e `padroes/09-contrato-api.md`.

## Passos

1. Crie o arquivo em `frontend/src/pages/<Nome>.tsx` — uma tela por arquivo,
   só composição/layout.
2. Registre a rota em `frontend/src/rotas.tsx`, num lugar só. Se a tela exige
   login, envolva com `<RotaProtegida/>` (`core/sessao.tsx`).
3. Chamadas ao backend **sempre** via `@/core/api` (`api.get("/rota")`), nunca
   URL na mão — vira `/api/v1/rota`. **Não** use `try/catch` pra erro de API:
   o interceptor já mostra toast sozinho (ver `02-frontend.md`).
4. Dado de servidor sempre por `@tanstack/react-query` (`useQuery`/
   `useMutation` com chave em factory), nunca `useEffect` + `useState` manual.
5. Formulários com `react-hook-form` + `zod` (`@hookform/resolvers/zod`);
   validação de verdade também no backend.
6. Feedback de sucesso/erro pontual: `notificar.sucesso/erro` (`@/core/toast`)
   — não repita o que o interceptor já cobre.

## Design System (obrigatório)

- Cores, fonte (Red Hat Display), raios e sombras só dos tokens do
  `tailwind.config.js`. Nunca valor solto, nunca emoji.
- Fundo de página `bg-surface` (nunca branco puro); cards brancos com
  `border-surface-border` + `shadow-sm`; verde é acento, não fundo dominante.
- Ícones do `lucide-react`.
- Loading: se a tela tem uma query "principal", deixe a barra de loading
  global (`core/carregando.tsx`) indicar — normalmente não precisa de spinner
  próprio. Use skeleton (mesmo layout do conteúdo real) se quiser um estado de
  carregamento mais explícito.

## Não faça

- Não crie backend/endpoint "por via das dúvidas" (ver `07-frontend-primeiro.md`).
- Não use lib fora do `01-stack-permitida.md`.
- Não coloque segredo no frontend.
- Não guarde token/sessão/permissão em `localStorage`/`sessionStorage`
  (bloqueado por ESLint) — sessão é cookie, ver `core/sessao.tsx`.
- Não use `dangerouslySetInnerHTML` sem passar pela skill `seguranca`.
