# CI Gate Integration

## Command Pattern

Use `sheaft run` in CI:

```bash
sheaft run \
  --input path/to/traces.json \
  --policy configs/gate.policy.example.yaml \
  --out-dir out \
  --seed 42
```

Generated artifacts:

- `out/model.json`
- `out/report.json`
- `out/summary.md`

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
      - name: Run Sheaft gate
        run: |
          docker run --rm -v "$PWD:/workspace" -w /workspace sheaft:ci run \
            --input examples/otel/traces.sample.json \
            --policy configs/gate.policy.example.yaml \
            --out-dir out \
            --seed 42
      - name: Upload artifacts
        uses: actions/upload-artifact@v4
        with:
          name: sheaft-report
          path: out/
```

