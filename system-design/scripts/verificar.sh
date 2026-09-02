#!/usr/bin/env bash
# Varredura completa de qualidade/segurança — roda sob demanda (não a cada
# edição, ao contrário dos guards em .claude/settings.json). Ferramenta
# ausente vira aviso, nunca trava quem não tem tudo instalado localmente.
# Ver padroes/11-guards.md.
set -uo pipefail

raiz="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
erros=0

secao() { printf '\n== %s ==\n' "$1"; }

secao "Backend — go vet"
if [ -d "$raiz/backend" ]; then
  (cd "$raiz/backend" && go vet ./...) || erros=1
else
  echo "  (sem backend/ neste projeto — pulei)"
fi

if [ -d "$raiz/backend" ]; then
  secao "Backend — golangci-lint"
  if command -v golangci-lint >/dev/null 2>&1; then
    (cd "$raiz/backend" && golangci-lint run ./...) || erros=1
  else
    echo "  (aviso: golangci-lint não instalado — pulei esta checagem)"
  fi

  secao "Backend — govulncheck"
  if command -v govulncheck >/dev/null 2>&1; then
    (cd "$raiz/backend" && govulncheck ./...) || echo "  (vulnerabilidade encontrada — ver detalhes acima)"
  else
    echo "  (aviso: govulncheck não instalado — pulei esta checagem)"
  fi

  secao "Backend — testes (-race)"
  (cd "$raiz/backend" && go test -race ./...) || erros=1
fi

if [ -d "$raiz/frontend" ]; then
  secao "Frontend — eslint"
  (cd "$raiz/frontend" && pnpm lint) || erros=1

  secao "Frontend — typecheck"
  (cd "$raiz/frontend" && pnpm typecheck) || erros=1
else
  secao "Frontend"
  echo "  (sem frontend/ neste projeto — pulei)"
fi

secao "Segredo em .env versionado"
if git -C "$raiz" ls-files 2>/dev/null | grep -E '(^|/)\.env(\..+)?$' | grep -v '\.env\.example$' | grep -q .; then
  echo "  ALERTA: existe .env versionado no git (deveria estar no .gitignore)."
  erros=1
else
  echo "  OK — nenhum .env rastreado pelo git."
fi

secao "Resultado"
if [ "$erros" -eq 0 ]; then
  echo "Tudo certo."
else
  echo "Encontrei problema(s) acima — corrija antes de considerar pronto."
fi
exit "$erros"
