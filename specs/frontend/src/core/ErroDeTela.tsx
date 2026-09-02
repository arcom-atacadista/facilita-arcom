// Boundary de erro de runtime — trata "quebrou" (Suspense trata "esperando",
// ver padroes/02-frontend.md). Fallback em português, nunca mostra a stack.
import { AlertTriangle } from "lucide-react";
import { ErrorBoundary, type FallbackProps } from "react-error-boundary";

function TelaDeErro({ resetErrorBoundary }: FallbackProps) {
  return (
    <div className="flex min-h-[50vh] flex-col items-center justify-center gap-4 p-8 text-center">
      <AlertTriangle className="size-10 text-danger" />
      <div>
        <p className="text-lg font-semibold text-verde-escuro">
          Algo deu errado nesta tela.
        </p>
        <p className="mt-1 text-sm text-arcom-gray">
          Tente novamente — se persistir, avise o suporte.
        </p>
      </div>
      <button
        type="button"
        onClick={resetErrorBoundary}
        className="rounded-md bg-verde-arcom px-4 py-2 text-sm font-medium text-white transition-colors duration-fast hover:brightness-90"
      >
        Tentar de novo
      </button>
    </div>
  );
}

/**
 * Envolve uma rota/seção da tela. `chave` reseta o boundary quando muda (ex.:
 * o `pathname` da rota) — sem isso, navegar pra outra tela ainda mostra o
 * erro da anterior.
 */
export function ErroDeTela({
  children,
  chave,
}: {
  children: React.ReactNode;
  chave?: string;
}) {
  return (
    <ErrorBoundary FallbackComponent={TelaDeErro} resetKeys={[chave]}>
      {children}
    </ErrorBoundary>
  );
}
