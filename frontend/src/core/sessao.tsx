// Sessão do usuário. O token vive num cookie HttpOnly (o backend define) —
// este módulo nunca lê nem grava token; só sabe "quem está logado" perguntando
// pro backend. Permissão NUNCA é persistida no cliente (nada de
// localStorage) — quem decide acesso é sempre o backend, na hora da
// requisição real (ver padroes/02-frontend.md, item "Sessão").
import { useEffect } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Navigate, Outlet, useLocation } from "react-router-dom";
import { api, configurarApi } from "./api";
import { notificar } from "./toast";

// Ajuste os campos conforme o projeto — o que importa é o formato do
// contrato: sempre o retorno mais recente do backend, nunca algo
// recalculado/guardado no front.
export type Usuario = {
  id: string;
  nome: string;
  email: string;
};

const CHAVE_SESSAO = ["sessao"] as const;

async function buscarSessao(): Promise<Usuario | null> {
  try {
    const { data } = await api.get<Usuario>("/sessao", {
      skipErroGlobal: true,
    });
    return data;
  } catch {
    return null; // sem sessão é estado normal (deslogado), não é erro de tela
  }
}

// Este arquivo mistura hooks/componentes com tipo e constante de propósito
// (é o módulo de sessão inteiro, coeso) — desliga o aviso de Fast Refresh
// pontualmente em vez de fragmentar em mais arquivos por causa de uma regra
// de DX.
// eslint-disable-next-line react-refresh/only-export-components
export function useSessao() {
  const { data, isLoading } = useQuery({
    queryKey: CHAVE_SESSAO,
    queryFn: buscarSessao,
    staleTime: 5 * 60_000,
    retry: false,
  });
  return { usuario: data ?? null, carregando: isLoading };
}

/**
 * Liga o cliente axios ao ciclo de vida da sessão. Monte uma vez, na raiz do
 * app (ver App.tsx).
 */
export function ProvedorDeSessao({ children }: { children: React.ReactNode }) {
  const queryClient = useQueryClient();

  useEffect(() => {
    configurarApi({
      onNaoAutenticado: () => {
        queryClient.setQueryData(CHAVE_SESSAO, null);
      },
      onErro: (mensagem) => notificar.erro(mensagem),
    });
  }, [queryClient]);

  return <>{children}</>;
}

/** Layout route de guarda — envolva as rotas privadas com ela em rotas.tsx. */
export function RotaProtegida() {
  const { usuario, carregando } = useSessao();
  const location = useLocation();

  if (carregando) return null;
  if (!usuario)
    return <Navigate to="/login" state={{ de: location }} replace />;

  return <Outlet />;
}
