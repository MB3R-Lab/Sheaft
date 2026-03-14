# Changelog

## v0.1.1 - 2026-03-14

Patch technical-preview release focused on restoring the advertised snapshot compatibility surface with current Bering `io.mb3r.bering.snapshot@1.0.0`.

Included in this release:

- updated strict snapshot contract pin to the current published Bering `1.0.0` digest
- synced vendored and mirrored snapshot schemas with the published Bering snapshot schema
- updated snapshot loader to accept the current Bering snapshot envelope
- refreshed checked-in snapshot sample and generated example outputs to match current upstream structure
- added CI sanity checks that the published Bering model and snapshot schema URLs still match Sheaft pins and local schema copies

Stable within the `v0.1.1` preview:

- strict acceptance of current `io.mb3r.bering.model@1.0.0`
- strict acceptance of current `io.mb3r.bering.snapshot@1.0.0`
- checked-in model and snapshot smoke paths
- deterministic batch execution for a fixed seed and config

Still experimental in `v0.1.1`:

- long-running `serve` / `watch` service mode
- richer analysis config and operator-facing conventions
- local `discover` helper

Known limitations:

- only the Bering `1.0.0` model and snapshot contracts are accepted
- snapshot envelopes still rely on external predicate overlays or fallback journey resolution when richer predicate definitions are not embedded upstream
- no new discovery pipeline is introduced in this release

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
