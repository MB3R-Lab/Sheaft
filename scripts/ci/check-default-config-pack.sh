#!/usr/bin/env sh
set -eu

REPO_ROOT="${1:-.}"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

(
  cd "${REPO_ROOT}"
  go run ./cmd/releasectl package-default-config-pack \
    --version 0.0.0-dev \
    --out "${TMP_DIR}/default-pack.tar.gz" \
    --metadata-out "${TMP_DIR}/default-pack.json"
)

test -s "${TMP_DIR}/default-pack.tar.gz"
test -s "${TMP_DIR}/default-pack.json"

echo "Default config pack generation check passed"
