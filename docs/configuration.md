# Configuration and Schemas

## Legacy Policy

Legacy batch users can keep using:

- [configs/gate.policy.example.yaml](../configs/gate.policy.example.yaml)
- [api/schema/policy.schema.json](../api/schema/policy.schema.json)

This remains the simplest path for one-profile batch gating.

## Rich Analysis Config

Use the versioned analysis config for advanced batch and service mode:

- [configs/analysis.example.yaml](../configs/analysis.example.yaml)
- [configs/analysis.v1.1.example.yaml](../configs/analysis.v1.1.example.yaml)
- [configs/analysis.sweep.example.yaml](../configs/analysis.sweep.example.yaml)
- [api/schema/analysis.schema.json](../api/schema/analysis.schema.json)

Key sections:

- `profiles`
- `sweeps`
- `reliability`
- `endpoint_weights`
- `baselines`
- `predicate_contract`
- `fault_contract`
- `contract_policy`
- `gate`

`schema_version: "1.0"` remains the simple analysis-config surface. `schema_version: "1.1"` adds fault-profile selection and artifact baselines without changing legacy `1.0` configs. `schema_version: "1.2"` adds failure-tolerance sweeps, confidence-certified boundaries, and explicit boundary gate rules without changing ordinary profile aggregation.

`failure_probability` remains a legacy shorthand for homogeneous node failures. Prefer `reliability` for stochastic connectivity analysis:

- `reliability.node_live_probability`: default node live probability `theta`
- `reliability.edge_live_probability`: default edge live probability `rho`
- `reliability.services`: per-service node live probability overrides
- `reliability.edges`: per-edge live probability overrides keyed by stable edge ID

Profile-level `reliability` values inherit the top-level block and can override individual defaults or service/edge entries. Edge reliability requires path-aware artifact analysis because legacy service predicates do not identify which edge carried a successful call.

For artifact baseline comparisons, prefer homogeneous `reliability.edge_live_probability` unless every compared artifact exposes the same edge IDs. Per-edge `reliability.edges` entries are validated against each artifact under analysis.

Sampling modes:

- `independent_replica`: replicas fail independently; a service stays live while any replica survives
- `independent_service`: each service is sampled once per trial regardless of replica count
- `fixed_k_service_set`: exactly `fixed_k_failures` services fail per trial
- `fixed_k_replica_slots`: exactly `k` replica slots fail per trial; when `fixed_k_failures` is omitted, `k = ceil(failure_probability * total_replica_slots)`

Services with an explicit Bering `metadata.failure_eligible: false` are excluded from baseline sampling in every mode and remain live unless an explicit fault profile targets them. Omitted `failure_eligible` retains the backward-compatible default: the service participates in baseline sampling. Fixed-k counts and derived replica-slot totals include only failure-eligible services.

## Failure-Tolerance Sweeps

Sweeps answer an operation-specific question: at which evaluated fail-stop intensity does endpoint availability first fall below its configured SLO? They are separate from profiles: sweep points are not averaged into `cross_profile_*` and do not affect the existing profile gate.

Each `sweeps[]` entry references a base profile and defines one ordered axis:

- `independent_replica_failure_probability`: values in `[0,1]`; each value directly sets the homogeneous replica failure probability while retaining configured edge reliability and fault overlays
- `failed_replica_slots`: non-negative integer values; output includes both the exact lost slot count and `k / total_replica_slots`

Targets are endpoint IDs with explicit SLOs. Report output contains every evaluated point, the last point meeting SLO, the first violating point, and their crossing bracket. A curve that rises at any evaluated point is marked `non_monotonic` rather than being presented as a trustworthy boundary.

`confidence_level` defaults to `0.95`. Every endpoint point includes a two-sided Wilson interval. `certified_tolerance` is the greatest contiguous evaluated stress level whose lower confidence bound remains at or above the endpoint SLO. Insufficient trials or an interval crossing the SLO is reported as `indeterminate`, never converted to a point-estimate pass.

Boundary rules are explicit and independent of normal profile thresholds:

```yaml
gate:
  mode: fail
  default_action: fail
  boundary_rules:
    - sweep: checkout-independent-replica-failures
      endpoint_id: gateway:POST /checkout
      minimum_certified_tolerance: 0.05
      baseline: last-release
      max_regression: 0.02
```

The minimum must be one of the evaluated axis values. `max_regression` requires a named `analysis.baselines` entry. Raw baseline artifacts are rerun with the same sweep seed and definition; prior reports are comparable only when their sweep fingerprint matches. Missing or incompatible evidence becomes `boundary_indeterminate`. In `mode: fail`, `default_action: fail` makes that fail closed; `indeterminate_action: warn` can weaken an individual rule explicitly.

These boundaries describe the configured fail-stop model. They do not model traffic redistribution, queue saturation, capacity exhaustion, or retry feedback, and must not be described as overload-cascade tipping points.

## Serve Config

Use the versioned serve config for long-running posture mode:

- [configs/sheaft.example.yaml](../configs/sheaft.example.yaml)
- [api/schema/serve.schema.json](../api/schema/serve.schema.json)

## Predicate Overlay Contract

For legacy models that only expose `success_predicate_ref`, supply:

- [configs/predicate-contract.example.yaml](../configs/predicate-contract.example.yaml)
- [api/schema/predicate-contract.schema.json](../api/schema/predicate-contract.schema.json)

The overlay can also carry endpoint weights.

Predicate types are:

- `all_of`, `any_of`, and `k_of_n` for legacy service-availability predicates
- `edge_aware` with `entry_service`, `mandatory_targets`, and `edge_modes` for path-aware immediate or eventual predicates

## Fault Contract

For advanced correlated outages, edge cuts, brownouts, and structured assertions, use the separate Sheaft-owned fault contract:

- [configs/fault-contract.example.yaml](../configs/fault-contract.example.yaml)
- [api/schema/fault-contract.schema.json](../api/schema/fault-contract.schema.json)

The analysis config points at it through `analysis.fault_contract`, and each profile can select a named contract profile through `profiles[].fault_profile`.

## Contract Policy

Use project-level contract pinning and deprecation controls when a deployment wants to accept only a subset of the globally supported Bering contracts:

- [configs/contract-policy.example.yaml](../configs/contract-policy.example.yaml)
- [configs/contract-policy.deprecated.example.yaml](../configs/contract-policy.deprecated.example.yaml)
- [api/schema/contract-policy.schema.json](../api/schema/contract-policy.schema.json)

The same structure can be embedded inline under `analysis.contract_policy`, or passed separately at runtime with `--contract-policy`.

## Artifact Schemas

- Versioned Bering `1.3.0` model schema mirror: [api/schema/model.v1.3.0.schema.json](../api/schema/model.v1.3.0.schema.json)
- Versioned Bering `1.3.0` snapshot schema mirror: [api/schema/snapshot.v1.3.0.schema.json](../api/schema/snapshot.v1.3.0.schema.json)
- Report schema: [api/schema/report.schema.json](../api/schema/report.schema.json)
- Oracle suite report schema: [api/schema/oracle-report.schema.json](../api/schema/oracle-report.schema.json)

Report output now carries both:

- `provenance`: artifact/overlay origin for predicates and weights
- `parameters`: resolved simulation inputs plus source attribution (`default`, `policy`, `override`, `external`) and calibration fallback markers
- `contract_policy`: whether the accepted contract is current or deprecated for this project, plus the effective action (`allow`, `warn`, `fail`)

## Migration Rule of Thumb

- keep `--policy` when one profile and simple thresholds are enough
- move to `--analysis` when you need profiles, weights, baselines, overlays, or contract pinning
- move to `schema_version: "1.1"` when you need `fault_contract`, `profiles[].fault_profile`, or artifact-vs-artifact baseline comparison
- move to `schema_version: "1.2"` when you need failure-tolerance curves and endpoint SLO crossing brackets
- use `serve` when posture must stay current as new artifacts arrive
