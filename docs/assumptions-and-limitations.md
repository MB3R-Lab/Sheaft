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

## Planned Do-Not-Trust Signals

The current product backlog treats these as the first detector candidates for R4.3. Until they are implemented as machine-readable report fields, reviewers should check them manually before using a Sheaft gate as a blocking release signal.

- Trace-only artifacts with no explicit resilience metadata for retries, timeouts, circuit breakers, placements, or shared resources.
- Low upstream discovery confidence or missing endpoint/path coverage on services that dominate the configured workload.
- Advanced metrics marked unavailable on the same endpoint or path that drives a gate decision.
- Baseline comparisons where the primary and baseline artifacts do not share overlapping metrics.
- Directory watch inputs where multiple artifacts are expected to be merged; Sheaft currently selects the newest matching file instead.
- Policy thresholds copied across environments without environment-specific failure assumptions.
- Reports generated from stale artifacts whose provenance or generated-at timestamp does not match the release under review.

Backlog outcome: these checks should become explicit report fields with detector ids, severity, affected endpoints or profiles, and suggested remediation.
