#!/usr/bin/env sh
set -eu

REGISTRY_NAME="${REGISTRY_NAME:-sheaft-local-registry}"
REGISTRY_PORT="${REGISTRY_PORT:-5000}"

if docker ps --format '{{.Names}}' | grep -qx "${REGISTRY_NAME}"; then
  exit 0
fi

if docker ps -a --format '{{.Names}}' | grep -qx "${REGISTRY_NAME}"; then
  docker start "${REGISTRY_NAME}" >/dev/null
  exit 0
fi

docker run -d -p "${REGISTRY_PORT}:5000" --restart unless-stopped --name "${REGISTRY_NAME}" registry:2 >/dev/null
