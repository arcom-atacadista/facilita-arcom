#!/usr/bin/env node
// Guard (PostToolUse, matcher Edit|Write): acusa dependência fora do
// allowlist quando package.json/go.mod é editado. Aproximado de propósito —
// a checagem exata (com transitivas) é scripts/verificar.sh. Ver
// padroes/01-stack-permitida.md e padroes/11-guards.md.
"use strict";
const fs = require("fs");
const path = require("path");

function avisar(motivo) {
  process.stdout.write(JSON.stringify({ decision: "block", reason: motivo }));
  process.exit(0);
}

function lerEntrada() {
  return new Promise((resolve) => {
    let raw = "";
    process.stdin.on("data", (d) => (raw += d));
    process.stdin.on("end", () => {
      try {
        resolve(JSON.parse(raw || "{}"));
      } catch {
        resolve({});
      }
    });
  });
}

// Mantenha em sincronia manual com padroes/01-stack-permitida.md.
const PERMITIDAS_FRONT = [
  /^react$/, /^react-dom$/, /^react-router-dom$/, /^@radix-ui\//, /^lucide-react$/,
  /^react-hook-form$/, /^@hookform\/resolvers$/, /^zod$/, /^@tanstack\/react-query$/,
  /^zustand$/, /^sonner$/, /^react-error-boundary$/, /^axios$/, /^dayjs$/, /^clsx$/,
  /^tailwind-merge$/, /^class-variance-authority$/, /^tailwindcss/, /^postcss$/,
  /^autoprefixer$/, /^vite$/, /^vite-plugin-pwa$/, /^typescript$/, /^eslint/,
  /^@typescript-eslint\//, /^typescript-eslint$/, /^@eslint\/js$/, /^globals$/,
  /^prettier$/, /^@vitejs\/plugin-react$/, /^@types\//, /^react-icons$/, /^dompurify$/,
];

const PERMITIDAS_BACK = [
  "github.com/go-chi/chi", "github.com/go-chi/httprate", "github.com/go-chi/httplog",
  "github.com/joho/godotenv", "gorm.io/gorm", "gorm.io/driver/postgres",
  "github.com/jackc/pgx", "github.com/pressly/goose", "github.com/redis/go-redis",
  "github.com/hibiken/asynq", "golang.org/x/crypto", "github.com/golang-jwt/jwt",
  "github.com/go-playground/validator", "github.com/robfig/cron",
  "github.com/rabbitmq/amqp091-go", "github.com/google/uuid",
  "github.com/SherClockHolmes/webpush-go", "github.com/wneessen/go-mail",
];

function checarPackageJson(arquivo) {
  let json;
  try {
    json = JSON.parse(fs.readFileSync(arquivo, "utf8"));
  } catch {
    return;
  }
  const deps = Object.keys({ ...(json.dependencies || {}), ...(json.devDependencies || {}) });
  const fora = deps.filter((dep) => !PERMITIDAS_FRONT.some((re) => re.test(dep)));
  if (fora.length > 0) {
    avisar(`Dependência(s) fora do allowlist (padroes/01-stack-permitida.md), confirme antes de manter: ${fora.join(", ")}`);
  }
}

function checarGoMod(arquivo) {
  const conteudo = fs.readFileSync(arquivo, "utf8");
  const linhas = conteudo.split("\n");
  const fora = [];
  for (const linha of linhas) {
    if (linha.includes("// indirect")) continue; // transitiva — não é escolha da IA, go mod tidy que traz
    const m = linha.match(/^\s+([a-zA-Z0-9./_-]+)\s+v[0-9]/);
    if (!m) continue;
    const mod = m[1];
    if (mod === "meu-projeto") continue; // não deveria aparecer, mas por via das dúvidas
    if (!PERMITIDAS_BACK.some((prefixo) => mod.startsWith(prefixo))) fora.push(mod);
  }
  if (fora.length > 0) {
    avisar(`Módulo(s) Go fora do allowlist (padroes/01-stack-permitida.md), confirme antes de manter: ${fora.join(", ")}`);
  }
}

(async () => {
  const entrada = await lerEntrada();
  const arquivo = entrada?.tool_input?.file_path;
  if (!arquivo || !fs.existsSync(arquivo)) process.exit(0);

  const base = path.basename(arquivo);
  if (base === "package.json") checarPackageJson(arquivo);
  else if (base === "go.mod") checarGoMod(arquivo);

  process.exit(0);
})();
