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

require_contract_in_matrix() {
  name_const="$1"
  version_const="$2"
  uri_const="$3"
  digest_const="$4"
  label="$5"

  name="$(extract_const "${name_const}")"
  version="$(extract_const "${version_const}")"
  uri="$(extract_const "${uri_const}")"
  digest="$(extract_const "${digest_const}")"

  require_in_file "${name}@${version}" "${MATRIX_FILE}" "${label} contract"
  require_in_file "${uri}" "${MATRIX_FILE}" "${label} uri"
  require_in_file "${digest}" "${MATRIX_FILE}" "${label} digest"
}

require_contract_in_matrix \
  "BeringModelV100Name" \
  "BeringModelV100Version" \
  "BeringModelV100URI" \
  "BeringModelV100Digest" \
  "model v1.0.0"

require_contract_in_matrix \
  "BeringSnapshotV100Name" \
  "BeringSnapshotV100Version" \
  "BeringSnapshotV100URI" \
  "BeringSnapshotV100Digest" \
  "snapshot v1.0.0"

require_contract_in_matrix \
  "BeringModelV110Name" \
  "BeringModelV110Version" \
  "BeringModelV110URI" \
  "BeringModelV110Digest" \
  "model v1.1.0"

require_contract_in_matrix \
  "BeringSnapshotV110Name" \
  "BeringSnapshotV110Version" \
  "BeringSnapshotV110URI" \
  "BeringSnapshotV110Digest" \
  "snapshot v1.1.0"

require_in_file "docs/compatibility-matrix.md" "${README_FILE}" "README reference"

if [ -n "${BASE_REF}" ]; then
  if ! git rev-parse --verify "${BASE_REF}" >/dev/null 2>&1; then
    echo "Base ref is not available locally: ${BASE_REF}" >&2
    exit 1
  fi

  contract_changes="$(git diff --name-only "${BASE_REF}"...HEAD -- \
    internal/modelcontract/contract.go \
    internal/modelcontract/schema/model.schema.json \
    internal/modelcontract/schema/model.v1.0.0.schema.json \
    internal/modelcontract/schema/model.v1.1.0.schema.json \
    internal/modelcontract/schema/snapshot.schema.json \
    internal/modelcontract/schema/snapshot.v1.0.0.schema.json \
    internal/modelcontract/schema/snapshot.v1.1.0.schema.json \
    api/schema/model.schema.json \
    api/schema/model.v1.0.0.schema.json \
    api/schema/model.v1.1.0.schema.json \
    api/schema/snapshot.schema.json \
    api/schema/snapshot.v1.0.0.schema.json \
    api/schema/snapshot.v1.1.0.schema.json)"
  matrix_changes="$(git diff --name-only "${BASE_REF}"...HEAD -- docs/compatibility-matrix.md)"

  if [ -n "${contract_changes}" ] && [ -z "${matrix_changes}" ]; then
    echo "Contract pin files changed without updating docs/compatibility-matrix.md" >&2
    echo "Changed contract files:" >&2
    printf '%s\n' "${contract_changes}" >&2
    exit 1
  fi
fi

echo "Compatibility matrix check passed"
echo "Model:    $(extract_const BeringModelV100Name)@$(extract_const BeringModelV100Version)"
echo "Snapshot: $(extract_const BeringSnapshotV100Name)@$(extract_const BeringSnapshotV100Version)"
echo "Model:    $(extract_const BeringModelV110Name)@$(extract_const BeringModelV110Version)"
echo "Snapshot: $(extract_const BeringSnapshotV110Name)@$(extract_const BeringSnapshotV110Version)"
