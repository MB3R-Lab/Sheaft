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

EXPECTED_NAME="$(extract_const ExpectedSchemaName)"
EXPECTED_VERSION="$(extract_const ExpectedSchemaVersion)"
EXPECTED_URI="$(extract_const ExpectedSchemaURI)"
EXPECTED_DIGEST="$(extract_const ExpectedSchemaDigest)"

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

for key in ("name", "version", "uri", "digest", "updated_at"):
    value = data.get(key, "")
    print(f"{key}={value}")
PY
)"

METADATA_NAME="$(printf '%s\n' "${METADATA_OUTPUT}" | sed -n 's/^name=//p')"
METADATA_VERSION="$(printf '%s\n' "${METADATA_OUTPUT}" | sed -n 's/^version=//p')"
METADATA_URI="$(printf '%s\n' "${METADATA_OUTPUT}" | sed -n 's/^uri=//p')"
METADATA_DIGEST="$(printf '%s\n' "${METADATA_OUTPUT}" | sed -n 's/^digest=//p')"
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

assert_equal "name" "${METADATA_NAME}" "${EXPECTED_NAME}"
assert_equal "version" "${METADATA_VERSION}" "${EXPECTED_VERSION}"
assert_equal "uri" "${METADATA_URI}" "${EXPECTED_URI}"
assert_equal "digest" "${METADATA_DIGEST}" "${EXPECTED_DIGEST}"

echo "Bering release metadata check passed"
echo "Version:    ${METADATA_VERSION}"
echo "Updated at: ${METADATA_UPDATED_AT}"
