# Bering-Sheaft Contract Release Workflow

This document defines how Sheaft updates and verifies upstream Bering schema contract releases.

## Goal

Keep Sheaft as a strict downstream consumer:

- Bering owns schema publication and release metadata.
- Sheaft pins explicit contract name, version, URI, and digest.
- Sheaft only accepts contracts that match those pins exactly.

## Current Upstream Metadata Source

The canonical published Bering release manifest currently consumed by Sheaft is:

- `https://mb3r-lab.github.io/Bering/schema/index.json`

Sheaft CI verifies that this published metadata matches the pinned model contract in `internal/modelcontract/contract.go`.
Published remote schema sync is verified separately for both the model and snapshot contracts.

## Release Policy

### Non-breaking upstream schema release

When Bering publishes a new contract line that Sheaft wants to support:

1. Bering publishes the new schema file at a stable versioned URI.
2. Bering updates `schema/index.json` so `name`, `version`, `uri`, and `digest` match the released schema.
3. Sheaft updates:
   - `internal/modelcontract/contract.go`
   - vendored schema snapshot under `internal/modelcontract/schema/`
   - mirrored public schema under `api/schema/`
   - `compatibility-manifest.json`
   - [compatibility matrix](compatibility-matrix.md)
4. Sheaft CI must pass:
   - remote schema sync check
   - Bering release metadata check
   - compatibility matrix check
   - `go test ./...`

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

The published upstream release manifest currently checked by Sheaft still covers the Bering model contract only. Snapshot compatibility is now verified via the published snapshot schema URL and strict local pins, but not yet via an equivalent upstream snapshot release-manifest document.
