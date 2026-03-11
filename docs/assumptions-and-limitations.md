# Assumptions and Limitations

## Assumptions

- Input topology and endpoint metadata are produced upstream.
- Supported contracts are explicitly versioned and whitelisted.
- Failure modeling is fail-stop and profile-driven.
- Weighted aggregates reflect configured workload mix, not observed runtime traffic by default.

## Current Limitations

- No correlated shock or latency distribution modeling.
- No automatic discovery ownership in production flow.
- Baseline comparison currently expects report artifacts as the baseline data source.
- Directory watch mode selects the newest matching file; it does not merge multiple artifacts.

## Guidance

- Use Sheaft as a cheap, repeatable consumer-side posture check.
- Treat low-confidence or degraded posture results as escalation inputs, not proof of runtime safety.
- Keep thresholds and profile definitions environment-specific and explicit.
