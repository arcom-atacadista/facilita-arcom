// Todas as rotas num arquivo só (ver padroes/02-frontend.md). Uma tela por
// arquivo em src/pages/, sempre carregada com lazy — cada rota vira o próprio
// chunk do build.
import { lazy, Suspense } from "react";
import { Route, Routes, useLocation } from "react-router-dom";
import { ErroDeTela } from "./core/ErroDeTela";

const Inicio = lazy(() => import("./pages/Inicio"));
const NaoEncontrado = lazy(() => import("./pages/NaoEncontrado"));

function TelaDeCarregamento() {
  return null; // a barra de carregamento global já indica progresso
}

export function Rotas() {
  const location = useLocation();

  return (
    <ErroDeTela chave={location.pathname}>
      <Suspense fallback={<TelaDeCarregamento />}>
        <Routes>
          <Route path="/" element={<Inicio />} />

          {/* Rotas privadas do projeto entram envolvidas em <RotaProtegida/>
              (core/sessao.tsx), assim que o projeto tiver login:

              <Route element={<RotaProtegida />}>
                <Route path="/painel" element={<Painel />} />
              </Route>
          */}

          <Route path="*" element={<NaoEncontrado />} />
        </Routes>
      </Suspense>
    </ErroDeTela>
  );
}
