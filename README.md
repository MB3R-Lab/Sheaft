# Sheaft

Sheaft is a resilience posture engine for model artifacts produced upstream by Bering or another compatible producer.

It supports two operating modes:

- CI/CD batch gating from externally produced artifacts.
- Continuous posture monitoring from watched model or snapshot artifacts.

Sheaft stays downstream of topology discovery. `discover` still exists as a local helper, but it is not the production discovery path.

## What It Does

- `simulate`: consume a model or snapshot artifact and write a posture report.
- `gate`: evaluate a report against the legacy policy subset or the richer analysis rules.
- `run`: one-shot batch pipeline for CI/CD.
- `serve` / `watch`: long-running posture service with HTTP status, diff, history, and metrics endpoints.

The current batch commands remain supported:

```bash
sheaft simulate --model <artifact.json> --policy <policy.yaml> --out <report.json> --seed 42
sheaft gate --report <report.json> --policy <policy.yaml>
sheaft run --model <artifact.json> --policy <policy.yaml> --out-dir out --seed 42
```

The richer path is additive:

```bash
sheaft simulate --model <artifact.json> --analysis configs/analysis.example.yaml --out out/report.json
sheaft run --model <artifact.json> --analysis configs/analysis.example.yaml --out-dir out
sheaft serve --config configs/sheaft.example.yaml
```

## Supported Upstream Contracts

Sheaft validates artifacts against an explicit whitelist.

- `io.mb3r.bering.model@1.0.0`
- `io.mb3r.bering.snapshot@1.0.0`

Unknown contracts are rejected with an error that lists the supported contracts. There is no silent fallback for unsupported upstream schemas.

## Quickstart

### Batch Gate

```bash
sheaft run \
  --model examples/outputs/model.sample.json \
  --policy configs/gate.policy.example.yaml \
  --out-dir examples/outputs/generated \
  --seed 42
```

### Advanced Batch Analysis

```bash
sheaft run \
  --model examples/outputs/snapshot.sample.json \
  --analysis configs/analysis.example.yaml \
  --out-dir examples/outputs/posture
```

### Long-Running Service

```bash
sheaft serve --config configs/sheaft.example.yaml
```

HTTP endpoints:

- `/healthz`
- `/readyz`
- `/status`
- `/current-report`
- `/current-diff`
- `/history`
- `/metrics`

## Analysis Model

The richer analysis config supports:

- named scenario profiles
- per-profile sampling mode and failure settings
- weighted endpoint aggregates
- baseline report comparison
- external predicate and workload overlays for legacy models
- explicit multi-profile gate evaluation rules

Supported sampling modes:

- `independent_replica`
- `independent_service`
- `fixed_k_service_set`

Supported predicate types:

- `all_of`
- `any_of`
- `k_of_n`

If no richer predicate definition is supplied, Sheaft falls back to legacy path-based journey evaluation.

## Metrics

Prometheus/OpenMetrics output includes:

- `recomputes_total`
- `recompute_duration_seconds`
- `current_model_age_seconds`
- `current_profile_aggregate_availability`
- `current_endpoint_availability`
- `endpoints_below_threshold`
- `active_model_info`
- `active_topology_version`
- `previous_gap`
- `baseline_gap`

Example scrape config: [examples/prometheus/prometheus.sheaft.yml](examples/prometheus/prometheus.sheaft.yml)  
Example Grafana dashboard: [examples/grafana/sheaft.posture.dashboard.json](examples/grafana/sheaft.posture.dashboard.json)

## Repository Layout

```text
cmd/sheaft                 CLI entrypoint
internal/app               CLI orchestration
internal/artifact          supported artifact adapters and contract loading
internal/analyzer          shared batch/service analysis pipeline
internal/simulation        deterministic multi-profile simulation engine
internal/gate              gate evaluation
internal/report            rich posture reports and diffs
internal/service           watch/serve mode and metrics
internal/config            legacy policy + richer analysis/serve config loading
api/schema                 JSON schemas
configs                    example configs
examples                   sample artifacts, dashboard, and scrape config
docs                       architecture, methodology, migration, and config docs
```

## Docs

- [Architecture](docs/architecture.md)
- [Methodology](docs/methodology.md)
- [Configuration and Schemas](docs/configuration.md)
- [CI Gate](docs/ci-gate.md)
- [Service Mode](docs/observability-mode.md)
- [Migration Notes](docs/migration.md)
- [Assumptions and Limitations](docs/assumptions-and-limitations.md)

## Example Artifacts

- Plain model: [examples/outputs/model.sample.json](examples/outputs/model.sample.json)
- Snapshot envelope: [examples/outputs/snapshot.sample.json](examples/outputs/snapshot.sample.json)
- Legacy report sample: [examples/outputs/report.sample.json](examples/outputs/report.sample.json)
- Rich report sample: [examples/outputs/posture-generated/report.json](examples/outputs/posture-generated/report.json)
- Rich summary sample: [examples/outputs/posture-generated/summary.md](examples/outputs/posture-generated/summary.md)
- Predicate overlay: [configs/predicate-contract.example.yaml](configs/predicate-contract.example.yaml)

## Exit Codes

- `0`: success / pass / warn / report
- `2`: gate failure in `mode=fail`
- `1`: runtime, config, or input error

## Notes

- Determinism is preserved for a fixed seed and config.
- Production topology discovery remains upstream.
- `discover` is retained as an experimental local helper only.

## License

MIT, see [LICENSE](LICENSE).
