# Stage 1: builda o bundle estático com pnpm.
# node:22-alpine porque o pnpm pinado abaixo exige Node >= 22.13 (pnpm 11+,
# que traz minimumReleaseAge por padrão — ver 01-stack-permitida.md).
FROM node:22-alpine AS build
# Cache do corepack DENTRO da imagem (e não no home do root), pra o usuário
# non-root achar o pnpm em runtime sem precisar baixar de novo.
ENV COREPACK_HOME=/app/.corepack
WORKDIR /app
# corepack enable precisa de root (cria os atalhos em /usr/local/bin).
RUN corepack enable
# Daqui pra baixo tudo roda como o usuário non-root `node` (uid 1000, já existe
# na imagem): install e build. Se o build for comprometido (dependência
# maliciosa), não é root dentro do container.
RUN chown node:node /app
USER node
# Pina a versão do pnpm (mesma do packageManager em package.json). Sem isso o
# corepack baixa a "latest", que exige Node mais novo e quebra o build.
RUN corepack prepare pnpm@11.18.0 --activate
COPY --chown=node:node .npmrc package.json pnpm-workspace.yaml pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY --chown=node:node . .
RUN pnpm run build

# Stage 2: serve o build com nginx em vez do "vite preview" (servidor de dev,
# não pensado pra produção — sem isso ele sobe sem endurecimento nenhum).
# nginxinc/nginx-unprivileged já roda non-root de fábrica (uid 101, escuta em
# 8080 — non-root não pode bindar <1024), então nenhum `USER`/`chown` extra é
# necessário aqui. O proxy de /api pro backend está no nginx.conf.template
# (envsubst de API_TARGET em runtime — ver padroes/04-docker-deploy.md).
FROM nginxinc/nginx-unprivileged:1.27-alpine
COPY --chown=nginx:nginx nginx.conf.template /etc/nginx/templates/default.conf.template
COPY --chown=nginx:nginx nginx-headers-seguranca.conf /etc/nginx/headers-seguranca.conf
COPY --chown=nginx:nginx --from=build /app/dist /usr/share/nginx/html
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --retries=5 --start-period=10s \
  CMD wget -q -O /dev/null http://localhost:8080/ || exit 1
