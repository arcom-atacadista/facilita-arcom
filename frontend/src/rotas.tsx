// Todas as rotas num arquivo só (ver system-design/padroes/02-frontend.md).
// Uma tela por arquivo em src/pages/, sempre com lazy — cada rota vira o
// próprio chunk do build.
import { lazy, Suspense } from "react";
import { Navigate, Route, Routes, useLocation } from "react-router-dom";
import { Shell } from "./components/Shell";
import { ErroDeTela } from "./core/ErroDeTela";
import { RotaProtegida } from "./core/sessao";

const Entrar = lazy(() => import("./pages/Entrar"));
const PrimeiroAcesso = lazy(() => import("./pages/PrimeiroAcesso"));
const Negociar = lazy(() => import("./pages/Negociar"));

const Carteira = lazy(() => import("./pages/Carteira"));
const Divida = lazy(() => import("./pages/Divida"));
const Acordos = lazy(() => import("./pages/Acordos"));
const Disparos = lazy(() => import("./pages/Disparos"));
const Politicas = lazy(() => import("./pages/Politicas"));
const Usuarios = lazy(() => import("./pages/Usuarios"));

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
          {/* Públicas. /negociar/:token é a tela que o cliente devedor abre
              pelo link da mensagem — sem conta, sem menu. */}
          <Route path="/entrar" element={<Entrar />} />
          <Route path="/primeiro-acesso" element={<PrimeiroAcesso />} />
          <Route path="/negociar/:token" element={<Negociar />} />

          {/* Privadas. RotaProtegida manda pro login quem não tem sessão;
              esconder um item de menu é conveniência, quem barra o acesso de
              verdade é a rota do backend. */}
          <Route element={<RotaProtegida />}>
            <Route element={<Shell />}>
              <Route path="/" element={<Navigate to="/carteira" replace />} />
              <Route path="/carteira" element={<Carteira />} />
              <Route path="/carteira/:id" element={<Divida />} />
              <Route path="/acordos" element={<Acordos />} />
              <Route path="/disparos" element={<Disparos />} />
              <Route path="/politicas" element={<Politicas />} />
              <Route path="/usuarios" element={<Usuarios />} />
            </Route>
          </Route>

          <Route path="*" element={<NaoEncontrado />} />
        </Routes>
      </Suspense>
    </ErroDeTela>
  );
}
