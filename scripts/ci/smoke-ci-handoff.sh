#!/usr/bin/env sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
REPO_ROOT="$(CDPATH= cd -- "${SCRIPT_DIR}/../.." && pwd)"

MODE="${1:-native}"
ARTIFACT_SOURCE="${2:-examples/outputs/snapshot.sample.json}"
SMOKE_ROOT_REL=".tools/ci-template-smoke/${MODE}"
ARTIFACT_REL="${SMOKE_ROOT_REL}/artifacts/input.json"
OUT_REL="${SMOKE_ROOT_REL}/out"

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

require_file() {
  path="$1"
  label="$2"

  if [ ! -f "${path}" ]; then
    echo "Missing ${label}: ${path}" >&2
    exit 1
  fi
}

docker_host_mount_path() {
  path="$1"

  if command -v cygpath >/dev/null 2>&1; then
    cygpath -w "${path}"
    return 0
  fi

  printf '%s\n' "${path}"
}

cd "${REPO_ROOT}"

rm -rf "${SMOKE_ROOT_REL}"
mkdir -p "${SMOKE_ROOT_REL}/artifacts" "${OUT_REL}"
cp "${ARTIFACT_SOURCE}" "${ARTIFACT_REL}"

case "${MODE}" in
  native)
    "${GO_BIN}" run ./cmd/sheaft run \
      --model "${ARTIFACT_REL}" \
      --analysis configs/analysis.example.yaml \
      --out-dir "${OUT_REL}"
    ;;
  docker)
    DOCKER_MOUNT_PATH="$(docker_host_mount_path "${REPO_ROOT}")"
    docker build -f build/Dockerfile -t sheaft:ci-smoke .
    MSYS_NO_PATHCONV=1 docker run --rm -v "${DOCKER_MOUNT_PATH}:/workspace" -w /workspace sheaft:ci-smoke run \
      --model "${ARTIFACT_REL}" \
      --analysis configs/analysis.example.yaml \
      --out-dir "${OUT_REL}"
    ;;
  *)
    echo "Unsupported smoke mode: ${MODE}. Use 'native' or 'docker'." >&2
    exit 1
    ;;
esac

require_file "${ARTIFACT_REL}" "handoff artifact"
require_file "${OUT_REL}/model.json" "mirrored model artifact"
require_file "${OUT_REL}/report.json" "posture report"
require_file "${OUT_REL}/summary.md" "summary markdown"

echo "CI handoff smoke passed (${MODE})"
echo "Workspace: ${SMOKE_ROOT_REL}"
