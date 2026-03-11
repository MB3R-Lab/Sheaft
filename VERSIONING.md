# Versioning

Sheaft has multiple version surfaces. They are related, but they are not interchangeable.

## 1. Sheaft App Version

- Format: `X.Y.Z`
- Source of truth for tagged releases: Git tag `vX.Y.Z`
- Stored in `release-manifest.json` as `app_version`
- Controls binary archive names, chart packaging version override, and image release tags

## 2. Upstream Contract Compatibility

- Source of truth: `internal/modelcontract/contract.go`
- Generated machine-readable view: `compatibility-manifest.json`
- Contents:
  - supported upstream artifact types
  - supported upstream schema names
  - supported upstream schema versions
  - required upstream schema digests
  - tested Bering app versions, if known

This is compatibility metadata, not ownership metadata.

- Bering owns schema names and schema versions.
- Sheaft only declares which Bering contracts it accepts.
- Changing Sheaft app version does not imply a schema version change.

## 3. Helm Chart Version

- Format: `X.Y.Z`
- Published as an OCI chart, not through a classic Helm repository
- Defaults to the app version for a tagged release
- Recorded in `release-manifest.json` under `chart.version`

## 4. OCI Image Tags

Each release image is published with:

- `vX.Y.Z`
- `vX.Y`
- `sha-<commit>`

`release-manifest.json` records the image references and, for published releases, the image digest.

## 5. Checksums and SBOMs

- Every archive has a SHA-256 checksum in the release checksums file.
- Every archive gets an SBOM artifact.
- `release-manifest.json` records the archive checksum and SBOM reference per platform.

## Practical Rule

Use the right metadata for the right decision:

- selecting a Sheaft binary/image/chart: use the Sheaft app version
- validating whether a Bering artifact is accepted: use `compatibility-manifest.json`
- scripting downloads and verification: use `release-manifest.json`
