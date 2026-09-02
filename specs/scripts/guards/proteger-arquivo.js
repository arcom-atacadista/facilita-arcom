#!/usr/bin/env node
// Guard (PreToolUse, matcher Edit|Write|MultiEdit): recusa editar arquivos
// de infraestrutura da plataforma. Ver system-design/padroes/00-visao-geral.md e
// system-design/padroes/11-guards.md. Node (não jq/bash) porque é a única ferramenta
// garantida em qualquer máquina rodando este template (o próprio Claude
// Code roda sobre Node).
"use strict";

const fs = require("fs");

// Presença deste arquivo = projeto legado (drop-in num repo que já existia
// com stack própria, possivelmente diferente da ARCOM). Ver CLAUDE.md.
const PROJETO_LEGADO = fs.existsSync(".claude/PROJETO-EXISTENTE");

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

// Arquivos que nunca devem ser editados à mão, qualquer ferramenta.
const SEMPRE_PROTEGIDOS = [
  /(^|\/)\.npmrc$/,
  /(^|\/)pnpm-lock\.yaml$/,
  /(^|\/)go\.sum$/,
  /(^|\/)Dockerfile$/,
  /(^|\/)nginx\.conf\.template$/,
  /(^|\/)nginx-headers-seguranca\.conf$/,
  /(^|\/)docker-compose\.ya?ml$/,
  /(^|\/)docker-compose\.prod\.ya?ml$/,
];

// A pasta system-design/ inteira é o Padrão ARCOM — ninguém edita isso por
// dentro de uma sessão do Claude Code, nem admin (mudança de padrão é PR
// revisado no repo do template, não edição ad-hoc num projeto consumidor).
const PASTA_TRAVADA = /(^|\/)system-design\//;

// Trechos do vite.config.ts que são da plataforma — editar o resto (ex.:
// bloco manifest do PWA) continua permitido.
const TRECHOS_PROTEGIDOS_VITE = /proxy|cors|xfwd/i;

(async () => {
  const entrada = await lerEntrada();
  const arquivo = entrada?.tool_input?.file_path;
  const ferramenta = entrada?.tool_name;
  if (!arquivo) process.exit(0);

  if (PASTA_TRAVADA.test(arquivo)) {
    negar(
      `${arquivo} está em system-design/ — é o Padrão ARCOM, não código deste projeto. ` +
        "Não é editável por aqui; mudança de padrão é feita no repositório-template.",
    );
  }

  // As proteções abaixo assumem que docker-compose/.npmrc/vite.config.ts são
  // o scaffolding gerado pela ARCOM. Num projeto legado (drop-in) isso não é
  // garantido — pode ser infra própria do time, legítima de editar — então
  // essas duas checagens ficam de fora; só a trava de system-design/ acima
  // continua valendo sempre.
  if (PROJETO_LEGADO) process.exit(0);

  if (SEMPRE_PROTEGIDOS.some((re) => re.test(arquivo))) {
    negar(
      `${arquivo} é infraestrutura da plataforma (system-design/padroes/00-visao-geral.md) — não editar à mão. ` +
        "Dependência muda via pnpm/go mod; mudança de infra de verdade é decisão da equipe de infra.",
    );
  }

  if (/vite\.config\.ts$/.test(arquivo)) {
    if (ferramenta === "Write") {
      negar(
        "vite.config.ts: use uma edição pontual (Edit), não reescreva o arquivo inteiro — " +
          "o proxy/CORS/headers são da plataforma (system-design/padroes/02-frontend.md).",
      );
    }
    const trechos = [
      entrada?.tool_input?.old_string,
      entrada?.tool_input?.new_string,
      ...(Array.isArray(entrada?.tool_input?.edits)
        ? entrada.tool_input.edits.flatMap((e) => [e.old_string, e.new_string])
        : []),
    ]
      .filter(Boolean)
      .join("\n");
    if (TRECHOS_PROTEGIDOS_VITE.test(trechos)) {
      negar(
        "Esse trecho de vite.config.ts é o proxy/CORS/headers de segurança da plataforma — não mexer " +
          "(system-design/padroes/02-frontend.md). Editar o bloco `manifest` do PWA continua permitido.",
      );
    }
  }

  process.exit(0);
})();
