# External Benchmark Contract

Heavy datasets and reproducibility bundles are intentionally out of scope for this repository.
The model contract is also owned externally by Bering.

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
- keep consumer compatibility with Bering schema snapshot (`internal/modelcontract/schema/model.schema.json`).
- fail CI if model metadata schema binding does not match pinned Bering contract.
