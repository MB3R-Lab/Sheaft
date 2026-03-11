# CI Gate

## Legacy Policy Flow

```bash
sheaft run \
  --model path/to/model.json \
  --policy configs/gate.policy.example.yaml \
  --out-dir out \
  --seed 42
```

Generated artifacts:

- `out/model.json`
- `out/report.json`
- `out/summary.md`

## Rich Analysis Flow

```bash
sheaft run \
  --model path/to/artifact.json \
  --analysis configs/analysis.example.yaml \
  --out-dir out
```

Use the richer config when CI needs:

- multiple scenario profiles
- weighted aggregates
- baseline comparisons
- external predicate overlays
- explicit multi-profile gate rules

## Exit Codes

- `0`: pass / warn / report
- `2`: gate failure in `mode=fail`
- `1`: input, contract, config, or runtime error

## GitHub Actions Example

```yaml
name: sheaft-gate
on: [pull_request]
jobs:
  resilience:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Build image
        run: docker build -f build/Dockerfile -t sheaft:ci .
      - name: Fetch upstream artifact
        run: |
          mkdir -p artifacts
          cp path/from/previous-step/model-or-snapshot.json artifacts/input.json
      - name: Run Sheaft
        run: |
          docker run --rm -v "$PWD:/workspace" -w /workspace sheaft:ci run \
            --model artifacts/input.json \
            --analysis configs/analysis.example.yaml \
            --out-dir out
      - name: Upload artifacts
        uses: actions/upload-artifact@v4
        with:
          name: sheaft-report
          path: out/
```
