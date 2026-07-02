# Methodology

Sheaft estimates resilience posture from externally produced topology artifacts.

The major-release mapping from the paper notation (`G`, `R`, `P`, `Phi`, `theta`, `rho`) to the implementation is documented in [v1-major-semantics.md](v1-major-semantics.md).

## Simulation Model

For each configured profile Sheaft:

1. resolves endpoint success logic from richer predicates when available
2. falls back to legacy path discovery or explicit journeys when richer predicates are absent
3. samples service availability states according to the selected sampling mode and configured node live probabilities
4. samples path edge availability from configured edge live probabilities when edge-aware artifacts are used
5. applies optional fault-profile overlays after baseline sampling:
   - correlated service or placement outages
   - edge fail-stop faults
   - edge or service partial degradations
6. evaluates endpoint success:
   - legacy explicit predicates remain service-availability based
   - `edge_aware` predicates and journey-based paths can depend on service liveness, edge liveness, retries, timeout viability, and brownout error rates
7. estimates endpoint and path success over deterministic Monte Carlo trials
8. computes unweighted and weighted aggregates plus advanced diagnostics

## Sampling Modes

- `independent_replica`: replicas fail independently and a service stays available while any replica survives
- `independent_service`: each service is sampled once per trial regardless of replica count
- `fixed_k_service_set`: exactly `k` services fail per trial

When `1.3.0` placement buckets exist, `independent_replica` samples those buckets explicitly. A service remains effectively alive while at least one bucket still has a live replica.

## Stochastic Connectivity Parameters

The baseline stochastic connectivity model is controlled through `analysis.reliability` and `profiles[].reliability`:

- `node_live_probability` is the default service live probability `theta`
- `edge_live_probability` is the default edge live probability `rho`
- `services` overrides `theta` for individual service IDs
- `edges` overrides `rho` for individual stable edge IDs

If `reliability.node_live_probability` is omitted, Sheaft preserves legacy behavior by using `1 - failure_probability` as the homogeneous node live probability. If `reliability.edge_live_probability` is omitted, edges are treated as perfectly live unless a fault contract kills or degrades them.

## Predicate Semantics

Supported predicate types:

- `all_of`: every operand must succeed
- `any_of`: at least one operand must succeed
- `k_of_n`: at least `k` operands must succeed
- `edge_aware`: every `mandatory_target` must be reachable from `entry_service` through live edges selected by `edge_modes`

Operands for `all_of`, `any_of`, and `k_of_n` can be service IDs or nested predicates. These legacy predicates remain service-based. `edge_aware` uses path execution semantics: `sync` means blocking synchronous edges, and `async` is included only when listed explicitly.

## Legacy Fallback

If no richer predicate definition is supplied for an endpoint:

- blocking synchronous edges define the immediate success graph
- `any_of` is applied across discovered or overridden paths
- `all_of` is applied within each path

This fallback remains the baseline rule for supported artifacts that do not provide richer endpoint semantics.

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
