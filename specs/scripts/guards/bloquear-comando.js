#!/usr/bin/env node
// Guard (PreToolUse, matcher Bash): recusa comandos que contrariam o Padrão
// ARCOM antes de rodar. Ver padroes/01-stack-permitida.md e
// padroes/11-guards.md.
"use strict";

function negar(motivo) {
  process.stdout.write(
    JSON.stringify({
      hookSpecificOutput: {
        hookEventName: "PreToolUse",
        permissionDecision: "deny",
        permissionDecisionReason: motivo,
      },
    }),
  );
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

const REGRAS = [
  {
    re: /(^|[;&|]\s*)(npm|yarn)\s+(install|add|i\b|remove|uninstall)/,
    motivo: "Só pnpm é permitido no frontend (padroes/01-stack-permitida.md) — use 'pnpm install'/'pnpm add'.",
  },
  {
    re: /curl[^|]*\|\s*(sh|bash)\b/,
    motivo: "Não execute script baixado direto (curl | sh) sem revisar antes — risco de supply chain.",
  },
  {
    re: /\bchmod\s+777\b/,
    motivo: "chmod 777 nunca é a permissão certa — use o mínimo necessário.",
  },
  {
    re: /\bgit\s+add\b.*\.env(\.prod)?(\s|$)/,
    motivo: "Não adicione .env/.env.prod ao git — são segredo (padroes/01-stack-permitida.md).",
  },
  {
    re: /--no-verify\b/,
    motivo: "Não pule hooks do git (--no-verify) sem pedir confirmação explícita ao usuário.",
  },
];

(async () => {
  const entrada = await lerEntrada();
  const comando = entrada?.tool_input?.command;
  if (!comando) process.exit(0);

  for (const regra of REGRAS) {
    if (regra.re.test(comando)) negar(regra.motivo);
  }

  process.exit(0);
})();
