#!/usr/bin/env sh
set -eu

REPO_ROOT="${1:-.}"
MANIFEST_PATH="${REPO_ROOT}/compatibility-manifest.json"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

find_go() {
  if command -v go >/dev/null 2>&1; then
    command -v go
    return 0
  fi

  if [ -x "/c/Program Files/Go/bin/go.exe" ]; then
    printf '%s\n' "/c/Program Files/Go/bin/go.exe"
    return 0
  fi

  echo "Go executable not found. Set GO_BIN or add go to PATH." >&2
  exit 1
}

GO_BIN="${GO_BIN:-$(find_go)}"

if [ ! -f "${MANIFEST_PATH}" ]; then
  echo "Compatibility manifest is missing: ${MANIFEST_PATH}" >&2
  exit 1
fi

(
  cd "${REPO_ROOT}"
  "${GO_BIN}" run ./cmd/releasectl compatibility-manifest --out "${TMP_DIR}/compatibility-manifest.json"
  "${GO_BIN}" run ./cmd/releasectl validate-compatibility-manifest --manifest "${MANIFEST_PATH}"
)

if ! cmp -s "${MANIFEST_PATH}" "${TMP_DIR}/compatibility-manifest.json"; then
  echo "compatibility-manifest.json is stale; regenerate it from current contract pins" >&2
  exit 1
fi

echo "Compatibility manifest check passed"
