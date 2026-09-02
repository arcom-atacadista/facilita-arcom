// Cliente HTTP único do app. Sempre "/api/v1" — o Vite faz o proxy pro
// backend (mesma origem, sem CORS). Nunca coloque a URL do servidor na mão.
//
// A sessão vive em cookie HttpOnly (o backend define; ver
// padroes/02-frontend.md e .claude/skills/autenticacao) — por isso
// `withCredentials: true` e por isso este módulo NUNCA lê nem grava token em
// lugar nenhum. Não existe "Authorization: Bearer" aqui.
import axios, { type AxiosError } from "axios";
import { normalizarErro, type ErroApi } from "./erro";

// Módulo augmentation: toda AxiosRequestConfig do projeto ganha este campo —
// assim `api.get(url, { skipErroGlobal: true })` funciona tipado em qualquer
// chamada, sem precisar de um tipo próprio em cada call site.
declare module "axios" {
  export interface AxiosRequestConfig {
    /** Não mostrar toast nem disparar `onNaoAutenticado` para esta requisição
     * específica (ex.: a própria checagem de sessão no boot). */
    skipErroGlobal?: boolean;
  }
}

export const api = axios.create({
  baseURL: "/api/v1",
  timeout: 20_000,
  withCredentials: true,
  headers: { "X-Requested-With": "XMLHttpRequest" },
});

type Callbacks = {
  /** 401: sessão ausente/expirada. Limpe o estado de sessão e mande pro login. */
  onNaoAutenticado: () => void;
  /** Qualquer outro erro que deva virar toast. */
  onErro: (mensagem: string) => void;
};

let idInterceptorResposta: number | null = null;

/**
 * Liga os callbacks de sessão/toast ao cliente axios. Chamado uma vez pelo
 * provider de sessão (ver core/sessao.tsx). Faz eject do interceptor anterior
 * antes de registrar de novo — sem isso, um useEffect que roda mais de uma
 * vez (StrictMode, HMR) empilha interceptors e o comportamento duplica.
 */
export function configurarApi({ onNaoAutenticado, onErro }: Callbacks) {
  if (idInterceptorResposta !== null) {
    api.interceptors.response.eject(idInterceptorResposta);
  }

  idInterceptorResposta = api.interceptors.response.use(
    (resposta) => resposta,
    (erro: AxiosError) => {
      const problema: ErroApi = normalizarErro(
        erro.response?.status,
        erro.response?.data,
      );

      if (erro.config?.skipErroGlobal) {
        return Promise.reject(problema);
      }

      if (problema.status === 401) {
        onNaoAutenticado();
      } else {
        onErro(problema.mensagem);
      }

      return Promise.reject(problema);
    },
  );
}
