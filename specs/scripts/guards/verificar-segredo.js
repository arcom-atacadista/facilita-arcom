#!/usr/bin/env node
// Guard (PostToolUse, matcher Edit|Write): acusa segredo aparecendo onde não
// deveria. Não bloqueia a escrita (já aconteceu) — devolve `decision:block`,
// que o Claude Code mostra pro modelo como feedback forte pra corrigir na
// mesma resposta. Ver padroes/11-guards.md.
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

const PADRAO_VITE_SECRET = /VITE_[A-Z0-9_]*(KEY|SECRET|TOKEN|PASSWORD)[A-Z0-9_]*\s*=/;
const PADRAO_ENV_SECRET = /^[A-Z0-9_]*(KEY|SECRET|TOKEN|PASSWORD)[A-Z0-9_]*\s*=\s*\S+/m;
const PLACEHOLDERS_OBVIOS = /troque-em-producao|dev-secret|exemplo|seu-valor-aqui|change-?me/i;

(async () => {
  const entrada = await lerEntrada();
  const arquivo = entrada?.tool_input?.file_path;
  if (!arquivo || !fs.existsSync(arquivo)) process.exit(0);

  let conteudo = "";
  try {
    conteudo = fs.readFileSync(arquivo, "utf8");
  } catch {
    process.exit(0);
  }

  if (PADRAO_VITE_SECRET.test(conteudo)) {
    avisar(
      `${arquivo}: variável VITE_*_KEY/SECRET/TOKEN/PASSWORD vai pro bundle público do frontend — ` +
        "CRÍTICO (padroes/01-stack-permitida.md). Segredo nunca é VITE_*, mova pro backend.",
    );
  }

  const base = path.basename(arquivo);
  const ehEnv = /^\.env(\..+)?$/.test(base) && base !== ".env.example";
  if (ehEnv && PADRAO_ENV_SECRET.test(conteudo) && !PLACEHOLDERS_OBVIOS.test(conteudo)) {
    avisar(
      `${arquivo} parece ter segredo real — confirme que está no .gitignore e que não é o mesmo valor de outro ambiente.`,
    );
  }

  process.exit(0);
})();
