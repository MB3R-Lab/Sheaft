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
- [api/schema/analysis.schema.json](../api/schema/analysis.schema.json)

Key sections:

- `profiles`
- `reliability`
- `endpoint_weights`
- `baselines`
- `predicate_contract`
- `fault_contract`
- `contract_policy`
- `gate`

`schema_version: "1.0"` remains the baseline surface. `schema_version: "1.1"` adds fault-profile selection and cross-line artifact baselines without changing legacy `1.0` configs.

`failure_probability` remains a legacy shorthand for homogeneous node failures. Prefer `reliability` for stochastic connectivity analysis:

- `reliability.node_live_probability`: default node live probability `theta`
- `reliability.edge_live_probability`: default edge live probability `rho`
- `reliability.services`: per-service node live probability overrides
- `reliability.edges`: per-edge live probability overrides keyed by stable edge ID

Profile-level `reliability` values inherit the top-level block and can override individual defaults or service/edge entries. Edge reliability requires path-aware artifact analysis because legacy service predicates do not identify which edge carried a successful call.

For cross-line baseline comparisons, prefer homogeneous `reliability.edge_live_probability` unless every compared artifact exposes the same edge IDs. Per-edge `reliability.edges` entries are validated against each artifact under analysis, so a Bering `1.1.0` operation-aware edge ID will not match a sparse `1.0.0` baseline artifact.

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

- Plain model schema: [api/schema/model.schema.json](../api/schema/model.schema.json)
- Snapshot envelope schema: [api/schema/snapshot.schema.json](../api/schema/snapshot.schema.json)
- Versioned Bering `1.1.0` model schema mirror: [api/schema/model.v1.1.0.schema.json](../api/schema/model.v1.1.0.schema.json)
- Versioned Bering `1.1.0` snapshot schema mirror: [api/schema/snapshot.v1.1.0.schema.json](../api/schema/snapshot.v1.1.0.schema.json)
- Report schema: [api/schema/report.schema.json](../api/schema/report.schema.json)
- Oracle suite report schema: [api/schema/oracle-report.schema.json](../api/schema/oracle-report.schema.json)

Report output now carries both:

- `provenance`: artifact/overlay origin for predicates and weights
- `parameters`: resolved simulation inputs plus source attribution (`default`, `policy`, `override`, `external`) and calibration fallback markers
- `contract_policy`: whether the accepted contract is current or deprecated for this project, plus the effective action (`allow`, `warn`, `fail`)

## Migration Rule of Thumb

- keep `--policy` when one profile and simple thresholds are enough
- move to `--analysis` when you need profiles, weights, baselines, overlays, or contract pinning
- move to `schema_version: "1.1"` when you need `fault_contract`, `profiles[].fault_profile`, or artifact-vs-artifact baseline comparison across Bering contract lines
- use `serve` when posture must stay current as new artifacts arrive
