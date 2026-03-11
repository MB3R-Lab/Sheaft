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

An empty `tested_bering_app_versions` array means Sheaft has not published an app-level Bering compatibility statement beyond the schema pins that are already required.

## How Downstream CI Should Use It

Use `compatibility-manifest.json` before or alongside a Sheaft invocation when you need explicit policy around upstream artifact acceptance.

Recommended checks:

1. read the artifact metadata produced upstream;
2. compare the upstream schema name/version/digest against `supported_contracts`;
3. reject mismatches before promotion, or let Sheaft reject them at execution time;
4. treat the manifest as compatibility data, not as schema ownership.

## Current Scope

- Bering owns upstream schema publication and evolution.
- Sheaft declares compatibility with Bering release lines.
- Changing the Sheaft app version does not automatically widen or narrow compatibility.

See [docs/compatibility-matrix.md](compatibility-matrix.md) for the human-readable matrix and [VERSIONING.md](../VERSIONING.md) for version-surface rules.
