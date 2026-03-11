#!/usr/bin/env sh
set -eu

REPO_ROOT="${1:-.}"
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

cd "${REPO_ROOT}"

"${GO_BIN}" run ./cmd/sheaft run \
  --model examples/outputs/model.sample.json \
  --analysis configs/analysis.example.yaml \
  --contract-policy configs/contract-policy.example.yaml \
  --out-dir "${TMP_DIR}/allowed" >/dev/null

"${GO_BIN}" run ./cmd/sheaft run \
  --model examples/outputs/snapshot.sample.json \
  --analysis configs/analysis.example.yaml \
  --contract-policy configs/contract-policy.deprecated.example.yaml \
  --out-dir "${TMP_DIR}/deprecated" >"${TMP_DIR}/deprecated.stdout"

grep -Fq '"contract_policy": {' "${TMP_DIR}/deprecated/report.json"
grep -Fq '"status": "deprecated"' "${TMP_DIR}/deprecated/report.json"
grep -Fq '"action": "warn"' "${TMP_DIR}/deprecated/report.json"
grep -Fq 'contract policy: deprecated (warn)' "${TMP_DIR}/deprecated.stdout"

cat >"${TMP_DIR}/fail-policy.yaml" <<'EOF'
deprecated_action: fail
deprecated_contracts:
  - kind: snapshot
    name: io.mb3r.bering.snapshot
    versions:
      - "1.0.0"
EOF

if "${GO_BIN}" run ./cmd/sheaft run \
  --model examples/outputs/snapshot.sample.json \
  --analysis configs/analysis.example.yaml \
  --contract-policy "${TMP_DIR}/fail-policy.yaml" \
  --out-dir "${TMP_DIR}/fail" >"${TMP_DIR}/fail.stdout" 2>"${TMP_DIR}/fail.stderr"; then
  echo "Expected fail contract policy to stop execution" >&2
  exit 1
fi

grep -Fq 'contract policy:' "${TMP_DIR}/fail.stderr"

echo "Contract policy example check passed"
