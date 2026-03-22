# Methodology

Sheaft estimates resilience posture from externally produced topology artifacts.

## Simulation Model

For each configured profile Sheaft:

1. resolves endpoint success logic from richer predicates when available
2. falls back to legacy path discovery or explicit journeys when richer predicates are absent
3. samples service availability states according to the selected sampling mode
4. applies optional fault-profile overlays after baseline sampling:
   - correlated service or placement outages
   - edge fail-stop faults
   - edge or service partial degradations
5. evaluates endpoint success:
   - explicit predicates remain service-availability based
   - journey-based paths can depend on service liveness, edge liveness, retries, timeout viability, and brownout error rates
6. estimates endpoint and path success over deterministic Monte Carlo trials
7. computes unweighted and weighted aggregates plus advanced diagnostics

## Sampling Modes

- `independent_replica`: replicas fail independently and a service stays available while any replica survives
- `independent_service`: each service is sampled once per trial regardless of replica count
- `fixed_k_service_set`: exactly `k` services fail per trial

When `1.1.0` placement buckets exist, `independent_replica` samples those buckets explicitly. A service remains effectively alive while at least one bucket still has a live replica.

## Predicate Semantics

Supported predicate types:

- `all_of`: every operand must succeed
- `any_of`: at least one operand must succeed
- `k_of_n`: at least `k` operands must succeed

Operands can be service IDs or nested predicates.

## Legacy Fallback

If no richer predicate definition is supplied for an endpoint:

- blocking synchronous edges define the immediate success graph
- `any_of` is applied across discovered or overridden paths
- `all_of` is applied within each path

This fallback remains the baseline rule for `1.0.0` semantics and for backward-compatible journey discovery under `1.1.0`.

## Advanced Diagnostics

When the artifact and optional fault contract provide the required metadata, Sheaft also computes:

- timeout mismatch counts on blocking synchronous paths
- retry amplification factors
- blast-radius counts for correlated failures
- path-level expected success under partial degradations

If retry, timeout, latency, placement, or shared-resource metadata is missing, the affected metric is marked unavailable with a reason instead of being guessed.

## Determinism

For a fixed artifact, seed, and analysis config:

- profile seeds are derived deterministically
- profile execution order is stable
- report JSON ordering is stable enough for CI artifact diffing
