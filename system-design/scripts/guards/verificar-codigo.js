#!/usr/bin/env node
// Guard (PostToolUse, matcher Edit|Write): grep de padrões inseguros
// conhecidos, por linguagem — a lista curta e rápida; a varredura completa é
// scripts/verificar.sh. Ver padroes/11-guards.md e .claude/skills/seguranca.
"use strict";
const fs = require("fs");

// Remove comentário antes de checar — sem isso, o guard acusa o próprio
// comentário que EXPLICA a regra (ex.: "nunca use X") como se fosse a
// violação. Aproximado (não entende string com "//" dentro), mas suficiente
// pra um guard rápido — a varredura completa fica com ferramenta de verdade.
function semComentarios(conteudo) {
  return conteudo.replace(/\/\*[\s\S]*?\*\//g, "").replace(/\/\/.*$/gm, "");
}

function avisar(arquivo, achados) {
  process.stdout.write(
    JSON.stringify({
      decision: "block",
      reason: `Guard de segurança em ${arquivo}:\n- ${achados.join("\n- ")}`,
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

function checarGo(conteudo) {
  const achados = [];
  if (/\b(Raw|Exec)\(/.test(conteudo) && (/\+\s*[a-zA-Z_]/.test(conteudo) || /fmt\.Sprintf/.test(conteudo))) {
    achados.push("SQL possivelmente concatenado (Raw/Exec perto de '+' ou fmt.Sprintf) — use parâmetro '?' (skill seguranca §1).");
  }
  if (/math\/rand/.test(conteudo) && /token|senha|password|secret/i.test(conteudo)) {
    achados.push("math/rand perto de token/senha/secret — use crypto/rand (skill seguranca §3).");
  }
  if (/==\s*token|token\s*==/.test(conteudo) && !/subtle\.ConstantTimeCompare/.test(conteudo)) {
    achados.push("Comparação de token com '==' — use crypto/subtle.ConstantTimeCompare (skill seguranca §3).");
  }
  if (/http\.ListenAndServe\(/.test(conteudo)) {
    achados.push("http.ListenAndServe direto — use http.Server com timeouts + graceful shutdown (padroes/03-backend.md).");
  }
  if (/\.Updates\(/.test(conteudo) && !/\.Select\(/.test(conteudo)) {
    achados.push("Updates(...) sem Select(...) — risco de mass assignment (skill seguranca §4).");
  }
  if (/err\.Error\(\)/.test(conteudo) && /(w\.Write|Encode\(w)/.test(conteudo)) {
    achados.push("err.Error() perto de escrita na resposta HTTP — nunca vaze erro interno (padroes/09-contrato-api.md).");
  }
  if (/http\.Client\{\}/.test(conteudo) && !/Timeout:/.test(conteudo)) {
    achados.push("http.Client{} sem Timeout — pode travar goroutine indefinidamente (skill seguranca §8).");
  }
  return achados;
}

function checarFrontend(conteudo) {
  const achados = [];
  if (/dangerouslySetInnerHTML/.test(conteudo)) {
    achados.push("dangerouslySetInnerHTML — precisa de sanitização única (dompurify), ver skill seguranca §2.");
  }
  if (/\b(localStorage|sessionStorage)\b/.test(conteudo)) {
    achados.push("localStorage/sessionStorage — sessão é cookie HttpOnly, nunca guarde nada aqui (padroes/02-frontend.md).");
  }
  if (/https?:\/\/(localhost|backend):3000/.test(conteudo)) {
    achados.push("URL absoluta de servidor — use /api/v1/... (o proxy do Vite cuida do resto).");
  }
  return achados;
}

(async () => {
  const entrada = await lerEntrada();
  const arquivo = entrada?.tool_input?.file_path;
  if (!arquivo || !fs.existsSync(arquivo)) process.exit(0);

  let conteudo = "";
  try {
    conteudo = semComentarios(fs.readFileSync(arquivo, "utf8"));
  } catch {
    process.exit(0);
  }

  let achados = [];
  if (/\.go$/.test(arquivo)) achados = checarGo(conteudo);
  else if (/\.tsx?$/.test(arquivo)) achados = checarFrontend(conteudo);

  if (achados.length > 0) avisar(arquivo, achados);
  process.exit(0);
})();
