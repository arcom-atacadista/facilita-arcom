// Loading global — barra fina no topo, nunca overlay bloqueando a tela (ver
// padroes/02-frontend.md, item "Loading global"). Soma o que o TanStack Query
// está fazendo com o contador manual de core/estadoUI.ts.
import { useEffect, useState } from "react";
import { useIsFetching, useIsMutating } from "@tanstack/react-query";
import { useEstadoUI } from "./estadoUI";
import { cn } from "./cn";

const ATRASO_MS = 200; // não pisca em requisição rápida

export function BarraDeCarregamento() {
  const emBusca = useIsFetching();
  const emMutacao = useIsMutating();
  const ocupadoManual = useEstadoUI((s) => s.ocupadoManual);
  const ocupado = emBusca + emMutacao + ocupadoManual > 0;

  const [visivel, setVisivel] = useState(false);

  useEffect(() => {
    if (!ocupado) {
      setVisivel(false);
      return;
    }
    const temporizador = setTimeout(() => setVisivel(true), ATRASO_MS);
    return () => clearTimeout(temporizador);
  }, [ocupado]);

  return (
    <div
      role="status"
      aria-busy={visivel}
      aria-label="Carregando"
      className={cn(
        "fixed inset-x-0 top-0 z-50 h-0.5 overflow-hidden bg-transparent transition-opacity duration-normal",
        visivel ? "opacity-100" : "opacity-0",
      )}
    >
      <div className="h-full w-1/3 animate-pulse bg-verde-arcom" />
    </div>
  );
}
