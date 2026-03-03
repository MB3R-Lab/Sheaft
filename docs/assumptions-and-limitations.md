# Assumptions and Limitations

## Core Assumptions (v0)

- Failures are modeled as independent fail-stop crashes.
- Availability impact is approximated using connectivity + replica counts.
- OTel traces reflect relevant call paths for modeled endpoints.
- Policy thresholds represent accepted release risk for a given environment.

## Known Limitations

- No correlated shocks, gray failures, or latency-tail modeling in v0.
- Discovery coverage depends on observed traces (rare paths may be missed).
- Endpoint predicates are minimal and focused on immediate HTTP success.
- Async/event edges are currently treated as low-impact for immediate HTTP SLO estimation.

## Operational Guidance

- Use Sheaft as a broad, cheap filter before expensive live chaos campaigns.
- Treat low-confidence outputs as escalation candidates, not final proof.
- Keep policy thresholds explicit and environment-specific.

