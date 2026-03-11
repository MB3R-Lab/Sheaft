#!/usr/bin/env sh
set -eu

REPO_ROOT="${1:-.}"
BASE_REF="${2:-}"
CONTRACT_FILE="${REPO_ROOT}/internal/modelcontract/contract.go"
MATRIX_FILE="${REPO_ROOT}/docs/compatibility-matrix.md"
README_FILE="${REPO_ROOT}/README.md"
HELPER_FILE="${REPO_ROOT}/scripts/ci/contract-constants.sh"

if [ ! -f "${CONTRACT_FILE}" ]; then
  echo "Contract file is missing: ${CONTRACT_FILE}" >&2
  exit 1
fi

if [ ! -f "${MATRIX_FILE}" ]; then
  echo "Compatibility matrix is missing: ${MATRIX_FILE}" >&2
  exit 1
fi

if [ ! -f "${README_FILE}" ]; then
  echo "README is missing: ${README_FILE}" >&2
  exit 1
fi

if [ ! -f "${HELPER_FILE}" ]; then
  echo "Helper file is missing: ${HELPER_FILE}" >&2
  exit 1
fi

# shellcheck source=scripts/ci/contract-constants.sh
. "${HELPER_FILE}"

require_in_file() {
  needle="$1"
  file="$2"
  label="$3"

  if ! grep -Fq "${needle}" "${file}"; then
    echo "Missing ${label} entry in ${file}: ${needle}" >&2
    exit 1
  fi
}

MODEL_NAME="$(extract_const BeringModelV100Name)"
MODEL_VERSION="$(extract_const BeringModelV100Version)"
MODEL_URI="$(extract_const BeringModelV100URI)"
MODEL_DIGEST="$(extract_const BeringModelV100Digest)"

SNAPSHOT_NAME="$(extract_const BeringSnapshotV100Name)"
SNAPSHOT_VERSION="$(extract_const BeringSnapshotV100Version)"
SNAPSHOT_URI="$(extract_const BeringSnapshotV100URI)"
SNAPSHOT_DIGEST="$(extract_const BeringSnapshotV100Digest)"

require_in_file "${MODEL_NAME}@${MODEL_VERSION}" "${MATRIX_FILE}" "model contract"
require_in_file "${MODEL_URI}" "${MATRIX_FILE}" "model uri"
require_in_file "${MODEL_DIGEST}" "${MATRIX_FILE}" "model digest"
require_in_file "${SNAPSHOT_NAME}@${SNAPSHOT_VERSION}" "${MATRIX_FILE}" "snapshot contract"
require_in_file "${SNAPSHOT_URI}" "${MATRIX_FILE}" "snapshot uri"
require_in_file "${SNAPSHOT_DIGEST}" "${MATRIX_FILE}" "snapshot digest"
require_in_file "docs/compatibility-matrix.md" "${README_FILE}" "README reference"

if [ -n "${BASE_REF}" ]; then
  if ! git rev-parse --verify "${BASE_REF}" >/dev/null 2>&1; then
    echo "Base ref is not available locally: ${BASE_REF}" >&2
    exit 1
  fi

  contract_changes="$(git diff --name-only "${BASE_REF}"...HEAD -- \
    internal/modelcontract/contract.go \
    internal/modelcontract/schema/model.schema.json \
    api/schema/model.schema.json \
    api/schema/snapshot.schema.json)"
  matrix_changes="$(git diff --name-only "${BASE_REF}"...HEAD -- docs/compatibility-matrix.md)"

  if [ -n "${contract_changes}" ] && [ -z "${matrix_changes}" ]; then
    echo "Contract pin files changed without updating docs/compatibility-matrix.md" >&2
    echo "Changed contract files:" >&2
    printf '%s\n' "${contract_changes}" >&2
    exit 1
  fi
fi

echo "Compatibility matrix check passed"
echo "Model:    ${MODEL_NAME}@${MODEL_VERSION}"
echo "Snapshot: ${SNAPSHOT_NAME}@${SNAPSHOT_VERSION}"
