# Methodology (MVP)

This repository follows a connectivity-first resilience estimation method:

1. Receive a model artifact from Bering.
2. Enforce strict contract binding by schema metadata (`name/version/uri/digest` exact match).
3. Assume independent fail-stop crashes with per-service replica modeling.
4. Estimate endpoint availability as success rate across Monte Carlo trials.
5. Use policy thresholds to gate release risk.

## Immediate HTTP Success Semantics (v0)

- Blocking synchronous edges are part of required request paths.
- Asynchronous/non-blocking edges are excluded from immediate HTTP success checks.
- Endpoint succeeds when all services in at least one discovered path are alive (`OR` over paths, `AND` within a path).
- Journey paths are discovered from each endpoint `entry_service` on the blocking synchronous subgraph.
- Optional external journey contract can override auto-discovered paths (`--journeys`, schema in `api/schema/journeys.schema.json`).

## Reproducibility Controls

- Fixed random `seed`.
- Explicit simulation parameters (`trials`, `failure_probability`).
- Stable JSON outputs suitable for versioned CI artifacts.
- Pinned external model schema version and digest.
