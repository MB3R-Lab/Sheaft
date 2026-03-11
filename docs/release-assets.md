# Release Assets

The release payload is designed to be consumed by automation first.

The canonical files are:

- `dist/`
- `compatibility-manifest.json`
- `release-manifest.json`

## Asset Inventory

`release-manifest.json` describes:

- product name
- app version
- git commit
- build date
- platform archives
- per-archive checksums
- per-archive SBOM references
- OCI image references and digest
- OCI chart name and version
- compatibility manifest reference
- default config pack references

## Why Two Manifests Exist

`compatibility-manifest.json` answers:

- which upstream Bering contracts does this Sheaft line accept?

`release-manifest.json` answers:

- which assets belong to this Sheaft release payload?

Keep them separate:

- compatibility changes track Bering schema support
- release-manifest changes track build outputs for a specific Sheaft release

## CI Consumption Pattern

For GitHub, GitLab, Jenkins, or an internal release pipeline:

1. resolve the desired Sheaft release;
2. read `release-manifest.json`;
3. download the referenced binary/image/chart/config-pack assets;
4. verify checksums;
5. optionally inspect `compatibility-manifest.json` before allowing a Bering artifact into the next stage.

## Default Config Pack

The default config pack is a first-class release asset so downstream automation can start from versioned examples instead of copying files from the repository tree.

Use it when:

- bootstrapping CI gates
- building a first Kubernetes values override
- creating a baseline report bundle for tests
