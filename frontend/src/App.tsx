import { QueryClientProvider } from "@tanstack/react-query";
import { BrowserRouter } from "react-router-dom";
import { Toaster } from "sonner";
import { BarraDeCarregamento } from "./core/carregando";
import { ErroDeTela } from "./core/ErroDeTela";
import { queryClient } from "./core/query";
import { ProvedorDeSessao } from "./core/sessao";
import { Rotas } from "./rotas";

export default function App() {
  return (
    <ErroDeTela>
      <QueryClientProvider client={queryClient}>
        <ProvedorDeSessao>
          <BrowserRouter>
            <BarraDeCarregamento />
            <Rotas />
          </BrowserRouter>
        </ProvedorDeSessao>
        <Toaster position="top-right" richColors closeButton />
      </QueryClientProvider>
    </ErroDeTela>
  );
}
