# Assumptions and Limitations

## Assumptions

- Input topology and endpoint metadata are produced upstream.
- Supported contracts are explicitly versioned and whitelisted.
- `1.0.0` semantics are fail-stop and profile-driven.
- `1.1.0` advanced analysis depends on explicit retry, timeout, latency, placement, and shared-resource metadata.
- Weighted aggregates reflect configured workload mix, not observed runtime traffic by default.

## Current Limitations

- No live chaos execution or traffic generation.
- No automatic discovery ownership in production flow.
- Baseline comparison accepts prior reports and raw supported artifacts, but only overlapping metrics are directly comparable.
- Missing advanced metadata is reported as unavailable instead of being synthesized.
- Explicit predicates remain service-based; edge-aware behavior requires journey data or path diagnostics.
- Directory watch mode selects the newest matching file; it does not merge multiple artifacts.

## Guidance

- Use Sheaft as a cheap, repeatable consumer-side posture check.
- Treat low-confidence or degraded posture results as escalation inputs, not proof of runtime safety.
- Keep thresholds and profile definitions environment-specific and explicit.
