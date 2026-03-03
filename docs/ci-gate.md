# CI Gate Integration

## Command Pattern

Use `sheaft run` in CI:

```bash
sheaft run \
  --model path/to/bering-model.json \
  --policy configs/gate.policy.example.yaml \
  --out-dir out \
  --seed 42
```

Optional:

```bash
--journeys path/to/journeys.json
```

Generated artifacts:

- `out/model.json`
- `out/report.json`
- `out/summary.md`

`out/model.json` is a validated copy of the input Bering model.

## Exit Codes

- `0`: pass / warn / report
- `2`: policy fail in `mode=fail`
- `1`: input/config/runtime error

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
      - name: Fetch model artifact from Bering step
        run: |
          mkdir -p artifacts
          cp path/from/previous-step/bering-model.json artifacts/bering-model.json
      - name: Run Sheaft gate
        run: |
          docker run --rm -v "$PWD:/workspace" -w /workspace sheaft:ci run \
            --model artifacts/bering-model.json \
            --policy configs/gate.policy.example.yaml \
            --out-dir out \
            --seed 42
      - name: Upload artifacts
        uses: actions/upload-artifact@v4
        with:
          name: sheaft-report
          path: out/
```
