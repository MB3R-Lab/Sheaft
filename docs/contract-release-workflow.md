# Bering-Sheaft Contract Release Workflow

This document defines how Sheaft updates and verifies upstream Bering schema contract releases.

## Goal

Keep Sheaft as a strict downstream consumer:

- Bering owns schema publication and release metadata.
- Sheaft pins explicit contract name, version, URI, and digest.
- Sheaft only accepts contracts that match those pins exactly.

## Current Upstream Metadata Source

The canonical published Bering schema metadata currently consumed by Sheaft is:

- `https://mb3r-lab.github.io/Bering/schema/index.json`

Sheaft CI verifies that this published metadata matches the currently published `1.2.0` model and snapshot contracts in `internal/modelcontract/contract.go`.
Published remote schema sync is verified separately for every supported versioned model and snapshot schema mirror.

The current app-level Bering release/package sync target is:

- Release: `https://github.com/MB3R-Lab/Bering/releases/tag/v0.3.4`
- Release manifest asset: `release-manifest.json` (`sha256:d7f4b3d61ff8e36bd370dadd351e51a2b2803677ae12a5e0fbb81faacfd01e20`)
- Contracts pack asset: `bering-contracts_0.3.4.tar.gz` (`sha256:3865a8e7152c8a383f34d99334bc392cfd0ec793325b530d392723f67839ddbf`)
- Schema contract pins consumed by Sheaft remain explicit in `internal/modelcontract/contract.go` and currently include `1.0.0`, `1.1.0`, and `1.2.0`.

## Release Policy

### Non-breaking upstream schema release

When Bering publishes a new contract line that Sheaft wants to support:

1. Bering publishes the new schema file at a stable versioned URI.
2. Bering updates `schema/index.json` so `name`, `version`, `uri`, and `digest` match the released schema.
3. Sheaft updates:
   - `internal/modelcontract/contract.go`
   - vendored versioned schema snapshots under `internal/modelcontract/schema/`
   - mirrored public versioned schemas under `api/schema/`
   - `compatibility-manifest.json`
   - [compatibility matrix](compatibility-matrix.md)
4. Sheaft CI must pass:
   - remote schema sync check
   - Bering release metadata check
   - compatibility matrix check
   - `go test ./...`

### Upstream app/package release with unchanged schemas

When Bering publishes a product release that keeps the same schema contract lines:

1. confirm the Bering release and package assets are published;
2. update `tested_bering_app_versions` through the compatibility manifest generator;
3. update `README.md`, [compatibility matrix](compatibility-matrix.md), and this workflow note with the Bering release/package checkpoint;
4. regenerate `compatibility-manifest.json`;
5. run the compatibility-manifest check and `go test ./...`.

### Breaking upstream schema release

If the new Bering contract is not backward compatible for Sheaft:

1. do not silently replace the existing pin;
2. either add a new supported contract entry or keep the current line pinned;
3. update the compatibility matrix to show the supported release lines explicitly;
4. document migration impact in the Sheaft PR.

## Sheaft Release Checklist

- Confirm the Bering release metadata manifest is already updated upstream.
- Update the Sheaft contract constants and supported contract list.
- Refresh vendored and mirrored schema files.
- Regenerate `compatibility-manifest.json`.
- Update [compatibility matrix](compatibility-matrix.md).
- Run:
  - `sh scripts/ci/check-bering-release-metadata.sh`
  - `sh scripts/ci/check-remote-schema-sync.sh`
  - `sh scripts/ci/check-compatibility-matrix.sh`
  - `sh scripts/ci/check-compatibility-manifest.sh`
  - `go test ./...`
- Mention the upstream Bering release metadata timestamp in the PR notes.

## Current Scope Limitation

`schema/index.json` tracks the latest published Bering contract line. Older supported lines remain pinned and verified through their exact published versioned schema URLs plus the checked-in versioned schema mirrors.
