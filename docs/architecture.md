# Architecture (MVP v0)

Sheaft v0 is a pre-release resilience gate built around a simple, explicit pipeline:

1. **Discovery**: parse OTel trace artifacts and infer a directed service dependency graph.
2. **Modeling**: normalize discovered services, edges, and entry endpoints into a typed model JSON.
3. **Simulation**: run Monte Carlo fail-stop availability estimation over blocking synchronous paths.
4. **Policy gate**: compare estimated availabilities with thresholds and emit pass/warn/fail decision.
5. **Reporting**: write machine-readable report JSON and a concise markdown summary.

## Runtime Components

- `cmd/sheaft`: CLI entrypoint.
- `internal/app`: command orchestration and exit code mapping.
- `internal/discovery/otel`: OTel trace ingestion and graph extraction.
- `internal/model`: model types, validation, and model file IO.
- `internal/simulation`: deterministic Monte Carlo engine (`seed` + fixed params).
- `internal/gate`: policy evaluation (`warn` / `fail` / `report`).
- `internal/report`: report composition + JSON/markdown output.
- `internal/config`: policy/config loading and validation.

## Data Contracts

- `api/schema/model.schema.json`
- `api/schema/policy.schema.json`
- `api/schema/report.schema.json`

The schemas are intentionally minimal but stable for MVP workflows and CI usage.

