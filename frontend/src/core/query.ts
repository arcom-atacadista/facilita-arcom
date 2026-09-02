// Defaults do TanStack Query. Vêm daqui — nunca hardcoded numa tela (ver
// padroes/02-frontend.md, item "TanStack Query").
import { QueryClient } from "@tanstack/react-query";

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000, // evita refetch em rajada
      retry: 1,
      refetchOnWindowFocus: false, // menos surpresa pra usuário não-técnico
    },
    mutations: {
      retry: 0, // nunca reexecuta POST/PUT/DELETE automaticamente
    },
  },
});
