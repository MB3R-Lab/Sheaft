#!/usr/bin/env sh
set -eu

REPO_ROOT="${1:-.}"
CONTRACT_FILE="${REPO_ROOT}/internal/modelcontract/contract.go"
VENDORED_SCHEMA="${REPO_ROOT}/internal/modelcontract/schema/model.schema.json"
API_SCHEMA="${REPO_ROOT}/api/schema/model.schema.json"

extract_const() {
  name="$1"
  sed -n "s/^[[:space:]]*${name}[[:space:]]*=[[:space:]]*\"\\([^\"]*\\)\".*/\\1/p" "${CONTRACT_FILE}" | head -n 1
}

EXPECTED_URI="$(extract_const ExpectedSchemaURI)"
EXPECTED_DIGEST="$(extract_const ExpectedSchemaDigest)"
EXPECTED_VERSION="$(extract_const ExpectedSchemaVersion)"

if [ -z "${EXPECTED_URI}" ] || [ -z "${EXPECTED_DIGEST}" ] || [ -z "${EXPECTED_VERSION}" ]; then
  echo "Failed to read ExpectedSchema* constants from ${CONTRACT_FILE}" >&2
  exit 1
fi

if [ ! -f "${VENDORED_SCHEMA}" ] || [ ! -f "${API_SCHEMA}" ]; then
  echo "Schema files are missing" >&2
  exit 1
fi

TMP_REMOTE="$(mktemp)"
TMP_REMOTE_CANON="${TMP_REMOTE}.canon"
TMP_VENDORED_CANON="${TMP_REMOTE}.vendored.canon"
TMP_API_CANON="${TMP_REMOTE}.api.canon"
trap 'rm -f "${TMP_REMOTE}" "${TMP_REMOTE_CANON}" "${TMP_VENDORED_CANON}" "${TMP_API_CANON}"' EXIT

echo "Fetching remote schema: ${EXPECTED_URI}"
curl -fsSL "${EXPECTED_URI}" -o "${TMP_REMOTE}"

if command -v sha256sum >/dev/null 2>&1; then
  REMOTE_HASH="$(sha256sum "${TMP_REMOTE}" | awk '{print $1}')"
else
  REMOTE_HASH="$(shasum -a 256 "${TMP_REMOTE}" | awk '{print $1}')"
fi
REMOTE_DIGEST="sha256:${REMOTE_HASH}"

if [ "${REMOTE_DIGEST}" != "${EXPECTED_DIGEST}" ]; then
  echo "Digest mismatch: remote=${REMOTE_DIGEST} expected=${EXPECTED_DIGEST}" >&2
  exit 1
fi

if command -v jq >/dev/null 2>&1; then
  jq -cS . "${TMP_REMOTE}" > "${TMP_REMOTE_CANON}"
  jq -cS . "${VENDORED_SCHEMA}" > "${TMP_VENDORED_CANON}"
  jq -cS . "${API_SCHEMA}" > "${TMP_API_CANON}"
else
  cp "${TMP_REMOTE}" "${TMP_REMOTE_CANON}"
  cp "${VENDORED_SCHEMA}" "${TMP_VENDORED_CANON}"
  cp "${API_SCHEMA}" "${TMP_API_CANON}"
fi

if ! cmp -s "${TMP_REMOTE_CANON}" "${TMP_VENDORED_CANON}"; then
  echo "Vendored schema differs from remote schema" >&2
  exit 1
fi

if ! cmp -s "${TMP_REMOTE_CANON}" "${TMP_API_CANON}"; then
  echo "API schema differs from remote schema" >&2
  exit 1
fi

if command -v jq >/dev/null 2>&1; then
  REMOTE_ID="$(jq -r '."$id"' "${TMP_REMOTE}")"
  if [ "${REMOTE_ID}" != "${EXPECTED_URI}" ]; then
    echo "Remote schema \$id mismatch: remote=${REMOTE_ID} expected=${EXPECTED_URI}" >&2
    exit 1
  fi
fi

case "${EXPECTED_URI}" in
  */v"${EXPECTED_VERSION}"/*) ;;
  *)
    echo "ExpectedSchemaVersion (${EXPECTED_VERSION}) is not reflected in ExpectedSchemaURI (${EXPECTED_URI})" >&2
    exit 1
    ;;
esac

echo "Schema contract check passed"
echo "Version: ${EXPECTED_VERSION}"
echo "Digest:  ${EXPECTED_DIGEST}"
