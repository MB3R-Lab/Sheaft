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

check_contract_version() {
  label="$1"
  uri_const="$2"
  digest_const="$3"
  version_const="$4"
  vendored_schema="$5"
  api_schema="$6"

  uri="$(extract_const "${uri_const}")"
  digest="$(extract_const "${digest_const}")"
  version="$(extract_const "${version_const}")"

  if [ -z "${uri}" ] || [ -z "${digest}" ] || [ -z "${version}" ]; then
    echo "Failed to read ${label} schema constants from ${CONTRACT_FILE}" >&2
    exit 1
  fi

  check_schema_contract \
    "${label}" \
    "${uri}" \
    "${digest}" \
    "${version}" \
    "${vendored_schema}" \
    "${api_schema}"
}

check_contract_version \
  "model-v1.0.0" \
  "BeringModelV100URI" \
  "BeringModelV100Digest" \
  "BeringModelV100Version" \
  "${REPO_ROOT}/internal/modelcontract/schema/model.v1.0.0.schema.json" \
  "${REPO_ROOT}/api/schema/model.v1.0.0.schema.json"

check_contract_version \
  "snapshot-v1.0.0" \
  "BeringSnapshotV100URI" \
  "BeringSnapshotV100Digest" \
  "BeringSnapshotV100Version" \
  "${REPO_ROOT}/internal/modelcontract/schema/snapshot.v1.0.0.schema.json" \
  "${REPO_ROOT}/api/schema/snapshot.v1.0.0.schema.json"

check_contract_version \
  "model-v1.1.0" \
  "BeringModelV110URI" \
  "BeringModelV110Digest" \
  "BeringModelV110Version" \
  "${REPO_ROOT}/internal/modelcontract/schema/model.v1.1.0.schema.json" \
  "${REPO_ROOT}/api/schema/model.v1.1.0.schema.json"

check_contract_version \
  "snapshot-v1.1.0" \
  "BeringSnapshotV110URI" \
  "BeringSnapshotV110Digest" \
  "BeringSnapshotV110Version" \
  "${REPO_ROOT}/internal/modelcontract/schema/snapshot.v1.1.0.schema.json" \
  "${REPO_ROOT}/api/schema/snapshot.v1.1.0.schema.json"
