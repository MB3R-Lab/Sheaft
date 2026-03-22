#!/bin/sh
set -eu

out_root="${1:-.tmp/smoke-examples}"
bin_path="${2:-./sheaft}"
serve_log="${out_root}/serve.log"
serve_port="${SHEAFT_SMOKE_PORT:-18080}"
repo_root="$(pwd)"
serve_config="${out_root}/sheaft.serve.yaml"

rm -rf "${out_root}"
mkdir -p "${out_root}/policy" "${out_root}/analysis"

cat >"${serve_config}" <<EOF
schema_version: "1.0"
listen: ":${serve_port}"

artifact:
  path: "${repo_root}/examples/outputs/snapshot.sample.json"
  mode: file

analysis_file: "${repo_root}/configs/analysis.example.yaml"

poll_interval: 30s
watch_fs: true
watch_polling: true

history:
  max_items: 20
  disk_dir: "${repo_root}/.sheaft/history"
EOF

"${bin_path}" run \
  --model examples/outputs/model.sample.json \
  --policy configs/gate.policy.example.yaml \
  --out-dir "${out_root}/policy" \
  --seed 42

"${bin_path}" run \
  --model examples/outputs/snapshot.sample.json \
  --analysis configs/analysis.example.yaml \
  --out-dir "${out_root}/analysis"

"${bin_path}" serve --config "${serve_config}" >"${serve_log}" 2>&1 &
pid=$!

cleanup() {
  if kill -0 "${pid}" >/dev/null 2>&1; then
    kill "${pid}" >/dev/null 2>&1 || true
    wait "${pid}" >/dev/null 2>&1 || true
  fi
}

trap cleanup EXIT INT TERM

attempt=0
until curl -fsS "http://127.0.0.1:${serve_port}/healthz" >/dev/null 2>&1; do
  attempt=$((attempt + 1))
  if [ "${attempt}" -ge 20 ]; then
    cat "${serve_log}" >&2
    echo "sheaft serve did not become reachable on :${serve_port}" >&2
    exit 1
  fi
  sleep 1
done

curl -fsS "http://127.0.0.1:${serve_port}/readyz" >"${out_root}/readyz.json"
if ! grep -q '"ready":true' "${out_root}/readyz.json"; then
  cat "${out_root}/readyz.json" >&2
  cat "${serve_log}" >&2
  echo "sheaft serve started but did not reach ready=true" >&2
  exit 1
fi

curl -fsS "http://127.0.0.1:${serve_port}/current-report" >"${out_root}/current-report.json"
