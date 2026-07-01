# Release Assets

The release payload is designed to be consumed by automation first.

The canonical files are:

- `dist/`
- `compatibility-manifest.json`
- `release-manifest.json` as a generated release asset, not a committed `main`-branch file

## Asset Inventory

The release asset `release-manifest.json` describes:

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

The GitHub Release also publishes release evidence generated during the release job:

- fixed benchmark slice `quality-report.json` from the benchmark manifest shipped in the default config pack
- synthetic oracle suite `oracle-report.json` for closed-form stochastic-connectivity semantics

## V1 Major Release Checklist

Before publishing the v1 major line, `make release-dry-run` must pass with:

- schema sync against the pinned Bering `1.0.0` and `1.1.0` contracts
- `validate-v1-release-docs`, which checks the v1 semantics documentation, compatibility matrix, schema files, and checked-in examples
- checked-in smoke examples
- fixed benchmark slice
- synthetic oracle suite
- report/gate compatibility through the existing sample policy and analysis configs

Release notes for v1 must point at [v1-major-semantics.md](v1-major-semantics.md) and must not claim automatic probability calibration, arbitrary non-product `P`, live chaos execution, or rich temporal workflow models.

## Why Two Manifests Exist

`compatibility-manifest.json` answers:

- which upstream Bering contracts does this Sheaft line accept?
- which Bering app release packages has this Sheaft line been synced against, if known?

`release-manifest.json` answers:

- which assets belong to this Sheaft release payload?

Keep them separate:

- compatibility changes track Bering schema support
- tested app-version entries track app-level Bering release/package syncs that keep the same schema pins
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

Pack contents are release-specific. A newer repository `main` branch can contain examples that are not part of the latest published tag, so downstream consumers should trust the tagged pack archive and `release-manifest.json`, not the moving repository head.

Use it when:

- bootstrapping CI gates
- building a first Kubernetes values override
- creating a baseline report bundle for tests
- running the fixed benchmark slice manifest that ships with the default examples
- running the synthetic oracle suite for release semantic evidence
