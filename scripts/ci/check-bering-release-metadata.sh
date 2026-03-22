#!/usr/bin/env sh
set -eu

REPO_ROOT="${1:-.}"
METADATA_URL="${BERING_RELEASE_METADATA_URL:-https://mb3r-lab.github.io/Bering/schema/index.json}"
CONTRACT_FILE="${REPO_ROOT}/internal/modelcontract/contract.go"
HELPER_FILE="${REPO_ROOT}/scripts/ci/contract-constants.sh"

if [ ! -f "${CONTRACT_FILE}" ]; then
  echo "Contract file is missing: ${CONTRACT_FILE}" >&2
  exit 1
fi

if [ ! -f "${HELPER_FILE}" ]; then
  echo "Helper file is missing: ${HELPER_FILE}" >&2
  exit 1
fi

# shellcheck source=scripts/ci/contract-constants.sh
. "${HELPER_FILE}"

EXPECTED_MODEL_NAME="$(extract_const BeringModelV110Name)"
EXPECTED_MODEL_VERSION="$(extract_const BeringModelV110Version)"
EXPECTED_MODEL_URI="$(extract_const BeringModelV110URI)"
EXPECTED_MODEL_DIGEST="$(extract_const BeringModelV110Digest)"
EXPECTED_SNAPSHOT_NAME="$(extract_const BeringSnapshotV110Name)"
EXPECTED_SNAPSHOT_VERSION="$(extract_const BeringSnapshotV110Version)"
EXPECTED_SNAPSHOT_URI="$(extract_const BeringSnapshotV110URI)"
EXPECTED_SNAPSHOT_DIGEST="$(extract_const BeringSnapshotV110Digest)"

TMP_METADATA="$(mktemp)"
trap 'rm -f "${TMP_METADATA}"' EXIT

echo "Fetching Bering release metadata: ${METADATA_URL}"
curl -fsSL "${METADATA_URL}" -o "${TMP_METADATA}"

python_cmd=""
if command -v python3 >/dev/null 2>&1; then
  python_cmd="python3"
elif command -v python >/dev/null 2>&1; then
  python_cmd="python"
else
  echo "python3 or python is required to parse release metadata json" >&2
  exit 1
fi

METADATA_OUTPUT="$("${python_cmd}" - "${TMP_METADATA}" <<'PY'
import json
import sys

with open(sys.argv[1], "r", encoding="utf-8") as fh:
    data = json.load(fh)

def emit(prefix, obj):
    for key in ("name", "version", "uri", "digest"):
        value = obj.get(key, "")
        print(f"{prefix}_{key}={value}")

emit("top_level", data)
emit("model", data.get("model", {}))
emit("snapshot", data.get("snapshot", {}))
contracts = {}
for entry in data.get("contracts", []):
    name = entry.get("name", "")
    version = entry.get("version", "")
    if name and version:
        contracts[(name, version)] = entry
for prefix, key in (
    ("contract_model", ("io.mb3r.bering.model", "1.1.0")),
    ("contract_snapshot", ("io.mb3r.bering.snapshot", "1.1.0")),
):
    emit(prefix, contracts.get(key, {}))
print(f"updated_at={data.get('updated_at', '')}")
PY
)"

METADATA_TOP_LEVEL_NAME="$(printf '%s\n' "${METADATA_OUTPUT}" | sed -n 's/^top_level_name=//p')"
METADATA_TOP_LEVEL_VERSION="$(printf '%s\n' "${METADATA_OUTPUT}" | sed -n 's/^top_level_version=//p')"
METADATA_TOP_LEVEL_URI="$(printf '%s\n' "${METADATA_OUTPUT}" | sed -n 's/^top_level_uri=//p')"
METADATA_TOP_LEVEL_DIGEST="$(printf '%s\n' "${METADATA_OUTPUT}" | sed -n 's/^top_level_digest=//p')"
METADATA_MODEL_NAME="$(printf '%s\n' "${METADATA_OUTPUT}" | sed -n 's/^model_name=//p')"
METADATA_MODEL_VERSION="$(printf '%s\n' "${METADATA_OUTPUT}" | sed -n 's/^model_version=//p')"
METADATA_MODEL_URI="$(printf '%s\n' "${METADATA_OUTPUT}" | sed -n 's/^model_uri=//p')"
METADATA_MODEL_DIGEST="$(printf '%s\n' "${METADATA_OUTPUT}" | sed -n 's/^model_digest=//p')"
METADATA_SNAPSHOT_NAME="$(printf '%s\n' "${METADATA_OUTPUT}" | sed -n 's/^snapshot_name=//p')"
METADATA_SNAPSHOT_VERSION="$(printf '%s\n' "${METADATA_OUTPUT}" | sed -n 's/^snapshot_version=//p')"
METADATA_SNAPSHOT_URI="$(printf '%s\n' "${METADATA_OUTPUT}" | sed -n 's/^snapshot_uri=//p')"
METADATA_SNAPSHOT_DIGEST="$(printf '%s\n' "${METADATA_OUTPUT}" | sed -n 's/^snapshot_digest=//p')"
METADATA_CONTRACT_MODEL_NAME="$(printf '%s\n' "${METADATA_OUTPUT}" | sed -n 's/^contract_model_name=//p')"
METADATA_CONTRACT_MODEL_VERSION="$(printf '%s\n' "${METADATA_OUTPUT}" | sed -n 's/^contract_model_version=//p')"
METADATA_CONTRACT_MODEL_URI="$(printf '%s\n' "${METADATA_OUTPUT}" | sed -n 's/^contract_model_uri=//p')"
METADATA_CONTRACT_MODEL_DIGEST="$(printf '%s\n' "${METADATA_OUTPUT}" | sed -n 's/^contract_model_digest=//p')"
METADATA_CONTRACT_SNAPSHOT_NAME="$(printf '%s\n' "${METADATA_OUTPUT}" | sed -n 's/^contract_snapshot_name=//p')"
METADATA_CONTRACT_SNAPSHOT_VERSION="$(printf '%s\n' "${METADATA_OUTPUT}" | sed -n 's/^contract_snapshot_version=//p')"
METADATA_CONTRACT_SNAPSHOT_URI="$(printf '%s\n' "${METADATA_OUTPUT}" | sed -n 's/^contract_snapshot_uri=//p')"
METADATA_CONTRACT_SNAPSHOT_DIGEST="$(printf '%s\n' "${METADATA_OUTPUT}" | sed -n 's/^contract_snapshot_digest=//p')"
METADATA_UPDATED_AT="$(printf '%s\n' "${METADATA_OUTPUT}" | sed -n 's/^updated_at=//p')"

if [ -z "${METADATA_UPDATED_AT}" ]; then
  echo "Bering release metadata is missing updated_at" >&2
  exit 1
fi

assert_equal() {
  label="$1"
  got="$2"
  want="$3"

  if [ "${got}" != "${want}" ]; then
    echo "Bering release metadata mismatch for ${label}: got=${got} want=${want}" >&2
    exit 1
  fi
}

assert_equal "top-level name" "${METADATA_TOP_LEVEL_NAME}" "${EXPECTED_MODEL_NAME}"
assert_equal "top-level version" "${METADATA_TOP_LEVEL_VERSION}" "${EXPECTED_MODEL_VERSION}"
assert_equal "top-level uri" "${METADATA_TOP_LEVEL_URI}" "${EXPECTED_MODEL_URI}"
assert_equal "top-level digest" "${METADATA_TOP_LEVEL_DIGEST}" "${EXPECTED_MODEL_DIGEST}"
assert_equal "model name" "${METADATA_MODEL_NAME}" "${EXPECTED_MODEL_NAME}"
assert_equal "model version" "${METADATA_MODEL_VERSION}" "${EXPECTED_MODEL_VERSION}"
assert_equal "model uri" "${METADATA_MODEL_URI}" "${EXPECTED_MODEL_URI}"
assert_equal "model digest" "${METADATA_MODEL_DIGEST}" "${EXPECTED_MODEL_DIGEST}"
assert_equal "snapshot name" "${METADATA_SNAPSHOT_NAME}" "${EXPECTED_SNAPSHOT_NAME}"
assert_equal "snapshot version" "${METADATA_SNAPSHOT_VERSION}" "${EXPECTED_SNAPSHOT_VERSION}"
assert_equal "snapshot uri" "${METADATA_SNAPSHOT_URI}" "${EXPECTED_SNAPSHOT_URI}"
assert_equal "snapshot digest" "${METADATA_SNAPSHOT_DIGEST}" "${EXPECTED_SNAPSHOT_DIGEST}"
assert_equal "contracts model name" "${METADATA_CONTRACT_MODEL_NAME}" "${EXPECTED_MODEL_NAME}"
assert_equal "contracts model version" "${METADATA_CONTRACT_MODEL_VERSION}" "${EXPECTED_MODEL_VERSION}"
assert_equal "contracts model uri" "${METADATA_CONTRACT_MODEL_URI}" "${EXPECTED_MODEL_URI}"
assert_equal "contracts model digest" "${METADATA_CONTRACT_MODEL_DIGEST}" "${EXPECTED_MODEL_DIGEST}"
assert_equal "contracts snapshot name" "${METADATA_CONTRACT_SNAPSHOT_NAME}" "${EXPECTED_SNAPSHOT_NAME}"
assert_equal "contracts snapshot version" "${METADATA_CONTRACT_SNAPSHOT_VERSION}" "${EXPECTED_SNAPSHOT_VERSION}"
assert_equal "contracts snapshot uri" "${METADATA_CONTRACT_SNAPSHOT_URI}" "${EXPECTED_SNAPSHOT_URI}"
assert_equal "contracts snapshot digest" "${METADATA_CONTRACT_SNAPSHOT_DIGEST}" "${EXPECTED_SNAPSHOT_DIGEST}"

echo "Bering release metadata check passed"
echo "Model:      ${METADATA_MODEL_VERSION}"
echo "Snapshot:   ${METADATA_SNAPSHOT_VERSION}"
echo "Updated at: ${METADATA_UPDATED_AT}"
