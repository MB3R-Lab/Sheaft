# Bering-Sheaft Compatibility Matrix

Sheaft is a strict downstream consumer of Bering-produced artifacts. It does not auto-negotiate schema versions: only the contract pins declared in `internal/modelcontract/contract.go` are accepted.

The machine-readable equivalent of this page is the repo-root `compatibility-manifest.json`.

## Current Matrix

| Sheaft line | Status | Bering model contract | Model URI | Model digest | Bering snapshot contract | Snapshot URI | Snapshot digest | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `main` (unreleased) | active baseline line | `io.mb3r.bering.model@1.0.0` | `https://mb3r-lab.github.io/Bering/schema/model/v1.0.0/model.schema.json` | `sha256:272277c093f37580adcd2dded225bd37c86539d642d7910baad7e4228227d1a7` | `io.mb3r.bering.snapshot@1.0.0` | `https://mb3r-lab.github.io/Bering/schema/snapshot/v1.0.0/snapshot.schema.json` | `sha256:87e4e887ed4a37b72f6136e268b73552eccb92941c4de2c6f3a514dd066ea972` | Stable fail-stop baseline semantics. Advanced metrics that need richer metadata remain unavailable unless supplied by a Sheaft-owned external contract. |
| `main` (unreleased) | active advanced line | `io.mb3r.bering.model@1.1.0` | `https://mb3r-lab.github.io/Bering/schema/model/v1.1.0/model.schema.json` | `sha256:bc9a60736c9e6bda9599243fd68f293b88f42ade65321d8267369a5c3214779a` | `io.mb3r.bering.snapshot@1.1.0` | `https://mb3r-lab.github.io/Bering/schema/snapshot/v1.1.0/snapshot.schema.json` | `sha256:53b127608b2aaa4fabb352b998cd6b2c5ed558764729a09abea56f4f9b40fa01` | Enables typed edge IDs, retry/timeout metadata, placement buckets, shared resources, and path-aware advanced diagnostics. |

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

This repository currently has no Git tags or published Sheaft release lines. The `main` row therefore represents the active unreleased consumer pin. When versioned Sheaft releases start, add one row per released line and keep `main` as the forward-looking row.
