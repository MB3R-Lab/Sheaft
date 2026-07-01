# Fixed Benchmark Slice

The fixed benchmark slice is the in-repository release-quality check for Sheaft-on-Bering behavior. It is intentionally small: it proves deterministic execution and report quality on a stable public fixture, while larger datasets stay in the external benchmark repository described in [benchmark-external.md](benchmark-external.md).

## Scope

The slice uses:

- manifest: [benchmarks/fixed-slice/manifest.json](../benchmarks/fixed-slice/manifest.json)
- artifact: [examples/outputs/snapshot-v1.2.0.sample.json](../examples/outputs/snapshot-v1.2.0.sample.json)
- analysis config: [configs/analysis.v1.1.example.yaml](../configs/analysis.v1.1.example.yaml)

The manifest fixes the expected contract, profile count, decision, confidence floor, advanced-metric availability, baseline-diff requirement, and stable-report repeatability requirement.

## Run

```bash
make benchmark-slice
```

Direct command:

```bash
go run ./cmd/releasectl benchmark-slice \
  --manifest benchmarks/fixed-slice/manifest.json \
  --out-dir .tmp/benchmark-slice
```

Generated outputs:

- `.tmp/benchmark-slice/model.json`
- `.tmp/benchmark-slice/report.json`
- `.tmp/benchmark-slice/summary.md`
- `.tmp/benchmark-slice/quality-report.json`

## Quality Metrics

`quality-report.json` records:

- gate decision stability against the fixed expectation
- profile count
- artifact confidence
- cross-profile weighted availability
- unavailable advanced metric count
- baseline diff count
- stable report hash from two repeated runs

The benchmark passes only when every check in `checks[]` has `status: "pass"`.

## Release Use

`make release-dry-run` includes this slice. That makes the fixed benchmark part of the local release evidence without depending on heavy external datasets.
