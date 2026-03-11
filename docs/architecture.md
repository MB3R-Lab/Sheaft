# Architecture

Sheaft is a downstream consumer of resilience model artifacts.

## Core Flow

1. An upstream producer writes a plain model artifact or snapshot envelope.
2. `internal/artifact` validates the declared contract against a supported whitelist and adapts the artifact into a normalized internal model.
3. `internal/analyzer` resolves predicates, optional overlays, journeys, baselines, and profile configuration.
4. `internal/simulation` runs deterministic Monte Carlo analysis for one or more profiles.
5. `internal/gate` applies explicit gate rules across endpoints, aggregates, and profiles.
6. `internal/report` emits JSON and markdown outputs plus diffs versus previous and baseline reports.
7. `internal/service` optionally turns the same pipeline into a long-running HTTP posture service.

## Design Principles

- Discovery ownership stays upstream.
- Batch and service mode share the same analysis engine.
- Contract handling is explicit and adapter-based.
- Legacy simple policy flow remains available.
- Richer predicate and workload data can come from the artifact, snapshot, or external overlay.

## Main Packages

- `internal/artifact`: plain model and snapshot envelope readers, contract compatibility, provenance normalization.
- `internal/config`: legacy policy loading plus versioned analysis and serve configs.
- `internal/analyzer`: shared orchestration for batch and service analysis.
- `internal/simulation`: multi-profile sampling, weighted aggregation, and legacy journey fallback.
- `internal/gate`: profile-aware gate evaluation.
- `internal/report`: richer posture reports, diffs, and summary rendering.
- `internal/service`: watch loop, bounded history, HTTP endpoints, and Prometheus/OpenMetrics metrics.

## Service Mode

`sheaft serve` watches:

- a single artifact file
- a directory of artifacts
- a stable path that is updated in place

On each new artifact it recomputes posture, updates history, refreshes metrics, and serves:

- `/healthz`
- `/readyz`
- `/status`
- `/current-report`
- `/current-diff`
- `/history`
- `/metrics`
