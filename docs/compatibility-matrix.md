# Bering-Sheaft Compatibility Matrix

Sheaft is a strict downstream consumer of Bering-produced artifacts. It does not auto-negotiate schema versions: only the contract pins declared in `internal/modelcontract/contract.go` are accepted.

The machine-readable equivalent of this page is the repo-root `compatibility-manifest.json`.

## Current App-Level Sync

The current `main` line has been synced against the Bering `v1.0.0` release/package. This is recorded in `compatibility-manifest.json` as `tested_bering_app_versions: ["v1.0.0"]`.

That package sync does not widen or narrow the accepted upstream schema contracts. The strict contract pins are the `1.3.0` Bering model/snapshot lines below.

For the v1 major release claim, the `1.3.0` line is the Bering v1 contract line used for typed topology `G=(V,E,tau)`, strict positive replica counts `R:V -> N_{>0}`, reliability evidence, operation-aware edge identity, and producer endpoint semantic hints. Pre-v1 preview lines `1.0.0`, `1.1.0`, and `1.2.0` are retired on the current main line. See [v1-major-semantics.md](v1-major-semantics.md).

## Current Matrix

| Sheaft line | Status | Bering model contract | Model URI | Model digest | Bering snapshot contract | Snapshot URI | Snapshot digest | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `v1.2.x` | current stable | `io.mb3r.bering.model@1.3.0` | `https://mb3r-lab.github.io/Bering/schema/model/v1.3.0/model.schema.json` | `sha256:2aa8a3550a25dc626ba6d2f5833569efca2f382b9e5c9c3405be93695d7d48ae` | `io.mb3r.bering.snapshot@1.3.0` | `https://mb3r-lab.github.io/Bering/schema/snapshot/v1.3.0/snapshot.schema.json` | `sha256:cb778e5b0866d9ce5cfe7f23b8d98a339603593a0247cccd9cddaf05c7ae4bb1` | Adds Sheaft-owned analysis schema `1.2` failure-tolerance boundaries without changing accepted Bering contracts. |
| `main` (unreleased) | active v1 line | `io.mb3r.bering.model@1.3.0` | `https://mb3r-lab.github.io/Bering/schema/model/v1.3.0/model.schema.json` | `sha256:2aa8a3550a25dc626ba6d2f5833569efca2f382b9e5c9c3405be93695d7d48ae` | `io.mb3r.bering.snapshot@1.3.0` | `https://mb3r-lab.github.io/Bering/schema/snapshot/v1.3.0/snapshot.schema.json` | `sha256:cb778e5b0866d9ce5cfe7f23b8d98a339603593a0247cccd9cddaf05c7ae4bb1` | Enables operation-aware typed edge IDs, strict positive replica counts, service/edge reliability evidence, retry/timeout metadata, placement buckets, shared resources, endpoint semantic hints, and path-aware advanced diagnostics. |

## Comparison Rule

The existing `analysis.baselines` flow can compare supported artifacts on the current `1.3.0` line:

- primary `1.3.0` artifact vs baseline `1.3.0` artifact
- primary `1.3.0` artifact vs prior Sheaft report generated from a supported artifact

Overlapping metrics produce diffs. Metrics that are unavailable on one side remain in the diff with an explicit non-comparable reason.

## Update Rules

- Update this matrix in the same PR that changes any Bering contract pin, URI, digest, or vendored schema snapshot.
- Regenerate `compatibility-manifest.json` in the same PR so the machine-readable release contract stays aligned.
- Keep `README.md`, `internal/modelcontract/contract.go`, and this matrix aligned.
- CI checks fail if the current contract pins are not represented here.
- CI also fails if `compatibility-manifest.json` drifts from the current strict contract pins.
- CI also fails on pull requests that modify contract pin files without changing this matrix.

## Release Note

Published Sheaft releases through `v0.2.4` used Bering `1.0.0` and `1.1.0` contract pins. Sheaft `v1.0.0` through `v1.2.x` accept only the Bering `1.3.0` v1 contract line; the Sheaft analysis-schema `1.2` number is unrelated to the retired Bering preview contract `1.2.0`. When a future Sheaft release changes accepted upstream contracts, add one row per released line and keep `main` as the forward-looking row.
