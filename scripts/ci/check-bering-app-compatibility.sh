#!/usr/bin/env sh
set -eu

REPO_ROOT="${1:-.}"
RELEASE_MANIFEST="${2:-}"
CONTRACTS_ARCHIVE="${3:-}"
EXPECTED_BERING_APP_VERSION="${4:-${EXPECTED_BERING_APP_VERSION:-1.2.0}}"

if [ -z "${RELEASE_MANIFEST}" ] || [ -z "${CONTRACTS_ARCHIVE}" ]; then
  echo "usage: $0 REPO_ROOT BERING_RELEASE_MANIFEST BERING_CONTRACTS_ARCHIVE [EXPECTED_APP_VERSION]" >&2
  exit 2
fi
for path in "${RELEASE_MANIFEST}" "${CONTRACTS_ARCHIVE}" "${REPO_ROOT}/compatibility-manifest.json"; do
  if [ ! -f "${path}" ]; then
    echo "required compatibility input is missing: ${path}" >&2
    exit 1
  fi
done

find_python() {
  if command -v python3 >/dev/null 2>&1; then
    command -v python3
  elif command -v python >/dev/null 2>&1; then
    command -v python
  else
    echo "python3 or python is required" >&2
    exit 1
  fi
}

PYTHON_BIN="$(find_python)"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

"${PYTHON_BIN}" - \
  "${RELEASE_MANIFEST}" \
  "${CONTRACTS_ARCHIVE}" \
  "${REPO_ROOT}/compatibility-manifest.json" \
  "${EXPECTED_BERING_APP_VERSION}" <<'PY'
import hashlib
import json
import pathlib
import sys

release_path, archive_path, compatibility_path = map(pathlib.Path, sys.argv[1:4])
expected_version = sys.argv[4]

with release_path.open(encoding="utf-8") as stream:
    release = json.load(stream)
with compatibility_path.open(encoding="utf-8") as stream:
    compatibility = json.load(stream)

if release.get("product_name") != "bering":
    raise SystemExit("release manifest product_name is not bering")
if release.get("app_version") != expected_version:
    raise SystemExit(
        f"Bering app version mismatch: got={release.get('app_version')} want={expected_version}"
    )
if release.get("git_tag") != f"v{expected_version}":
    raise SystemExit(
        f"Bering git tag mismatch: got={release.get('git_tag')} want=v{expected_version}"
    )

expected_digest = release.get("contracts_pack", {}).get("digest", "")
actual_digest = "sha256:" + hashlib.sha256(archive_path.read_bytes()).hexdigest()
if actual_digest != expected_digest:
    raise SystemExit(
        f"contracts archive digest mismatch: got={actual_digest} want={expected_digest}"
    )

released = {
    (item.get("name"), item.get("version")): item
    for item in release.get("contracts", [])
}
for expected in compatibility.get("supported_contracts", []):
    key = (
        expected.get("schema_name"),
        expected.get("schema_version"),
    )
    actual = released.get(key)
    if actual is None:
        raise SystemExit(f"Bering release lacks supported contract {key}")
    for release_key, compatibility_key in (
        ("uri", "schema_uri"),
        ("digest", "schema_digest"),
    ):
        if actual.get(release_key) != expected.get(compatibility_key):
            raise SystemExit(
                f"Bering contract {key} {release_key} mismatch: "
                f"got={actual.get(release_key)} want={expected.get(compatibility_key)}"
            )
PY

tar -xzf "${CONTRACTS_ARCHIVE}" -C "${TMP_DIR}"
FIXTURE_DIR="${TMP_DIR}/fixtures/sheaft-v1"
FIXTURE_MANIFEST="${FIXTURE_DIR}/manifest.json"
if [ ! -f "${FIXTURE_MANIFEST}" ]; then
  echo "contracts archive lacks fixtures/sheaft-v1/manifest.json" >&2
  exit 1
fi

"${PYTHON_BIN}" - "${FIXTURE_MANIFEST}" "${REPO_ROOT}/compatibility-manifest.json" <<'PY'
import hashlib
import json
import pathlib
import sys

fixture_path = pathlib.Path(sys.argv[1])
compatibility_path = pathlib.Path(sys.argv[2])
fixture = json.loads(fixture_path.read_text(encoding="utf-8"))
compatibility = json.loads(compatibility_path.read_text(encoding="utf-8"))

expected_contracts = {
    (item["schema_name"], item["schema_version"]): (
        item["schema_uri"], item["schema_digest"]
    )
    for item in compatibility["supported_contracts"]
}
actual_contracts = {
    (item["name"], item["version"]): (item["uri"], item["digest"])
    for item in fixture.get("contracts", [])
}
if actual_contracts != expected_contracts:
    raise SystemExit("canonical fixture contracts do not match Sheaft strict pins")

for artifact in fixture.get("artifacts", []):
    path = fixture_path.parent / artifact["path"]
    if not path.is_file():
        raise SystemExit(f"canonical fixture is missing: {path}")
    digest = hashlib.sha256(path.read_bytes()).hexdigest()
    if digest != artifact["sha256"]:
        raise SystemExit(
            f"canonical fixture digest mismatch for {path.name}: "
            f"got={digest} want={artifact['sha256']}"
        )
PY

for artifact in bering-model.v1.sample.json bering-snapshot.v1.sample.json; do
  output_name="$(printf '%s' "${artifact}" | tr '.-' '__')"
  (
    cd "${REPO_ROOT}"
    go run ./cmd/sheaft run \
      --model "${FIXTURE_DIR}/${artifact}" \
      --policy configs/gate.policy.example.yaml \
      --out-dir "${TMP_DIR}/out-${output_name}" \
      --seed 42
  )
done

echo "Bering v${EXPECTED_BERING_APP_VERSION} app compatibility check passed"
