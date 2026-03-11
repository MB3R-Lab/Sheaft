#!/usr/bin/env sh
set -eu

REPO_ROOT="${REPO_ROOT:-.}"
HELM_BIN="${HELM_BIN:-helm}"
CHART_DIR="${CHART_DIR:-charts/sheaft}"
CHART_OUTPUT_DIR="${CHART_OUTPUT_DIR:-dist/charts}"
CHART_METADATA_OUTPUT="${CHART_METADATA_OUTPUT:-dist/chart-metadata.json}"
CHART_NAME="${CHART_NAME:-sheaft}"
APP_VERSION="${APP_VERSION:?APP_VERSION is required}"
CHART_VERSION="${CHART_VERSION:-${APP_VERSION}}"
CHART_OCI_REPOSITORY="${CHART_OCI_REPOSITORY:-}"
PUBLISH="${PUBLISH:-false}"

mkdir -p "${CHART_OUTPUT_DIR}" "$(dirname "${CHART_METADATA_OUTPUT}")"

"${HELM_BIN}" lint "${REPO_ROOT}/${CHART_DIR}" >/dev/null

package_output="$("${HELM_BIN}" package "${REPO_ROOT}/${CHART_DIR}" \
  --destination "${CHART_OUTPUT_DIR}" \
  --version "${CHART_VERSION}" \
  --app-version "${APP_VERSION}")"
archive_path="$(printf '%s\n' "${package_output}" | sed -n 's/^Successfully packaged chart and saved it to: //p')"

if [ -z "${archive_path}" ]; then
  echo "failed to detect packaged chart path" >&2
  exit 1
fi

published_flag=""
oci_reference=""
digest=""
if [ "${PUBLISH}" = "true" ]; then
  if [ -z "${CHART_OCI_REPOSITORY}" ]; then
    echo "CHART_OCI_REPOSITORY is required when PUBLISH=true" >&2
    exit 1
  fi
  push_output="$("${HELM_BIN}" push "${archive_path}" "${CHART_OCI_REPOSITORY}" 2>&1)"
  digest="$(printf '%s\n' "${push_output}" | awk 'tolower($1)=="digest:" { print $2; exit }')"
  if [ -z "${digest}" ]; then
    echo "failed to resolve chart digest from helm push output" >&2
    printf '%s\n' "${push_output}" >&2
    exit 1
  fi
  oci_reference="${CHART_OCI_REPOSITORY}/${CHART_NAME}:${CHART_VERSION}"
  published_flag="true"
fi

set -- go run ./cmd/releasectl chart-metadata \
  --name "${CHART_NAME}" \
  --version "${CHART_VERSION}" \
  --archive "${archive_path}" \
  --out "${CHART_METADATA_OUTPUT}"
if [ "${published_flag}" = "true" ]; then
  set -- "$@" --published --oci-reference "${oci_reference}" --digest "${digest}"
fi
"$@"
