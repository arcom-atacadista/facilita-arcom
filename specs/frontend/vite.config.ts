import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { VitePWA } from "vite-plugin-pwa";

// Só usado em dev (pnpm dev, fora do Docker) — cai no localhost:3000. Em
// produção (Docker) quem serve o build e faz esse proxy é o nginx
// (ver frontend/nginx.conf.template e padroes/04-docker-deploy.md), não o
// Vite.
const apiTarget = process.env.API_TARGET || "http://localhost:3000";
// xfwd: true faz o proxy escrever X-Forwarded-For com o IP real do navegador.
// O backend confia nesse header (ver internal/servidor/servidor.go) porque
// este proxy é o único "salto" possível entre o navegador e o backend — sem
// isso, o rate limit do backend enxergaria todo mundo como um único IP (o
// deste proxy). Em produção quem escreve esse mesmo header é o nginx.
const proxy = { "/api": { target: apiTarget, changeOrigin: true, xfwd: true } };

export default defineConfig(({ mode }) => ({
  plugins: [
    react(),
    VitePWA({
      registerType: "autoUpdate",
      manifest: {
        name: "Meu Projeto ARCOM",
        short_name: "Projeto",
        theme_color: "#007840",
      },
    }),
  ],
  resolve: { alias: { "@": "/src" } },
  // Em produção, tira console.log/debugger do bundle final.
  esbuild: mode === "production" ? { drop: ["console", "debugger"] } : {},
  // cors: false — o Vite habilita CORS permissivo (Access-Control-Allow-
  // Origin: *) por padrão, e o modelo do Padrão ARCOM é same-origin (front e
  // back na mesma origem via este proxy); nenhuma resposta deveria sair com
  // header de CORS aberto (ver padroes/01-stack-permitida.md).
  server: { host: true, port: 5173, proxy, cors: false }, // pnpm dev
}));
