# Methodology (MVP)

This repository follows a connectivity-first resilience estimation method:

1. Build dependency graph from artifacts already available to teams (v0: OTel traces).
2. Mark entry endpoints and their entry services.
3. Assume independent fail-stop crashes with per-service replica modeling.
4. Estimate endpoint availability as success rate across Monte Carlo trials.
5. Use policy thresholds to gate release risk.

## Immediate HTTP Success Semantics (v0)

- Blocking synchronous edges are part of required request paths.
- Asynchronous/non-blocking edges are excluded from immediate HTTP success checks.
- Endpoint succeeds when all required services in at least one discovered path are alive.

## Reproducibility Controls

- Fixed random `seed`.
- Explicit simulation parameters (`trials`, `failure_probability`).
- Stable JSON outputs suitable for versioned CI artifacts.

