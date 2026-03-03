# Architecture (MVP v0)

Sheaft v0 is a pre-release resilience gate built around a simple, explicit pipeline:

1. **Discovery (external)**: Bering produces a typed model JSON.
2. **Contract check**: Sheaft validates exact schema binding (`name/version/uri/digest`).
3. **Simulation**: run Monte Carlo fail-stop availability estimation over blocking synchronous paths.
4. **Policy gate**: compare estimated availabilities with thresholds and emit pass/warn/fail decision.
5. **Reporting**: write machine-readable report JSON and a concise markdown summary.

## Runtime Components

- `cmd/sheaft`: CLI entrypoint.
- `internal/app`: command orchestration and exit code mapping.
- `internal/discovery/otel`: experimental local discovery helper.
- `internal/model`: model types, validation, and model file IO.
- `internal/modelcontract`: strict Bering schema pinning and vendored snapshot.
- `internal/simulation`: deterministic Monte Carlo engine (`seed` + fixed params).
- `internal/gate`: policy evaluation (`warn` / `fail` / `report`).
- `internal/report`: report composition + JSON/markdown output.
- `internal/config`: policy/config loading and validation.

## Data Contracts

- `api/schema/model.schema.json`
- `api/schema/policy.schema.json`
- `api/schema/report.schema.json`

Model schema ownership is external (Bering).  
This repository keeps a pinned snapshot for strict consumer-side validation.
