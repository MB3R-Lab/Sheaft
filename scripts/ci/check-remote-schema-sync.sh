#!/usr/bin/env sh
set -eu

REPO_ROOT="${1:-.}"
CONTRACT_FILE="${REPO_ROOT}/internal/modelcontract/contract.go"
HELPER_FILE="${REPO_ROOT}/scripts/ci/contract-constants.sh"

if [ ! -f "${HELPER_FILE}" ]; then
  echo "Helper file is missing: ${HELPER_FILE}" >&2
  exit 1
fi

# shellcheck source=scripts/ci/contract-constants.sh
. "${HELPER_FILE}"

MODEL_URI="$(extract_const ExpectedSchemaURI)"
MODEL_DIGEST="$(extract_const ExpectedSchemaDigest)"
MODEL_VERSION="$(extract_const ExpectedSchemaVersion)"

SNAPSHOT_URI="$(extract_const BeringSnapshotV100URI)"
SNAPSHOT_DIGEST="$(extract_const BeringSnapshotV100Digest)"
SNAPSHOT_VERSION="$(extract_const BeringSnapshotV100Version)"

if [ -z "${MODEL_URI}" ] || [ -z "${MODEL_DIGEST}" ] || [ -z "${MODEL_VERSION}" ]; then
  echo "Failed to read model schema constants from ${CONTRACT_FILE}" >&2
  exit 1
fi

if [ -z "${SNAPSHOT_URI}" ] || [ -z "${SNAPSHOT_DIGEST}" ] || [ -z "${SNAPSHOT_VERSION}" ]; then
  echo "Failed to read snapshot schema constants from ${CONTRACT_FILE}" >&2
  exit 1
fi

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

canonicalize_json() {
  src="$1"
  dest="$2"

  if command -v jq >/dev/null 2>&1; then
    jq -cS . "${src}" > "${dest}"
  else
    cp "${src}" "${dest}"
  fi
}

check_schema_contract() {
  label="$1"
  expected_uri="$2"
  expected_digest="$3"
  expected_version="$4"
  vendored_schema="$5"
  api_schema="$6"

  if [ ! -f "${vendored_schema}" ] || [ ! -f "${api_schema}" ]; then
    echo "${label} schema files are missing" >&2
    exit 1
  fi

  remote_schema="${TMP_DIR}/${label}.remote.json"
  remote_canon="${TMP_DIR}/${label}.remote.canon"
  vendored_canon="${TMP_DIR}/${label}.vendored.canon"
  api_canon="${TMP_DIR}/${label}.api.canon"

  echo "Fetching remote ${label} schema: ${expected_uri}"
  curl -fsSL "${expected_uri}" -o "${remote_schema}"

  if command -v sha256sum >/dev/null 2>&1; then
    remote_hash="$(sha256sum "${remote_schema}" | awk '{print $1}')"
  else
    remote_hash="$(shasum -a 256 "${remote_schema}" | awk '{print $1}')"
  fi
  remote_digest="sha256:${remote_hash}"

  if [ "${remote_digest}" != "${expected_digest}" ]; then
    echo "${label} schema digest mismatch: remote=${remote_digest} expected=${expected_digest}" >&2
    exit 1
  fi

  canonicalize_json "${remote_schema}" "${remote_canon}"
  canonicalize_json "${vendored_schema}" "${vendored_canon}"
  canonicalize_json "${api_schema}" "${api_canon}"

  if ! cmp -s "${remote_canon}" "${vendored_canon}"; then
    echo "Vendored ${label} schema differs from remote schema" >&2
    exit 1
  fi

  if ! cmp -s "${remote_canon}" "${api_canon}"; then
    echo "API ${label} schema differs from remote schema" >&2
    exit 1
  fi

  if command -v jq >/dev/null 2>&1; then
    remote_id="$(jq -r '."$id"' "${remote_schema}")"
    if [ "${remote_id}" != "${expected_uri}" ]; then
      echo "Remote ${label} schema \$id mismatch: remote=${remote_id} expected=${expected_uri}" >&2
      exit 1
    fi
  fi

  case "${expected_uri}" in
    */v"${expected_version}"/*) ;;
    *)
      echo "${label} schema version (${expected_version}) is not reflected in schema URI (${expected_uri})" >&2
      exit 1
      ;;
  esac

  echo "${label} schema contract check passed"
  echo "Version: ${expected_version}"
  echo "Digest:  ${expected_digest}"
}

check_schema_contract \
  "model" \
  "${MODEL_URI}" \
  "${MODEL_DIGEST}" \
  "${MODEL_VERSION}" \
  "${REPO_ROOT}/internal/modelcontract/schema/model.schema.json" \
  "${REPO_ROOT}/api/schema/model.schema.json"

check_schema_contract \
  "snapshot" \
  "${SNAPSHOT_URI}" \
  "${SNAPSHOT_DIGEST}" \
  "${SNAPSHOT_VERSION}" \
  "${REPO_ROOT}/internal/modelcontract/schema/snapshot.schema.json" \
  "${REPO_ROOT}/api/schema/snapshot.schema.json"
