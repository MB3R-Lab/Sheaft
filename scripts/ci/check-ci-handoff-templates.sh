#!/usr/bin/env sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
REPO_ROOT="$(CDPATH= cd -- "${SCRIPT_DIR}/../.." && pwd)"

require_in_file() {
  needle="$1"
  file="$2"
  label="$3"

  if ! grep -Fq "${needle}" "${file}"; then
    echo "Missing ${label} in ${file}: ${needle}" >&2
    exit 1
  fi
}

require_common_handoff() {
  file="$1"
  label="$2"

  require_in_file "BERING_ARTIFACT_SOURCE" "${file}" "${label} artifact source variable"
  require_in_file "artifacts/input.json" "${file}" "${label} handoff path"
  require_in_file "configs/analysis.v1.1.example.yaml" "${file}" "${label} analysis config"
  require_in_file "out" "${file}" "${label} output directory"
}

DOC_FILE="${REPO_ROOT}/docs/ci-gate.md"
GITHUB_FILE="${REPO_ROOT}/examples/ci/github-actions.sheaft.yml"
GITLAB_FILE="${REPO_ROOT}/examples/ci/gitlab-ci.sheaft.yml"
JENKINS_FILE="${REPO_ROOT}/examples/ci/Jenkinsfile"
WORKFLOW_FILE="${REPO_ROOT}/.github/workflows/ci-template-smoke.yml"
SMOKE_SCRIPT="${REPO_ROOT}/scripts/ci/smoke-ci-handoff.sh"

for file in \
  "${DOC_FILE}" \
  "${GITHUB_FILE}" \
  "${GITLAB_FILE}" \
  "${JENKINS_FILE}" \
  "${WORKFLOW_FILE}" \
  "${SMOKE_SCRIPT}"
do
  if [ ! -f "${file}" ]; then
    echo "Required CI validation file is missing: ${file}" >&2
    exit 1
  fi
done

require_common_handoff "${DOC_FILE}" "docs"
require_common_handoff "${GITHUB_FILE}" "GitHub Actions template"
require_common_handoff "${GITLAB_FILE}" "GitLab template"
require_common_handoff "${JENKINS_FILE}" "Jenkins template"

require_in_file "retention-days: 7" "${GITHUB_FILE}" "GitHub Actions upstream retention"
require_in_file "retention-days: 14" "${GITHUB_FILE}" "GitHub Actions output retention"
require_in_file "actions/download-artifact@v4" "${GITHUB_FILE}" "GitHub Actions handoff download"
require_in_file "actions/upload-artifact@v4" "${GITHUB_FILE}" "GitHub Actions artifact upload"
require_in_file "docker build -f build/Dockerfile -t sheaft:ci ." "${GITHUB_FILE}" "GitHub Actions Docker build"

require_in_file "expire_in: 7 days" "${GITLAB_FILE}" "GitLab upstream retention"
require_in_file "expire_in: 14 days" "${GITLAB_FILE}" "GitLab output retention"
require_in_file "docker:27-dind" "${GITLAB_FILE}" "GitLab Docker service"
require_in_file 'docker build -f build/Dockerfile -t "$SHEAFT_IMAGE" .' "${GITLAB_FILE}" "GitLab Docker build"

require_in_file "stash name: 'bering-artifact'" "${JENKINS_FILE}" "Jenkins artifact stash"
require_in_file "unstash 'bering-artifact'" "${JENKINS_FILE}" "Jenkins artifact handoff"
require_in_file "archiveArtifacts artifacts: 'artifacts/input.json,out/**', fingerprint: true" "${JENKINS_FILE}" "Jenkins archive step"
require_in_file "docker build -f build/Dockerfile -t sheaft:ci ." "${JENKINS_FILE}" "Jenkins Docker build"

require_in_file "sh scripts/ci/check-ci-handoff-templates.sh" "${WORKFLOW_FILE}" "workflow template check"
require_in_file "sh scripts/ci/smoke-ci-handoff.sh native" "${WORKFLOW_FILE}" "workflow native smoke"
require_in_file "sh scripts/ci/smoke-ci-handoff.sh docker" "${WORKFLOW_FILE}" "workflow docker smoke"
require_in_file "actions/setup-go@v5" "${WORKFLOW_FILE}" "workflow Go setup"

require_in_file "scripts/ci/check-ci-handoff-templates.sh" "${DOC_FILE}" "docs template check command"
require_in_file "scripts/ci/smoke-ci-handoff.sh native" "${DOC_FILE}" "docs native smoke command"
require_in_file "scripts/ci/smoke-ci-handoff.sh docker" "${DOC_FILE}" "docs docker smoke command"
require_in_file ".github/workflows/ci-template-smoke.yml" "${DOC_FILE}" "docs workflow reference"

echo "CI handoff template check passed"
