# Sheaft

## Research basis

1. [**Model Discovery and Graph Simulation: A Lightweight Gateway to Chaos Engineering**](https://www.alphaxiv.org/abs/2506.11176)  
   The 48th IEEE/ACM International Conference on Software Engineering - New Ideas and Emerging Results (ICSE-NIER 2026)
2. [**Evaluating Asynchronous Semantics in Trace-Discovered Resilience Models: A Case Study on the OpenTelemetry Demo**](https://www.alphaxiv.org/abs/2512.12314)  
   The 40th International Conference on Advanced Information Networking and Applications (AINA-2026)

Sheaft is a pre-release resilience gate for microservice systems.  
It turns existing engineering artifacts into an explicit dependency model, runs fast graph-based availability simulation, and emits a release decision (`pass` / `warn` / `fail`) plus machine-readable artifacts.

## What it is

Sheaft provides an MVP workflow for:

1. `discover`: OTel trace artifacts -> dependency model JSON.
2. `simulate`: model + policy -> endpoint availability estimates.
3. `gate`: compare estimates against policy thresholds.
4. `run`: one-shot pipeline for CI/CD.

This is an artifact-derived, low-risk complement to live chaos engineering, not a replacement for all live validation.

## Why this exists

Live chaos campaigns are valuable but expensive and operationally constrained when used broadly and continuously.  
Sheaft focuses on making resilience checks:

- cheap (uses existing traces/configured policy),
- safe (no direct production fault injection),
- regular (CI-friendly, repeatable in minutes).

## How it works

`discover -> model -> simulate -> gate`

- **Discover**: infer service graph and endpoints from OTel traces.
- **Model**: normalize to typed schema with metadata/confidence.
- **Simulate**: run fail-stop Monte Carlo over blocking synchronous dependencies.
- **Gate**: evaluate availability thresholds and emit decision.

## MVP scope (v0)

- OTel-first input (`.json`) for model discovery.
- Fail-stop, independent crash approximation.
- Sync/blocking path emphasis for immediate HTTP success.
- Async semantics are present but not primary for v0 gate decisions.
- Default policy behavior is `warn` (non-blocking).

## Repository layout

```text
cmd/sheaft                 CLI entrypoint
internal/app               command orchestration
internal/discovery/otel    OTel trace -> graph discovery
internal/model             domain model + validation + IO
internal/simulation        Monte Carlo availability engine
internal/gate              policy evaluation
internal/report            JSON/markdown reporting
internal/config            policy/config parsing
api/schema                 model/policy/report JSON schemas
configs                    example runtime + policy configs
examples                   sample traces + sample outputs
docs                       architecture/methodology/limits/roadmap
scripts/ci                 CI helper script
test                       fixtures + integration/e2e tests
```

## Quickstart (Docker-first)

### 1) Build image

```bash
docker build -f build/Dockerfile -t sheaft:dev .
```

### 2) Run end-to-end pipeline

```bash
docker run --rm -v "$PWD:/workspace" -w /workspace sheaft:dev run \
  --input examples/otel/traces.sample.json \
  --policy configs/gate.policy.example.yaml \
  --out-dir examples/outputs/generated \
  --seed 42
```

### 3) Inspect artifacts

```bash
cat examples/outputs/generated/report.json
cat examples/outputs/generated/summary.md
```

## CLI

```bash
sheaft discover --input <trace-file|dir> --out <model.json>
sheaft simulate --model <model.json> --policy <policy.yaml> --out <report.json> --seed <int>
sheaft gate --report <report.json> --policy <policy.yaml> --mode warn|fail|report
sheaft run --input <trace-file|dir> --policy <policy.yaml> --out-dir <dir> --seed <int>
```

Exit codes:

- `0`: success / pass / warn / report
- `2`: policy fail (when mode is `fail`)
- `1`: runtime/config/input error

## CI integration

See [docs/ci-gate.md](docs/ci-gate.md) for a full workflow snippet.

Minimal GitHub Actions step:

```yaml
- name: Run Sheaft gate
  run: |
    docker run --rm -v "$PWD:/workspace" -w /workspace sheaft:dev run \
      --input examples/otel/traces.sample.json \
      --policy configs/gate.policy.example.yaml \
      --out-dir out \
      --seed 42
```

## Outputs

- Model schema: `api/schema/model.schema.json`
- Policy schema: `api/schema/policy.schema.json`
- Report schema: `api/schema/report.schema.json`
- Sample model: `examples/outputs/model.sample.json`
- Sample report: `examples/outputs/report.sample.json`

## Limitations

Current MVP limitations are explicit:

- connectivity-first approximation (graph + replicas);
- no correlated or gray-failure modeling in v0;
- async edge treatment has limited effect for immediate HTTP SLO in the studied benchmark case.

See [docs/assumptions-and-limitations.md](docs/assumptions-and-limitations.md).

## Roadmap

- Epic index: https://github.com/MB3R-Lab/Sheaft/issues/71
- R1-R10 epics: [docs/roadmap.md](docs/roadmap.md)

## License

MIT, see [LICENSE](LICENSE).
