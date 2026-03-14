# Changelog

## v0.1.0 - 2026-03-14

First public Sheaft release, published as an experimental technical preview rather than a stable GA release.

Included in this release:

- downstream CLI surface for `simulate`, `gate`, `run`, and experimental `serve` / `watch`
- strict compatibility with `io.mb3r.bering.model@1.0.0` and `io.mb3r.bering.snapshot@1.0.0`
- checked-in sample artifacts and configs for first-run smoke paths
- release packaging for Linux and macOS on `amd64` and `arm64`, plus optional Windows archives through GoReleaser
- release metadata via `compatibility-manifest.json`, `release-manifest.json`, checksums, SBOMs, OCI image packaging, OCI Helm chart packaging, and the default config pack

Stable within the `v0.1.0` preview:

- accepted upstream Bering contracts
- batch mode CLI command names and report generation flow
- reproducible archive naming through GoReleaser

Still experimental in `v0.1.0`:

- long-running `serve` / `watch` service mode
- richer analysis config and operator-facing conventions
- local `discover` helper

Known limitations:

- only the Bering `1.0.0` model and snapshot contracts are accepted
- no new discovery pipeline is introduced in this release
- service mode is intended for evaluation and technical-preview feedback, not a stable operations contract
