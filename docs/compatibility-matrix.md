# Bering-Sheaft Compatibility Matrix

Sheaft is a strict downstream consumer of Bering-produced artifacts. It does not auto-negotiate schema versions: only the contract pins declared in `internal/modelcontract/contract.go` are accepted.

## Current Matrix

| Sheaft line | Status | Bering model contract | Model URI | Model digest | Bering snapshot contract | Snapshot URI | Snapshot digest | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `main` (unreleased) | active | `io.mb3r.bering.model@1.0.0` | `https://mb3r-lab.github.io/Bering/schema/model/v1.0.0/model.schema.json` | `sha256:272277c093f37580adcd2dded225bd37c86539d642d7910baad7e4228227d1a7` | `io.mb3r.bering.snapshot@1.0.0` | `https://mb3r-lab.github.io/Bering/schema/snapshot/v1.0.0/snapshot.schema.json` | `sha256:0b1ff66a64419d5f2e838663451a739fe34b3871bc1ccb9102ebec0fb8ec0b83` | Strict pin validated by `artifact.Load` and CI checks. |

## Update Rules

- Update this matrix in the same PR that changes any Bering contract pin, URI, digest, or vendored schema snapshot.
- Keep `README.md`, `internal/modelcontract/contract.go`, and this matrix aligned.
- CI checks fail if the current contract pins are not represented here.
- CI also fails on pull requests that modify contract pin files without changing this matrix.

## Release Note

This repository currently has no Git tags or published Sheaft release lines. The `main` row therefore represents the active unreleased consumer pin. When versioned Sheaft releases start, add one row per released line and keep `main` as the forward-looking row.
