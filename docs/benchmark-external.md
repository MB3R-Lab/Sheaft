# External Benchmark Contract

Heavy datasets and reproducibility bundles are intentionally out of scope for this repository.

## Contract for External Benchmark Repository

The external benchmark repository should provide:

- versioned trace datasets and topology snapshots;
- chaos experiment manifests and run scripts;
- ground-truth availability measurements;
- replay scripts that regenerate `model.json` and `report.json`;
- release-tagged quality reports.

## Integration in This Repo

- keep only small fixtures in `test/fixtures/`;
- reference benchmark run IDs in PRs/releases;
- keep schema compatibility (`api/schema/*.json`) as a stable interface.

