# Changelog

## Unreleased

## v1.2.0 - 2026-07-26

- added analysis schema `1.2` failure-tolerance sweeps for independent replica failure probability and exact failed replica-slot counts
- added endpoint SLO crossing points, Wilson confidence intervals, conservative certified tolerance, and deterministic sweep fingerprints to `report.json` and `summary.md`
- added fail-closed boundary gates for minimum certified tolerance, maximum baseline regression, and indeterminate evidence
- added paired raw-artifact sweep evaluation and compatible prior-report boundary diffs
- kept sweep points outside existing profile aggregation and retained backward compatibility for analysis schemas `1.0` and `1.1`
- documented results as fail-stop model boundaries, not capacity or overload-cascade predictions

## v1.1.0 - 2026-07-02

- added `fixed_k_replica_slots` sampling for fixed-fraction experiment reproduction; when `fixed_k_failures` is omitted, Sheaft derives `k = ceil(failure_probability * total_replica_slots)`
- added schema, methodology, configuration example, and simulation coverage for fixed replica-slot sampling
- kept Bering contract compatibility unchanged on the strict `io.mb3r.bering.model@1.3.0` and `io.mb3r.bering.snapshot@1.3.0` line

## v1.0.0 - 2026-07-02

- added v1 major semantics documentation for the stochastic connectivity baseline, migration notes, and explicit out-of-scope release boundaries
- added release dry-run validation for v1 docs/schema/example consistency
- recorded app-level sync with Bering `v1.0.0` in `compatibility-manifest.json`
- breaking: retired pre-v1 Bering preview contract lines `1.0.0`, `1.1.0`, and `1.2.0`; Sheaft v1 accepts only `io.mb3r.bering.model@1.3.0` and `io.mb3r.bering.snapshot@1.3.0`

## v0.2.4 - 2026-05-31

Technical-preview feature release focused on trustable gate evidence, benchmark packaging, and explainable gate decisions for the `v0.2.x` line.

Included in this release:

- added a fixed Sheaft-on-Bering benchmark slice with `make benchmark-slice` and a versioned manifest under `benchmarks/fixed-slice`
- added benchmark quality reporting with repeatability, confidence, advanced-metric availability, baseline-diff, contract, and cross-profile checks
- added machine-readable gate decision reasons in `report.json` and a Why section in `summary.md`
- added `sheaft gate --why` and `sheaft run --why` for human-readable gate explanations
- added the benchmark manifest to the default config pack and the benchmark slice to release dry-run validation
- refreshed roadmap, release tracking, and assumptions documentation after closing R4.1, R4.2, R4.3, and R9.1

Stable within the `v0.2.4` preview:

- the same strict `1.0.0` and `1.1.0` Bering contract acceptance introduced in `v0.2.0`
- deterministic batch analysis, baseline comparison, and CI gate behavior from the `v0.2.0` line
- release validation now includes the fixed benchmark slice and quality-report generation
- gate decisions now carry threshold, aggregate, and assertion causes for release review

Still experimental in `v0.2.4`:

- long-running `serve` / `watch` service mode remains technical-preview surface
- local `discover` helper
- broader operator-facing packaging and operational conventions around image/chart deployment
- benchmark scale and external quality datasets remain outside this repository

## v0.2.3 - 2026-05-01

Technical-preview patch release focused on Kubernetes handoff reliability for the `v0.2.x` line.

Included in this release:

- fixed `serve` / `watch` startup when the configured artifact file appears after the service starts
- kept polling active when filesystem watcher setup cannot attach to a not-yet-created artifact directory and `watch_polling` is enabled
- added regression coverage for late Bering artifact handoff in service mode

Stable within the `v0.2.3` preview:

- the same strict `1.0.0` and `1.1.0` Bering contract acceptance introduced in `v0.2.0`
- deterministic batch analysis, baseline comparison, and CI gate behavior from the `v0.2.0` line
- service mode can recover when a downstream chart starts Sheaft before the first Bering artifact exists

Still experimental in `v0.2.3`:

- long-running `serve` / `watch` service mode remains technical-preview surface
- local `discover` helper
- broader operator-facing packaging and operational conventions around image/chart deployment

## v0.2.2 - 2026-05-01

Technical-preview patch release focused on post-audit hardening for the `v0.2.x` line.

Included in this release:

- clarified roadmap and assumptions docs so product priorities match the tracker-backed backlog
- hardened `serve` with explicit HTTP read-header, read, write, and idle timeouts
- fixed watch-loop error handling so filesystem watcher setup/add failures are reported and closed watcher channels do not spin
- moved release workflow, Docker builder, and module toolchain pins to Go `1.26.2`
- fixed Go `1.26` vet compatibility in contract policy errors
- hardened the default Helm chart security context to run non-root, drop capabilities, and use a read-only root filesystem

Stable within the `v0.2.2` preview:

- the same strict `1.0.0` and `1.1.0` Bering contract acceptance introduced in `v0.2.0`
- the same deterministic baseline comparison and advanced analysis behavior shipped in `v0.2.0`
- release validation is clean under `go test`, `go vet`, chart validation, and `govulncheck`

Still experimental in `v0.2.2`:

- long-running `serve` / `watch` service mode remains technical-preview surface despite the hardening fixes
- local `discover` helper
- broader operator-facing packaging and operational conventions around image/chart deployment

## v0.2.1 - 2026-03-22

Technical-preview patch release focused on post-release hardening for the `v0.2.0` line.

Included in this release:

- clarified README wording so dual-line support reads as accepted upstream contract lines rather than simultaneous artifact dependencies
- added and repaired `release-dry-run` status coverage for `main`
- fixed CI template convention checks to expect the shipped `configs/analysis.v1.1.example.yaml` handoff path
- fixed default config pack validation for absolute temporary output paths used by CI

Stable within the `v0.2.1` preview:

- the same strict `1.0.0` and `1.1.0` Bering contract acceptance introduced in `v0.2.0`
- the same deterministic baseline comparison and advanced analysis behavior shipped in `v0.2.0`
- release validation now includes a working `release-dry-run` signal on `main`

Still experimental in `v0.2.1`:

- long-running `serve` / `watch` service mode
- local `discover` helper
- broader operator-facing packaging and operational conventions around image/chart deployment

## v0.2.0 - 2026-03-22

Technical-preview feature release that turns Sheaft into a real dual-line downstream consumer for Bering `1.0.0` and `1.1.0`, while keeping `1.0.0` as the baseline semantics line.

Included in this release:

- strict dual support for Bering `io.mb3r.bering.model@1.0.0` / `@1.1.0`
- strict dual support for Bering `io.mb3r.bering.snapshot@1.0.0` / `@1.1.0`
- versioned `analysis` config `1.1` plus a separate Sheaft fault contract `1.0`
- typed `1.1.0` metadata loading for edge IDs, placements, shared resources, retries, timeouts, and observed summaries
- placement-aware and edge-aware advanced analysis, including timeout mismatch, retry amplification, blast radius, and asymmetric edge faults
- artifact-vs-artifact baseline comparison across contract lines through the existing `analysis.baselines` flow

Stable within the `v0.2.0` preview:

- strict acceptance of supported `1.0.0` and `1.1.0` Bering model/snapshot contracts
- deterministic batch execution and deterministic baseline comparison for a fixed artifact, config, and seed
- fail-stop baseline semantics for `1.0.0`
- honest advanced diagnostics for timeout mismatch, retry amplification, blast radius, and asymmetric edge faults when metadata exists

Still experimental in `v0.2.0`:

- long-running `serve` / `watch` service mode
- local `discover` helper
- broader operator-facing packaging and operational conventions around image/chart deployment

Known limitations:

- no live chaos execution, traffic generation, or new discovery pipeline is added in this release
- explicit legacy predicates remain service-based unless explicit path/journey information exists
- advanced metrics stay unavailable when the artifact or external contract does not provide the required metadata

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
