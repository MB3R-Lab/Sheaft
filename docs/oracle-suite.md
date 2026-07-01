# Synthetic Oracle Suite

The synthetic oracle suite is release evidence for Sheaft's stochastic connectivity semantics. It uses generated in-memory topologies with closed-form expectations, then compares deterministic Monte Carlo output against documented confidence tolerances.

## Scope

The suite covers:

- synchronous chains with node and edge reliability
- fan-out `any path` fallback
- mandatory single-point dependencies
- replicated targets saturating at an edge bottleneck
- compact node-vs-edge sensitivity grid
- immediate-response invariance to async edges
- eventual-completion sensitivity to declared async edges
- correlated failure, timeout, and trace-incompleteness boundaries

## Run

```bash
make oracle-suite
```

Direct command:

```bash
go run ./cmd/releasectl oracle-suite --out-dir .tmp/oracle-suite
```

Generated outputs:

- `.tmp/oracle-suite/oracle-report.json`
- `.tmp/oracle-suite/summary.md`

`oracle-report.json` follows [api/schema/oracle-report.schema.json](../api/schema/oracle-report.schema.json).

## Tolerances

Each probabilistic check uses a deterministic seed and a per-check tolerance of:

- `0.001` for expected values at `0` or `1`
- otherwise, `max(0.012, 4 * sqrt(p * (1 - p) / trials))`

This keeps the suite compact enough for CI while making failures point to semantic regressions rather than incidental random drift.

## Release Use

`make release-dry-run` includes this suite. The fixed benchmark slice still proves repeatable behavior on a checked-in Bering fixture; the oracle suite proves the formal stochastic-connectivity behavior on synthetic closed-form cases.
