# 02 — Frontend (React + Vite + Tailwind)

Convenções pra tela. Libs permitidas estão em `01-stack-permitida.md`. Leia
`09-contrato-api.md` antes deste — o formato de erro/dados vem de lá.

> Como roda: no Docker (produção), o **nginx** serve o build e faz **proxy de
> `/api`** pro backend (config no `nginx.conf.template`). Fora do Docker, use
> `pnpm dev` — o Vite faz o mesmo proxy, apontando pro `localhost:3000`
> (config no `vite.config.ts`).

## Estrutura de pastas

```
frontend/
├── .npmrc              # NÃO alterar
├── Dockerfile           # build (node) + runtime (nginx)
├── nginx.conf.template  # estático + proxy de /api em produção (não mexer)
├── nginx-headers-seguranca.conf  # headers de segurança do estático (não mexer)
├── vite.config.ts       # PWA + proxy de /api em dev (não mexer no proxy)
├── tailwind.config.js
├── postcss.config.js
├── package.json
├── pnpm-lock.yaml       # obrigatório
├── pnpm-workspace.yaml  # só o campo allowBuilds — NÃO transformar em monorepo
├── index.html
└── src/
    ├── main.tsx         # entrada (StrictMode + <App/>)
    ├── App.tsx          # providers (query, sessão, router) + <Rotas/>
    ├── rotas.tsx         # TODAS as rotas num arquivo só
    ├── index.css
    ├── core/            # infraestrutura da aplicação — NÃO é lugar de tela
    │   ├── api.ts        # axios + interceptors + configurarApi()
    │   ├── erro.ts        # ErroApi/Problema (ver 09-contrato-api.md)
    │   ├── query.ts       # QueryClient + defaults
    │   ├── toast.ts       # notificar.{sucesso,erro,aviso,info} (sonner)
    │   ├── carregando.tsx # barra de loading global
    │   ├── estadoUI.ts    # zustand — só o que não é dado de servidor
    │   ├── sessao.tsx     # useSessao() + <RotaProtegida/>
    │   ├── ErroDeTela.tsx # ErrorBoundary
    │   └── cn.ts          # helper de classes
    ├── components/
    │   └── ui/            # shadcn — gerado, não editar à mão
    ├── pages/             # uma tela por arquivo
    ├── hooks/
    └── tipos/
```

**Regra de import:** `core/`, `components/` e `hooks/` **nunca** importam de
`pages/` — o fluxo é sempre `core/components -> pages`, nunca o contrário
(imposto pelo `eslint-plugin-import`, ver `eslint.config.js`).

## Chamadas ao backend — `core/api.ts`

```ts
// src/core/api.ts (resumo — ver o arquivo completo no template)
export const api = axios.create({
  baseURL: "/api/v1",
  timeout: 20_000,
  withCredentials: true, // manda o cookie de sessão
});
```

```ts
import { api } from "@/core/api";
const { data } = await api.get<Produto[]>("/produtos"); // vira /api/v1/produtos
```

Nunca ponha `http://localhost:3000` ou URL de servidor na mão — só caminhos
relativos. O proxy do Vite cuida do resto (evita CORS). `/api/health` e
`/api/ready` são infraestrutura, fora de `/api/v1` — só use
`api.get("/health", { baseURL: "/api" })` no caso raro de precisar checá-los
do front (ver `src/pages/Inicio.tsx`).

### Erro de API — automático, sem `try/catch` na tela

O interceptor de `core/api.ts` já converte toda resposta de erro
(`application/problem+json`, ver `09-contrato-api.md`) num `ErroApi`
(`core/erro.ts`) e:
- em `401`, limpa a sessão (`core/sessao.tsx` cuida disso);
- em qualquer outro erro, chama `notificar.erro(mensagem)` (toast) sozinho.

**Nenhuma tela precisa de `try/catch` pra erro de API.** Use
`{ skipErroGlobal: true }` na config da chamada só quando a própria tela vai
tratar o erro de outro jeito (ex.: mostrar mensagem inline em vez de toast —
ver `checarBackend` em `pages/Inicio.tsx`, ou o boot da sessão).

```ts
await api.get("/produtos", { skipErroGlobal: true }); // opt-out pontual
```

## Loading global

`core/carregando.tsx` soma `useIsFetching()` + `useIsMutating()` do TanStack
Query com o contador manual de `core/estadoUI.ts` (zustand) numa barra fina no
topo — **nunca** overlay bloqueando a tela. Pra uma operação assíncrona que
não é uma query/mutation (ex.: gerar PDF no navegador), use
`useEstadoUI().iniciarOcupado()` / `.terminarOcupado()` ao redor dela.

Conteúdo de tela usa **skeleton com o layout da tela real**, nunca spinner
central — `isPending` de uma query já é o sinal certo pra decidir isso.

## Notificações — `core/toast.ts`

```ts
import { notificar } from "@/core/toast";
notificar.sucesso("Produto salvo com sucesso");
notificar.erro("Não foi possível salvar"); // erro de API já dispara sozinho
```

`sonner` por baixo, com deduplicação (5s) — várias falhas iguais viram 1 toast.

## Erro de runtime — `core/ErroDeTela.tsx`

Um boundary na raiz (`App.tsx`) e um por rota (`rotas.tsx`, com `chave` =
`location.pathname`, pra trocar de tela não deixar o erro da anterior preso).
Fallback em português com botão "Tentar de novo" — **nunca mostre a stack**.

## Sessão — `core/sessao.tsx`

**O token vive num cookie `HttpOnly` que o backend define.** O frontend nunca
lê, grava nem decide nada com base num token — só pergunta pro backend "quem
está logado" (`useSessao()`, que consulta `GET /api/v1/sessao` via
TanStack Query).

```tsx
// rotas.tsx — proteger uma rota
<Route element={<RotaProtegida />}>
  <Route path="/painel" element={<Painel />} />
</Route>
```

Regras que valem sempre, mesmo sob pressão do pedido do usuário:
- **Nunca** guarde token, sessão ou permissão em `localStorage`/`sessionStorage`
  (bloqueado por ESLint — `no-restricted-globals`). Se parece necessário,
  provavelmente o desenho está errado: pergunte pro backend de novo.
- Quem decide acesso a um recurso é **sempre o backend**, na hora da
  requisição real — a UI pode esconder um botão por conveniência, mas isso
  nunca é a proteção.
- Ver `.claude/skills/autenticacao` pro fluxo completo de login/logout.

## Validação — zod + react-hook-form

```tsx
const schema = z.object({
  nome: z.string().min(3, "Mínimo 3 caracteres"),
  preco: z.coerce.number().positive("Preço deve ser positivo"),
});
type FormData = z.infer<typeof schema>;

const form = useForm<FormData>({ resolver: zodResolver(schema) });
```

Validação no front é conveniência/UX; a que vale é a do backend
(`go-playground/validator`, ver `03-backend.md`). Nunca só uma das duas.

## Dados do servidor — TanStack Query

Cache/estado de servidor sempre por `@tanstack/react-query`, nunca
`useEffect` + `useState` manual. Defaults ficam em `core/query.ts`
(`staleTime`, `retry`) — **nunca hardcode isso numa tela**.

```ts
// uma chave por recurso, como factory — nunca array literal solto
export const produtoKeys = {
  all: ["produtos"] as const,
  lista: () => [...produtoKeys.all, "lista"] as const,
  item: (id: string) => [...produtoKeys.all, "item", id] as const,
};

const { data } = useQuery({ queryKey: produtoKeys.lista(), queryFn: () => api.get("/produtos") });
```

Invalide a query certa (`queryClient.invalidateQueries({ queryKey: produtoKeys.lista() })`)
no `onSuccess` da mutation.

## Navegação — `rotas.tsx`

`react-router-dom`, API declarativa (`<Routes>`), **todas as rotas num
arquivo só** (`src/rotas.tsx`). Cada página com `React.lazy` — vira o próprio
chunk do build. Layout route com `<Outlet/>` pra seção que compartilha casca
(sidebar, header). Uma tela por arquivo em `src/pages/`.

## Estilo — Tailwind + shadcn

Classes utilitárias do Tailwind (nada de CSS-in-JS). Helper `cn()`
(`core/cn.ts`):

```ts
import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}
```

Componentes de UI seguem o padrão shadcn (Radix + Tailwind), gerados em
`src/components/ui/` — não editar à mão, regerar. Ícones: `lucide-react`.

**Identidade visual:** toda UI segue o Design System ARCOM — cores, Red Hat
Display, raios, sombras e tom de voz vêm dos tokens do `tailwind.config.js`.
Ver `08-design-system.md` (obrigatório).

## Datas

`dayjs` com plugins `utc` e `timezone`. Nunca `moment`. O backend sempre
devolve UTC (ver `09-contrato-api.md`) — converta pro fuso local só na hora de
exibir.

## PWA

`vite-plugin-pwa` já configurado no `vite.config.ts`. Pra mudar nome/ícone,
edite só o bloco `manifest`.

## Variáveis de ambiente

Só `VITE_*` e só coisas **públicas** (vão pro bundle, legíveis no DevTools).
Segredo nunca no frontend.

## Qualidade

- `pnpm lint` (ESLint 9, flat config — `import/no-restricted-paths`,
  `react/no-danger`, `no-restricted-globals` pra storage, tipos checados).
- `pnpm typecheck` (`tsc --noEmit`) — o `vite build` sozinho não valida tipo.
- `pnpm run build` roda `tsc -b && vite build`: se o `tsc -b` deixar
  `vite.config.js`/`vite.config.d.ts`/`*.tsbuildinfo` na pasta, são artefatos
  gerados (já cobertos em `.gitignore`/`.dockerignore`) — apague-os, não
  commite.
