# Compatibility

Sheaft is a strict downstream consumer of Bering artifacts.

It does not negotiate schema versions at runtime. An incoming artifact is accepted only when its declared schema metadata matches a supported contract exactly:

- schema name
- schema version
- schema URI
- schema digest

## Source of Truth

Runtime truth lives in `internal/modelcontract/contract.go`.

`compatibility-manifest.json` is generated from that code and is validated in CI so it cannot drift from the current exact-match contract logic.

## What the Manifest Means

`compatibility-manifest.json` declares:

- supported upstream artifact types
- supported upstream schema names
- supported upstream schema versions
- required schema digests
- tested Bering app versions, if known

`tested_bering_app_versions` lists Bering product tags this Sheaft line has been app-level synced against. The current line records Bering `v1.0.0`; schema acceptance is still controlled only by `supported_contracts`.

An empty `tested_bering_app_versions` array would mean Sheaft has not published an app-level Bering compatibility statement beyond the schema pins that are already required.

## How Downstream CI Should Use It

Use `compatibility-manifest.json` before or alongside a Sheaft invocation when you need explicit policy around upstream artifact acceptance.

Recommended checks:

1. read the artifact metadata produced upstream;
2. compare the upstream schema name/version/digest against `supported_contracts`;
3. reject mismatches before promotion, or let Sheaft reject them at execution time;
4. treat the manifest as compatibility data, not as schema ownership.

## Project-Level Contract Policy

Global runtime support and project-level acceptance are separate.

- Global support is the explicit registry in `internal/modelcontract/contract.go`.
- Project-level acceptance is narrower and can be configured with `contract_policy`.

The project policy supports:

- `allowed_kinds`
- `allowed_contracts`
- `deprecated_action`
- `deprecated_contracts`

Usage examples:

```bash
sheaft run \
  --model examples/outputs/snapshot-v1.3.0.sample.json \
  --analysis configs/analysis.v1.1.example.yaml \
  --contract-policy configs/contract-policy.example.yaml \
  --out-dir out
```

```yaml
contract_policy:
  allowed_kinds: [model, snapshot]
  allowed_contracts:
    - kind: snapshot
      name: io.mb3r.bering.snapshot
      versions: ["1.3.0"]
```

When a contract is still supported globally but deprecated for a given project, Sheaft can either:

- continue and mark the report with `contract_policy.status=deprecated` plus `action=warn`
- fail before simulation with a contract policy error when `deprecated_action=fail`

## Current Scope

- Bering owns upstream schema publication and evolution.
- Sheaft v1 declares compatibility with the Bering `1.3.0` model/snapshot release line.
- Pre-v1 preview lines `1.0.0`, `1.1.0`, and `1.2.0` are retired on the current main line.
- `1.3.0` is the current v1 line for path-aware timeout, retry, placement, shared-resource, reliability evidence, endpoint semantics, and edge-scoped analysis when the artifact carries that metadata.
- The current app-level Bering release/package sync target is `v1.0.0`; the accepted Bering schema contract pins are tracked separately in the compatibility matrix.
- Missing advanced metadata is surfaced as unavailable rather than guessed.
- Changing the Sheaft app version does not automatically widen or narrow compatibility.

See [docs/compatibility-matrix.md](compatibility-matrix.md) for the human-readable matrix and [VERSIONING.md](../VERSIONING.md) for version-surface rules.
