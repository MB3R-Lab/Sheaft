# Component Criticality

Sheaft analysis schema `1.3` adds opt-in Birnbaum importance for services and edges. Each result answers: how much does the success probability of this endpoint change when this component is guaranteed live versus guaranteed failed, keeping the distribution of all other components fixed?

For independent binary component states and a monotone endpoint success function:

```text
I_B(i) = R(endpoint | component i forced up) - R(endpoint | component i forced down)
       = d R(endpoint) / d q_i
       = P(component i is pivotal)
```

Here `q_i` is the probability that the selected component is live. The derivative identity applies to independent component states; this MVP accepts `independent_service` and `independent_replica` profiles without a `fault_profile`. Fixed-k sampling and correlated failure overlays are rejected rather than labelled classical Birnbaum importance. See the [research definition](https://doi.org/10.1016/j.ress.2024.110306).

## Run

```bash
sheaft run \
  --model examples/outputs/snapshot.sample.json \
  --analysis configs/analysis.importance.example.yaml \
  --out-dir .tmp/importance
```

Configuration:

```yaml
schema_version: "1.3"
seed: 42
trials: 12000
reliability:
  node_live_probability: 0.97
  edge_live_probability: 0.995
profiles:
  - name: steady-state
    sampling_mode: independent_replica
importance:
  - name: checkout-critical-components
    profile: steady-state
    method: birnbaum
    targets:
      - endpoint_id: "gateway:POST /checkout"
    components:
      kinds: [service, edge]
    confidence_level: 0.95
    top_k: 20
gate:
  mode: report
  default_action: report
```

`profile` names the existing baseline profile whose resolved reliability values, replica model, and trial budget are used. The method defaults to `birnbaum`, confidence to `0.95`, and component kinds to both `service` and `edge`. Targets are explicit endpoint IDs. Omitting `top_k`, or setting it to `0`, returns every selected component.

Optional `components.service_ids` and `components.edge_ids` restrict evaluation to exact artifact IDs. Omitting an ID list evaluates all components of that enabled kind. An unknown ID or endpoint is an error. Use the operation-aware edge ID from the Bering artifact; do not reconstruct it from service names.

`top_k` limits report rows after all selected components have been evaluated; it does not reduce computation. Cost grows with trial count, selected components, targets, and paths. Restrict component IDs and endpoints for large artifacts.

## Paired estimator

Each trial samples the baseline world once. For every selected component, Sheaft forces its binary state up and down in that same world and evaluates the requested endpoints. Other services, edges, and execution randomness stay fixed. A pivotal trial is successful with the component up and unsuccessful with it down. The estimate is the pivotal count divided by trials; its interval is a two-sided Wilson interval for that binomial count.

Execution randomness is sampled once per path and trial with a deterministic seed. Observed edge errors, retries, and timeout viability retain the existing endpoint execution semantics. Service-only predicates continue to ignore edges. Immediate-response endpoints ignore async edges unless their declared success semantics include them; eventual-completion endpoints include declared async dependencies. Unrelated components produce exactly zero paired differences.

The estimator rejects an observed non-monotone result (`down` succeeds but `up` fails). It does not interpret negative differences as a pivotal probability.

## Reading the result

`report.json` adds `importance[]`. Each entry records the method, baseline profile, sampling mode, seed, trial count, confidence level, evaluated component count, and per-endpoint rankings. Each component includes:

| Field | Meaning |
| --- | --- |
| `kind`, `id` | Exact service or edge identity |
| `state_semantics` | `service_liveness` or `edge_connectivity` |
| `live_probability` | Baseline probability `q_i` of that binary component state |
| `availability_if_up`, `availability_if_down` | Paired counterfactual endpoint estimates |
| `birnbaum_importance` | Probability that the component changes the endpoint outcome |
| `confidence` | Wilson interval; `successes` counts pivotal trials |
| `forced_down_impact` | Estimated baseline availability lost on total failure: `q_i * I_B(i)` |
| `perfect_component_gain` | Estimated gain from making the component perfect: `(1 - q_i) * I_B(i)` |
| `rank` | Descending point-estimate rank; exact ties share a rank |

The endpoint's `baseline_availability` is sampled on the same worlds as its counterfactuals. Its finite-sample difference from a counterfactual can differ slightly from the probability-weighted `impact` or `gain` estimate. The importance run uses its own deterministic seed; it need not exactly equal the separate normal profile's Monte Carlo estimate.

`summary.md` shows a component table for each target. Overlapping confidence intervals do not establish a reliable rank ordering. Wilson intervals describe Monte Carlo uncertainty conditional on the configured model, not uncertainty in discovered topology or input reliability. They are individual intervals, not a simultaneous confidence guarantee across all ranks.

## Replication and failure eligibility

For a service, the counterfactual toggles aggregate service liveness: at least one live replica versus no live replica. With independent replicas of live probability `p`, `q_service = 1 - (1-p)^r`, using the total replicas in placement buckets when available. Birnbaum service importance is the derivative with respect to `q_service`, not the per-replica probability `p`. It does not measure the benefit of adding one replica or failover capacity.

Services marked `failure_eligible: false` have baseline `q_service = 1`, but an explicitly requested counterfactual still forces them down. Their `perfect_component_gain` is zero; their structural importance and forced-down impact can be high. Importance does not silently omit protected infrastructure from criticality analysis.

For an edge, the binary variable is connectivity, not the observed per-request error rate. Changing retries or error rates changes the endpoint model and requires a new analysis.

## Product use and boundaries

Rankings help identify endpoint bottlenecks, choose chaos experiments, and prioritize candidate reliability improvements. Compare the point estimate with `perfect_component_gain`: a critical but already near-perfect component may offer less attainable gain than another dependency. Business impact, actual failure frequency, change costs, and confidence in input metadata remain separate inputs to prioritization.

Importance is separate from the existing profile and sweep gates. It is not averaged into profile availability or evaluated as a release gate. The MVP does not add automatic importance regression gates, group/zone importance, replica-slot importance, capacity feedback, or correlated-component derivatives. The existing Bering `1.3.0` model/snapshot handoff remains sufficient.
