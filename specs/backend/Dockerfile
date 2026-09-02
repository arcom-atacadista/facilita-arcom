# ---- build ----
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# -s -w tira símbolo de debug/DWARF (binário menor); -trimpath tira caminho
# absoluto do host do binário final. Builda os dois binários: o servidor e o
# migrate (roda como serviço separado no compose — ver 04-docker-deploy.md).
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate

# ---- runtime ----
FROM alpine:3.20
RUN apk add --no-cache ca-certificates wget poppler-utils \
 && adduser -D -u 10001 appuser
WORKDIR /app
COPY --from=build /out/server /out/migrate ./
USER appuser
EXPOSE 3000
HEALTHCHECK --interval=10s --timeout=3s --start-period=20s --retries=5 \
  CMD wget -q -O /dev/null http://localhost:3000/api/ready || exit 1
CMD ["./server"]
