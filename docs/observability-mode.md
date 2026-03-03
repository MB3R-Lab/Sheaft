# Observability Mode (MVP Notes)

MVP is CI-gate-first, but the same pipeline can be run as a lightweight observability capability:

- ingest updated topology traces periodically;
- re-run model discovery and simulation with fixed policy;
- emit posture deltas (`risk_score`, endpoint-level trend changes);
- alert when confidence/risk crosses configured bounds.

The current repository ships only minimal hooks (`internal/observability`) and does not yet include a long-running service.

