# Sheaft Consumer Semantics v1

This document defines how Sheaft v1 consumes supported upstream artifact contracts and overlays during analysis and gate evaluation.

## Version Scope

This semantics profile is bound to the currently supported contracts:

- `io.mb3r.bering.model@1.0.0`
- `io.mb3r.bering.snapshot@1.0.0`

Contract pins, URIs, and digests are tracked in [compatibility-matrix.md](compatibility-matrix.md).

## Consumer Pipeline

For a single batch or service recompute, Sheaft applies the following order:

1. load the input artifact and identify whether it is a plain model or snapshot envelope;
2. validate the declared contract strictly against the pinned whitelist;
3. normalize artifact metadata, predicates, weights, and provenance;
4. load optional external overlays and legacy journey overrides from analysis config;
5. resolve endpoint success semantics;
6. run deterministic profile-based simulation;
7. evaluate gate thresholds and emit report, summary, and diffs.

## Artifact Identification

### Plain model artifact

Sheaft treats an input as a plain model artifact when:

- the JSON root has `metadata.schema`;
- that schema matches the pinned `io.mb3r.bering.model@1.0.0` contract exactly.

### Snapshot envelope

Sheaft treats an input as a snapshot envelope when:

- the JSON root has `metadata.schema`;
- that schema matches the pinned `io.mb3r.bering.snapshot@1.0.0` contract exactly;
- the envelope carries snapshot fields such as `snapshot_id`, `ingest`, `coverage`, `diff`, `discovery`, and `model`.

## Strict Contract Rules

The following are hard errors:

- unknown contract name or version;
- mismatched contract URI for a known name/version;
- mismatched contract digest for a known name/version;
- missing required schema metadata fields.

There is no silent fallback to "best effort" parsing for unsupported contracts.

## Core Interpretation Rules

### Services

- `services[].id` is the identity used by predicates, journeys, and reports.
- `services[].name` is descriptive only; simulation logic keys on `id`.
- negative `replicas` is invalid.
- `replicas=0` is accepted by validation but normalized to `1` effective replica during simulation.

### Edges

- `edges[].kind=sync` and `blocking=true` participates in legacy journey discovery.
- `edges[].kind=async` is ignored by legacy journey discovery.
- `blocking=false` is ignored by legacy journey discovery even if the edge is synchronous.
- edge ordering does not carry meaning; Sheaft sorts paths deterministically before simulation.

### Endpoints

- `endpoints[].id` is the stable identity used in reports, thresholds, weights, and diffs.
- `entry_service` is the root of legacy journey discovery.
- `success_predicate_ref` is the contract-level pointer to a richer predicate definition when one is available.
- if no richer predicate resolves, Sheaft falls back to journeys override or path discovery.

### Metadata

- `metadata.schema` on both plain models and snapshot envelopes is the strict contract selector.
- `metadata.confidence` is carried into the report summary as-is.
- `metadata.source_type` and `metadata.source_ref` are carried into `report.input_artifact`.
- `topology_version` is propagated when available.
- for snapshots, top-level `snapshot_id`, top-level `topology_version`, and snapshot `metadata.{emitted_at,source_type,source_ref}` take precedence over nested model metadata when present.

## Predicate Semantics

Supported predicate node types:

- `all_of`: every referenced service or child predicate must succeed.
- `any_of`: at least one referenced service or child predicate must succeed.
- `k_of_n`: at least `k` referenced service or child predicate operands must succeed.

Predicate evaluation is purely service-availability based in v1. There is no latency, timeout, partial-success, or gray-failure semantics in this profile.

## Legacy Fallback Semantics

If an endpoint does not resolve to a richer predicate:

1. Sheaft looks for a journey override for that endpoint.
2. If no override exists, Sheaft discovers all acyclic paths starting from `entry_service` over `sync + blocking` edges only.
3. Each path becomes an `all_of` requirement across services in that path.
4. Multiple paths are combined under `any_of`.

This means:

- all services on one path must be alive;
- any surviving path is sufficient for endpoint success.

## Precedence Rules

### Predicate precedence

Highest to lowest precedence:

1. external predicate overlay from `analysis.predicate_contract`
2. model-embedded `predicates`
3. legacy journey override from `analysis.journeys`
4. legacy path discovery from `entry_service` + `sync/blocking` edges

Notes:

- external overlay merges on top of the artifact-level predicate map and wins on key collision;
- the current upstream snapshot contract does not carry a top-level predicate map;
- the v1 upstream JSON schema does not require embedded predicate maps, but Sheaft supports them as compatible producer/adapter behavior.

### Weight precedence

Highest to lowest precedence:

1. per-profile `profiles[].endpoint_weights`
2. analysis-level `endpoint_weights`
3. external overlay `predicate_contract.endpoint_weights`
4. snapshot `discovery.endpoints[].metadata.weight`
5. model-embedded `endpoint_weights`
6. equal-weight average across endpoints

Notes:

- snapshot discovery weights replace model-embedded weights when present;
- if all resulting weights are missing or non-positive, Sheaft falls back to an unweighted arithmetic mean.

### Gate threshold precedence

Endpoint threshold precedence:

1. `gate.profile_endpoint_thresholds[profile][endpoint]`
2. `gate.endpoint_thresholds[endpoint]`
3. `gate.global_threshold`

Aggregate threshold precedence:

1. `gate.profile_aggregate_thresholds[profile]`
2. `gate.aggregate_threshold`

Cross-profile aggregate threshold:

- `gate.cross_profile_aggregate_threshold` is evaluated separately against the mean weighted aggregate across all profiles.

## Determinism Rules

- a fixed artifact, config, and seed produce stable results;
- profile seeds are derived deterministically from base seed, profile name, and profile index when not set explicitly;
- endpoint IDs, service IDs, and discovered journeys are sorted before evaluation.

## Report Provenance Rules

`report.provenance.predicate_source` and `report.provenance.weights_source` use these values:

- `model`
- `snapshot`
- `external_overlay`
- `default`

`default` means no artifact-level predicate or weight data was present for that dimension.

With the current `io.mb3r.bering.snapshot@1.0.0` envelope, `snapshot` provenance is expected for weights sourced from `discovery.endpoints[].metadata.weight`, not for top-level snapshot predicates.

## Examples

The examples below are normative expected behaviors for v1.

### 1. Plain model with exact model contract is accepted

Input:

- [examples/outputs/model.sample.json](../examples/outputs/model.sample.json)

Expected outcome:

- artifact kind resolves to `model`;
- contract validation succeeds only if `name`, `version`, `uri`, and `digest` all match the pinned model contract.

### 2. Plain model with unsupported contract version is rejected

Input:

- same structure as the model contract, but `metadata.schema.version=9.9.9`

Expected outcome:

- load fails before simulation;
- error names the unsupported contract and lists supported contracts.

### 3. Snapshot envelope uses top-level snapshot contract

Input:

- [examples/outputs/snapshot.sample.json](../examples/outputs/snapshot.sample.json)

Expected outcome:

- artifact kind resolves to `snapshot`;
- report input artifact carries `contract_name=io.mb3r.bering.snapshot` and `contract_version=1.0.0`.

### 4. Snapshot metadata overrides nested model metadata where present

Input:

- snapshot with top-level `snapshot_id`, `topology_version`, and snapshot `metadata.{emitted_at,source_ref,source_type}`

Expected outcome:

- report input artifact uses those top-level snapshot fields rather than nested model metadata when both exist.

### 5. External predicate overlay overrides model predicates or legacy fallback

Input:

- snapshot sample plus [configs/predicate-contract.example.yaml](../configs/predicate-contract.example.yaml)

Expected outcome:

- external overlay wins on predicate key collision;
- report predicate provenance becomes `external_overlay`.

### 6. Snapshot discovery weights override model-embedded weights when config does not override them

Input:

- snapshot sample with `discovery.endpoints[].metadata.weight`;
- model without conflicting endpoint weights;
- no config-level weight override.

Expected outcome:

- Sheaft uses the snapshot discovery weights;
- report weight provenance becomes `snapshot`.

### 7. Weight precedence is profile > analysis > external overlay > snapshot discovery > model

Input:

- snapshot sample with discovery endpoint weights;
- analysis config with top-level `endpoint_weights`;
- profile-specific `endpoint_weights` inside one profile.

Expected outcome:

- the profile-specific map is used for that profile;
- profiles without their own override inherit the analysis-level map;
- artifact-level weights are only used if config-level weights do not override them.

### 8. No weights means unweighted average

Input:

- model or snapshot with no weights anywhere;
- no config-level weights.

Expected outcome:

- weighted aggregate equals the arithmetic mean of endpoint availability values.

### 9. Legacy journey discovery ignores async and non-blocking edges

Input:

- endpoint without a richer predicate;
- graph contains both blocking sync edges and async/non-blocking edges.

Expected outcome:

- only blocking sync edges contribute to discovered journeys;
- async or non-blocking edges have no effect on fallback endpoint success logic.

### 10. Multiple discovered paths mean `any_of(paths)` and `all_of(services in path)`

Input:

- `frontend -> checkoutA` and `frontend -> checkoutB`, both blocking sync;
- failure probability `0.5`;
- no richer predicate, no journey override.

Expected outcome:

- endpoint success means `frontend AND (checkoutA OR checkoutB)`;
- estimated availability is approximately `0.5 * 0.75 = 0.375`.

### 11. Manual journey override replaces discovered paths for that endpoint

Input:

- same topology as example 10;
- journey override restricts the endpoint to `[frontend, checkoutA]`.

Expected outcome:

- endpoint success becomes `frontend AND checkoutA`;
- estimated availability is approximately `0.25`.

### 12. `independent_replica` differs from `independent_service`

Input:

- one service with `replicas=2`;
- failure probability `0.5`.

Expected outcome:

- `independent_replica` gives availability near `0.75`;
- `independent_service` gives availability near `0.5`.

### 13. `fixed_k_service_set` fails paths that require more services than the blast radius allows

Input:

- one linear path `frontend -> checkout -> payment`;
- `fixed_k_failures=1`.

Expected outcome:

- exactly one service fails each trial;
- a single-path endpoint that depends on all three services has availability `0`.

### 14. Gate threshold precedence is profile-specific first

Input:

- profile endpoint threshold and global threshold both exist for the same endpoint.

Expected outcome:

- the profile-specific endpoint threshold is applied for that profile;
- otherwise endpoint-specific threshold wins over global threshold.

### 15. `evaluation_rule=all_profiles` differs from `evaluation_rule=any_profile`

Input:

- two profiles, one passing and one failing;
- `mode=fail`.

Expected outcome:

- `all_profiles`: overall decision is `fail`;
- `any_profile`: overall decision is `pass`.

## Non-Goals of v1

This profile does not define:

- timeout, latency, gray failure, or partial-success semantics;
- resiliency pattern annotations such as retry/backoff or circuit breaker behavior;
- multi-version schema negotiation;
- topology discovery semantics owned by Bering.
