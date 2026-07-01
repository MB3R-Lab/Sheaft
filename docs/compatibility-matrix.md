# Bering-Sheaft Compatibility Matrix

Sheaft is a strict downstream consumer of Bering-produced artifacts. It does not auto-negotiate schema versions: only the contract pins declared in `internal/modelcontract/contract.go` are accepted.

The machine-readable equivalent of this page is the repo-root `compatibility-manifest.json`.

## Current App-Level Sync

The current `main` line has been synced against the published Bering `v0.3.4` release/package. This is recorded in `compatibility-manifest.json` as `tested_bering_app_versions: ["v0.3.4"]`.

That package sync does not widen or narrow the accepted upstream schema contracts. The strict contract pins remain the `1.0.0` and `1.1.0` Bering model/snapshot lines below.

For the v1 major release claim, the `1.1.0` line is the Bering v1 contract line used for typed topology `G=(V,E,tau)`, reliability evidence, operation-aware edge identity, and producer endpoint semantic hints. The `1.0.0` line remains accepted as the baseline comparison reference. See [v1-major-semantics.md](v1-major-semantics.md).

## Current Matrix

| Sheaft line | Status | Bering model contract | Model URI | Model digest | Bering snapshot contract | Snapshot URI | Snapshot digest | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `main` (unreleased) | active baseline line | `io.mb3r.bering.model@1.0.0` | `https://mb3r-lab.github.io/Bering/schema/model/v1.0.0/model.schema.json` | `sha256:272277c093f37580adcd2dded225bd37c86539d642d7910baad7e4228227d1a7` | `io.mb3r.bering.snapshot@1.0.0` | `https://mb3r-lab.github.io/Bering/schema/snapshot/v1.0.0/snapshot.schema.json` | `sha256:87e4e887ed4a37b72f6136e268b73552eccb92941c4de2c6f3a514dd066ea972` | Stable fail-stop baseline semantics. Advanced metrics that need richer metadata remain unavailable unless supplied by a Sheaft-owned external contract. |
| `main` (unreleased) | active advanced line | `io.mb3r.bering.model@1.1.0` | `https://mb3r-lab.github.io/Bering/schema/model/v1.1.0/model.schema.json` | `sha256:76b2b22422b6e64f437fb144a02b6bd4629bf510cec5479a8496c41eb25fc406` | `io.mb3r.bering.snapshot@1.1.0` | `https://mb3r-lab.github.io/Bering/schema/snapshot/v1.1.0/snapshot.schema.json` | `sha256:c669dbc483ca8cfe1f58f994b6041a6767fdaa3df4fb5ae27d8253607b3f5cb5` | Enables operation-aware typed edge IDs, service/edge reliability evidence, retry/timeout metadata, placement buckets, shared resources, endpoint semantic hints, and path-aware advanced diagnostics. |

## Comparison Rule

`1.0.0` is the comparison reference line in Sheaft. The existing `analysis.baselines` flow can compare:

- primary `1.1.0` artifact vs baseline `1.0.0` artifact
- primary `1.1.0` artifact vs baseline `1.1.0` artifact
- primary `1.0.0` artifact vs prior Sheaft report or another `1.0.0` artifact

Overlapping metrics produce diffs. Metrics that are unavailable on one side remain in the diff with an explicit non-comparable reason.

## Update Rules

- Update this matrix in the same PR that changes any Bering contract pin, URI, digest, or vendored schema snapshot.
- Regenerate `compatibility-manifest.json` in the same PR so the machine-readable release contract stays aligned.
- Keep `README.md`, `internal/modelcontract/contract.go`, and this matrix aligned.
- CI checks fail if the current contract pins are not represented here.
- CI also fails if `compatibility-manifest.json` drifts from the current strict contract pins.
- CI also fails on pull requests that modify contract pin files without changing this matrix.

## Release Note

Published Sheaft releases through `v0.2.4` use the same accepted Bering `1.0.0` and `1.1.0` contract pins. The `main` row represents the active post-`v0.2.4` consumer pin; when a future Sheaft release changes accepted upstream contracts, add one row per released line and keep `main` as the forward-looking row.
