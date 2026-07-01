# Bering-Sheaft Compatibility Matrix

Sheaft is a strict downstream consumer of Bering-produced artifacts. It does not auto-negotiate schema versions: only the contract pins declared in `internal/modelcontract/contract.go` are accepted.

The machine-readable equivalent of this page is the repo-root `compatibility-manifest.json`.

## Current App-Level Sync

The current `main` line has been synced against the published Bering `v0.3.4` release/package. This is recorded in `compatibility-manifest.json` as `tested_bering_app_versions: ["v0.3.4"]`.

That package sync does not widen or narrow the accepted upstream schema contracts. The strict contract pins are the `1.0.0`, `1.1.0`, and `1.2.0` Bering model/snapshot lines below.

For the v1 major release claim, the `1.2.0` line is the Bering v1 contract line used for typed topology `G=(V,E,tau)`, reliability evidence, operation-aware edge identity, and producer endpoint semantic hints. The `1.0.0` line remains accepted as the baseline comparison reference; `1.1.0` remains accepted as a historical advanced-preview contract. See [v1-major-semantics.md](v1-major-semantics.md).

## Current Matrix

| Sheaft line | Status | Bering model contract | Model URI | Model digest | Bering snapshot contract | Snapshot URI | Snapshot digest | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `main` (unreleased) | active baseline line | `io.mb3r.bering.model@1.0.0` | `https://mb3r-lab.github.io/Bering/schema/model/v1.0.0/model.schema.json` | `sha256:272277c093f37580adcd2dded225bd37c86539d642d7910baad7e4228227d1a7` | `io.mb3r.bering.snapshot@1.0.0` | `https://mb3r-lab.github.io/Bering/schema/snapshot/v1.0.0/snapshot.schema.json` | `sha256:87e4e887ed4a37b72f6136e268b73552eccb92941c4de2c6f3a514dd066ea972` | Stable fail-stop baseline semantics. Advanced metrics that need richer metadata remain unavailable unless supplied by a Sheaft-owned external contract. |
| `main` (unreleased) | historical advanced-preview line | `io.mb3r.bering.model@1.1.0` | `https://mb3r-lab.github.io/Bering/schema/model/v1.1.0/model.schema.json` | `sha256:bc9a60736c9e6bda9599243fd68f293b88f42ade65321d8267369a5c3214779a` | `io.mb3r.bering.snapshot@1.1.0` | `https://mb3r-lab.github.io/Bering/schema/snapshot/v1.1.0/snapshot.schema.json` | `sha256:53b127608b2aaa4fabb352b998cd6b2c5ed558764729a09abea56f4f9b40fa01` | Preserved published contract line. It remains accepted, but the v1 major semantic claim uses `1.2.0`. |
| `main` (unreleased) | active v1 advanced line | `io.mb3r.bering.model@1.2.0` | `https://mb3r-lab.github.io/Bering/schema/model/v1.2.0/model.schema.json` | `sha256:4fa1a34e64703524cfe2289341fcea79986265db08c0220d6c89e38c0ff76bf8` | `io.mb3r.bering.snapshot@1.2.0` | `https://mb3r-lab.github.io/Bering/schema/snapshot/v1.2.0/snapshot.schema.json` | `sha256:cb737b0a4038e0bf30a397ca7ba7ff017d684fe3b25e7d8e3ae74ac59b45210b` | Enables operation-aware typed edge IDs, service/edge reliability evidence, retry/timeout metadata, placement buckets, shared resources, endpoint semantic hints, and path-aware advanced diagnostics. |

## Comparison Rule

`1.0.0` is the comparison reference line in Sheaft. The existing `analysis.baselines` flow can compare:

- primary `1.2.0` artifact vs baseline `1.0.0` artifact
- primary `1.2.0` artifact vs baseline `1.1.0` or `1.2.0` artifact
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

Published Sheaft releases through `v0.2.4` use the accepted Bering `1.0.0` and `1.1.0` contract pins. The `main` rows represent the active post-`v0.2.4` consumer pins, including the new `1.2.0` v1-major contract line; when a future Sheaft release changes accepted upstream contracts, add one row per released line and keep `main` as the forward-looking row.
