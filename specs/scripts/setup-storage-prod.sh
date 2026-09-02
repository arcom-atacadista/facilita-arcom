#!/usr/bin/env bash
# Cria um teto de disco RÍGIDO (100%) pros dados de Postgres (e Redis, se o
# projeto tiver) em produção.
#
# Por quê: o Docker Compose NÃO limita tamanho de volume (deploy.resources só
# cobre CPU/memória). Aqui cada pasta de dados ganha um filesystem próprio de
# tamanho fixo (imagem loopback ext4). Quando enche, o kernel barra a escrita —
# não passa do tamanho, garantido. É o teto "100%" que o yaml sozinho não dá.
#
# Rode UMA VEZ no host de produção, como root, ANTES do primeiro `docker compose up`:
#   sudo ./scripts/setup-storage-prod.sh
#
# Ajuste os tamanhos por variável de ambiente se quiser:
#   sudo PG_SIZE=10G REDIS_SIZE=2G ./scripts/setup-storage-prod.sh
#
# Redis é opt-in (ver 01-stack-permitida.md) — por padrão este script NÃO cria
# storage pra ele. Só crie se o projeto REALMENTE tem o serviço redis no
# docker-compose.yml: rode com COM_REDIS=1.
# Idempotente: se a imagem/mount já existem, não recria nem apaga dado.
set -euo pipefail

DATA_DIR="${DATA_DIR:-/array/meu-projeto/data}"   # mesmo default do docker-compose.prod.yml
IMG_DIR="${IMG_DIR:-/array/meu-projeto/img}"      # onde ficam os arquivos-imagem
PG_SIZE="${PG_SIZE:-10G}"                      # teto do Postgres
REDIS_SIZE="${REDIS_SIZE:-2G}"                 # teto do Redis (disco; a RAM é o maxmemory), só se COM_REDIS=1

if [ "$(id -u)" -ne 0 ]; then
  echo "Precisa rodar como root (use sudo)." >&2
  exit 1
fi

criar_fs() {
  local nome="$1" tamanho="$2" uid="$3" gid="$4"
  local img="$IMG_DIR/$nome.img"
  local mnt="$DATA_DIR/$nome"

  mkdir -p "$IMG_DIR" "$mnt"

  if [ ! -f "$img" ]; then
    echo "criando imagem $img ($tamanho)..."
    fallocate -l "$tamanho" "$img" 2>/dev/null || truncate -s "$tamanho" "$img"
    mkfs.ext4 -q -F "$img"
  else
    echo "imagem $img já existe — mantendo."
  fi

  if ! mountpoint -q "$mnt"; then
    echo "montando $mnt..."
    mount -o loop "$img" "$mnt"
  fi

  chown -R "$uid:$gid" "$mnt"

  if ! grep -qs " $mnt " /etc/fstab; then
    echo "$img $mnt ext4 loop 0 0" >> /etc/fstab
    echo "adicionado ao /etc/fstab."
  fi
}

# UIDs das imagens oficiais: postgres:16-alpine = 70:70, redis:7-alpine = 999:1000
criar_fs postgres "$PG_SIZE" 70 70
if [ "${COM_REDIS:-0}" = "1" ]; then
  criar_fs redis "$REDIS_SIZE" 999 1000
fi

echo ""
echo "OK — teto rígido aplicado em $DATA_DIR."
