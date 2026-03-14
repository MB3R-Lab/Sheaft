# Releasing Sheaft

Sheaft release automation is tag-driven and platform-neutral. The `v0.1.x` line is intentionally published as a technical preview, so release notes and README language should stay explicit about what is stable versus experimental.

The canonical release contract is the generated payload:

- `dist/`
- `compatibility-manifest.json`
- `release-manifest.json`

GitHub Releases publish that payload, but GitHub metadata is not the source of truth.

## Ownership Boundary

- Bering owns upstream artifact schemas and release metadata.
- Sheaft stays downstream and only declares compatibility with Bering-produced artifacts.
- Sheaft must not redefine or silently mutate Bering schema versions.
- Strict contract validation remains in `internal/modelcontract/contract.go` and is mirrored into `compatibility-manifest.json`.

## Version Surfaces

- Sheaft app version: semantic version without the leading `v` inside generated manifests.
- Git tag: `vX.Y.Z`
- Helm chart version: `X.Y.Z`
- Image tags:
  - `vX.Y.Z`
  - `vX.Y`
  - `sha-<commit>`
- Supported upstream contracts: declared in `compatibility-manifest.json`

See [VERSIONING.md](VERSIONING.md) for the detailed contract.

## Release Assets

A successful tagged release publishes:

- platform archives for `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, `windows/amd64`
- `checksums.txt`
- per-archive SBOM files
- source archive
- multi-arch OCI image
- OCI Helm chart
- `compatibility-manifest.json`
- `release-manifest.json`
- default config pack archive
- versioned GitHub release notes from `release/<tag>.md`, when present

## Local Validation

Dry-run validation:

```bash
make release-dry-run APP_VERSION=0.0.0-dev
```

`release-dry-run` includes the checked-in example smoke path. You can still run it separately when you only want to validate the first-run surface:

```bash
make smoke-examples
```

Local end-to-end publish to a local OCI registry:

```bash
make release-local APP_VERSION=0.0.0-dev LOCAL_REGISTRY=localhost:5000
```

`release-local` expects:

- Docker with `buildx`
- Helm 3 with OCI support
- Go 1.26+ on the release runner, because the release-tool bootstrap path installs `syft`
- network access for Go tool bootstrap

## GitHub Publisher Flow

`.github/workflows/release.yml` is the GitHub publisher only.

On tag push it:

1. runs tests;
2. runs smoke checks against checked-in examples;
3. builds release archives with GoReleaser;
4. publishes the OCI image;
5. publishes the OCI Helm chart;
6. generates `release-manifest.json`;
7. creates or updates the GitHub Release using `release/<tag>.md` when available;
8. uploads the canonical payload to the GitHub Release.

## Generic CI Reuse

GitLab and Jenkins should call the same repo entrypoints instead of reproducing logic:

- `make test`
- `make release-dry-run`
- `make release-local`
- `make chart-package`
- `make compatibility-manifest`
- `make release-manifest`

That keeps the release contract identical across GitHub, GitLab, Jenkins, and local runs.
