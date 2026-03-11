#!/usr/bin/env sh
set -eu

REPO_ROOT="${REPO_ROOT:-.}"
IMAGE_REPOSITORY="${IMAGE_REPOSITORY:?IMAGE_REPOSITORY is required}"
APP_VERSION="${APP_VERSION:?APP_VERSION is required}"
GIT_COMMIT="${GIT_COMMIT:-$(git -C "${REPO_ROOT}" rev-parse HEAD)}"
BUILD_DATE="${BUILD_DATE:-$(git -C "${REPO_ROOT}" log -1 --format=%cI)}"
IMAGE_METADATA_OUTPUT="${IMAGE_METADATA_OUTPUT:-dist/image-metadata.json}"
PLATFORMS="${PLATFORMS:-linux/amd64,linux/arm64}"
PUBLISH="${PUBLISH:-false}"

minor_tag="$(printf '%s' "${APP_VERSION}" | awk -F. '{print "v"$1"."$2}')"
full_tag="v${APP_VERSION}"
sha_tag="sha-${GIT_COMMIT}"

mkdir -p "$(dirname "${IMAGE_METADATA_OUTPUT}")"

if ! docker buildx version >/dev/null 2>&1; then
  echo "docker buildx is required" >&2
  exit 1
fi

build_platforms="${PLATFORMS}"
publish_flag=""
digest=""
if [ "${PUBLISH}" = "true" ]; then
  publish_flag="--push"
else
  build_platforms="$(printf '%s' "${PLATFORMS}" | cut -d, -f1)"
  publish_flag="--load"
fi

docker buildx build \
  --platform "${build_platforms}" \
  ${publish_flag} \
  --build-arg VERSION="${APP_VERSION}" \
  --build-arg COMMIT="${GIT_COMMIT}" \
  --build-arg BUILD_DATE="${BUILD_DATE}" \
  --label "org.opencontainers.image.title=Sheaft" \
  --label "org.opencontainers.image.version=${APP_VERSION}" \
  --label "org.opencontainers.image.revision=${GIT_COMMIT}" \
  --label "org.opencontainers.image.created=${BUILD_DATE}" \
  -t "${IMAGE_REPOSITORY}:${full_tag}" \
  -t "${IMAGE_REPOSITORY}:${minor_tag}" \
  -t "${IMAGE_REPOSITORY}:${sha_tag}" \
  -f "${REPO_ROOT}/build/Dockerfile" \
  "${REPO_ROOT}"

if [ "${PUBLISH}" = "true" ]; then
  digest="$(docker buildx imagetools inspect "${IMAGE_REPOSITORY}:${full_tag}" | awk 'tolower($1)=="digest:" { print $2; exit }')"
  if [ -z "${digest}" ]; then
    echo "failed to resolve pushed image digest for ${IMAGE_REPOSITORY}:${full_tag}" >&2
    exit 1
  fi
fi

set -- go run ./cmd/releasectl image-metadata --repository "${IMAGE_REPOSITORY}" --out "${IMAGE_METADATA_OUTPUT}"
if [ "${PUBLISH}" = "true" ]; then
  set -- "$@" --published --digest "${digest}"
fi
for tag in "${full_tag}" "${minor_tag}" "${sha_tag}"; do
  set -- "$@" --tag "${tag}"
done
for platform in $(printf '%s' "${build_platforms}" | tr ',' ' '); do
  set -- "$@" --platform "${platform}"
done
"$@"
