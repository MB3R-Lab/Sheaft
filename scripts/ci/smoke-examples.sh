#!/bin/sh
set -eu

out_root="${1:-.tmp/smoke-examples}"
bin_path="${2:-./sheaft}"
serve_log="${out_root}/serve.log"

rm -rf "${out_root}"
mkdir -p "${out_root}/policy" "${out_root}/analysis"

"${bin_path}" run \
  --model examples/outputs/model.sample.json \
  --policy configs/gate.policy.example.yaml \
  --out-dir "${out_root}/policy" \
  --seed 42

"${bin_path}" run \
  --model examples/outputs/snapshot.sample.json \
  --analysis configs/analysis.example.yaml \
  --out-dir "${out_root}/analysis"

"${bin_path}" serve --config configs/sheaft.example.yaml >"${serve_log}" 2>&1 &
pid=$!

cleanup() {
  if kill -0 "${pid}" >/dev/null 2>&1; then
    kill "${pid}" >/dev/null 2>&1 || true
    wait "${pid}" >/dev/null 2>&1 || true
  fi
}

trap cleanup EXIT INT TERM

attempt=0
until curl -fsS http://127.0.0.1:8080/healthz >/dev/null 2>&1; do
  attempt=$((attempt + 1))
  if [ "${attempt}" -ge 20 ]; then
    cat "${serve_log}" >&2
    echo "sheaft serve did not become reachable on :8080" >&2
    exit 1
  fi
  sleep 1
done

curl -fsS http://127.0.0.1:8080/readyz >"${out_root}/readyz.json"
if ! grep -q '"ready":true' "${out_root}/readyz.json"; then
  cat "${out_root}/readyz.json" >&2
  cat "${serve_log}" >&2
  echo "sheaft serve started but did not reach ready=true" >&2
  exit 1
fi

curl -fsS http://127.0.0.1:8080/current-report >"${out_root}/current-report.json"
