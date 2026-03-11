# Methodology

Sheaft estimates resilience posture from externally produced topology artifacts.

## Simulation Model

For each configured profile Sheaft:

1. resolves endpoint success logic from richer predicates when available
2. falls back to legacy path discovery or explicit journeys when richer predicates are absent
3. samples service availability states according to the selected sampling mode
4. estimates endpoint availability over deterministic Monte Carlo trials
5. computes unweighted and weighted aggregates

## Sampling Modes

- `independent_replica`: replicas fail independently and a service stays available while any replica survives
- `independent_service`: each service is sampled once per trial regardless of replica count
- `fixed_k_service_set`: exactly `k` services fail per trial

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

## Determinism

For a fixed artifact, seed, and analysis config:

- profile seeds are derived deterministically
- profile execution order is stable
- report JSON ordering is stable enough for CI artifact diffing
