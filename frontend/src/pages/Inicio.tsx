import { useQuery } from "@tanstack/react-query";
import { CheckCircle2, Loader2, XCircle } from "lucide-react";
import { api } from "@/core/api";
import { cn } from "@/core/cn";

type SaudeBackend = { status: string };

// /api/health é infraestrutura (liveness), fora do cliente versionado
// "/api/v1" — por isso o override de baseURL só nesta chamada. Rotas de
// negócio do projeto usam `api.get("/rota")` normalmente, sem isto.
async function checarBackend(): Promise<SaudeBackend> {
  const { data } = await api.get<SaudeBackend>("/health", {
    baseURL: "/api",
    skipErroGlobal: true, // a própria tela já mostra o estado; sem toast redundante
  });
  return data;
}

export default function Inicio() {
  const { data, isPending, isError } = useQuery({
    queryKey: ["saude-backend"],
    queryFn: checarBackend,
    retry: false,
  });

  const estado = isPending
    ? "carregando"
    : isError || data?.status !== "ok"
      ? "erro"
      : "ok";

  return (
    <main className="min-h-screen grid place-items-center p-6">
      <div className="w-full max-w-md rounded-lg border border-surface-border bg-white p-8 shadow-sm">
        <p className="text-xs font-bold uppercase tracking-wider text-verde-arcom">
          ARCOM
        </p>
        <h1 className="mt-1 text-3xl font-black text-verde-escuro">
          Meu Projeto
        </h1>
        <p className="mt-2 text-sm">
          Base do Padrão ARCOM no ar. Edite{" "}
          <code>frontend/src/pages/Inicio.tsx</code> pra começar.
        </p>

        <div
          className={cn(
            "mt-6 flex items-center gap-2 rounded-md border px-4 py-3 text-sm",
            estado === "ok" && "border-verde-arcom/30 text-verde-arcom",
            estado === "erro" && "border-danger/30 text-danger",
            estado === "carregando" && "border-surface-border",
          )}
        >
          {estado === "carregando" && (
            <>
              <Loader2 className="size-5 animate-spin" />
              Checando o backend...
            </>
          )}
          {estado === "ok" && (
            <>
              <CheckCircle2 className="size-5" />
              Backend conectado (GET /api/health).
            </>
          )}
          {estado === "erro" && (
            <>
              <XCircle className="size-5" />
              Backend não respondeu. Rode com <code>docker compose up</code>.
            </>
          )}
        </div>
      </div>
    </main>
  );
}
