#!/usr/bin/env sh
set -eu

REPO_ROOT="${1:-.}"
MANIFEST_PATH="${REPO_ROOT}/compatibility-manifest.json"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

if [ ! -f "${MANIFEST_PATH}" ]; then
  echo "Compatibility manifest is missing: ${MANIFEST_PATH}" >&2
  exit 1
fi

(
  cd "${REPO_ROOT}"
  go run ./cmd/releasectl compatibility-manifest --out "${TMP_DIR}/compatibility-manifest.json"
  go run ./cmd/releasectl validate-compatibility-manifest --manifest "${MANIFEST_PATH}"
)

if ! cmp -s "${MANIFEST_PATH}" "${TMP_DIR}/compatibility-manifest.json"; then
  echo "compatibility-manifest.json is stale; regenerate it from current contract pins" >&2
  exit 1
fi

echo "Compatibility manifest check passed"
