# Assumptions and Limitations

## Core Assumptions (v0)

- Failures are modeled as independent fail-stop crashes.
- Availability impact is approximated using connectivity + replica counts.
- Input model is produced upstream by Bering.
- Input model matches pinned contract metadata exactly.
- Policy thresholds represent accepted release risk for a given environment.

## Known Limitations

- No correlated shocks, gray failures, or latency-tail modeling in v0.
- Discovery quality is inherited from upstream Bering model quality.
- Endpoint predicates are minimal and focused on immediate HTTP success.
- Async/event edges are currently treated as low-impact for immediate HTTP SLO estimation.

## Operational Guidance

- Use Sheaft as a broad, cheap filter before expensive live chaos campaigns.
- Treat low-confidence outputs as escalation candidates, not final proof.
- Keep policy thresholds explicit and environment-specific.
