# Sheaft v1 Major Semantics

This page is the release-claim boundary for the Sheaft v1 major line.

Sheaft v1 implements the product-baseline stochastic connectivity model for Bering-produced service topologies. It does not claim to solve automatic telemetry calibration, arbitrary non-product probability measures, live chaos execution, or rich temporal workflow models.

The formal model behind this release claim is described in the preprint [Stochastic Connectivity as the Foundation of a Runtime Model for Microservice Availability Analysis](https://www.alphaxiv.org/abs/2607.00740).

## Release Claim

For a supported Bering artifact, Sheaft v1 evaluates endpoint availability over:

- a typed directed topology `G=(V,E,tau)`
- a replication map `R`
- a product probability model `P_Node * P_Edge`
- endpoint success predicates `Phi`
- node live probabilities `theta`
- edge live probabilities `rho`
- explicit immediate predicates and eventual predicates

The implementation is deterministic for a fixed artifact, analysis config, and seed.

## Paper Mapping

| Formal element | Sheaft v1 implementation | Evidence | Boundary |
| --- | --- | --- | --- |
| `G=(V,E,tau)` | Bering services become `V`; Bering dependencies become `E`; edge type `tau` is represented by `kind`, `blocking`, and optional operation-aware `identity`. | `io.mb3r.bering.model@1.3.0`, `io.mb3r.bering.snapshot@1.3.0`, [examples/outputs/model-v1.3.0.sample.json](../examples/outputs/model-v1.3.0.sample.json), [examples/outputs/snapshot-v1.3.0.sample.json](../examples/outputs/snapshot-v1.3.0.sample.json) | Sheaft consumes topology; discovery ownership remains upstream in Bering. |
| `R` | Service `replicas` and `metadata.placements[].replicas` define the replication map. `independent_replica` samples replicas or placement buckets; a service is live while at least one sampled replica bucket survives. | [docs/methodology.md](methodology.md), [configs/analysis.v1.1.example.yaml](../configs/analysis.v1.1.example.yaml) | Placement metadata is used when present; otherwise Sheaft falls back to coarse service replicas. |
| `P` | The v1 baseline is `P_Node * P_Edge`: node/service live events and edge live events are sampled independently before optional reviewed fault overlays. | [docs/oracle-suite.md](oracle-suite.md), `make oracle-suite` | Arbitrary non-product `P` is out of scope. Correlated failures are explicit fault overlays, not inferred probability distributions. |
| `theta` | `analysis.reliability.node_live_probability`, `profiles[].reliability.node_live_probability`, per-service overrides, and Bering reliability evidence resolve node live probabilities. Legacy `failure_probability` maps to `1 - theta` only when explicit node reliability is absent. | [docs/configuration.md](configuration.md), [api/schema/analysis.schema.json](../api/schema/analysis.schema.json) | Sheaft does not do automatic probability calibration from telemetry. Inputs must be configured or provided as upstream evidence. |
| `rho` | `analysis.reliability.edge_live_probability`, `profiles[].reliability.edge_live_probability`, per-edge overrides, and Bering edge reliability evidence resolve edge live probabilities. | [docs/configuration.md](configuration.md), [configs/analysis.v1.1.example.yaml](../configs/analysis.v1.1.example.yaml) | Per-edge overrides must use edge IDs present in the artifact being analyzed. |
| `Phi` | Legacy `all_of`, `any_of`, and `k_of_n` predicates remain service-only predicates. `edge_aware` predicates evaluate reachability from `entry_service` to every mandatory target through live selected edge modes. | [configs/predicate-contract.example.yaml](../configs/predicate-contract.example.yaml), [api/schema/predicate-contract.schema.json](../api/schema/predicate-contract.schema.json) | Existing service-only predicates are not silently reinterpreted as path predicates. Use `edge_aware` or producer endpoint semantics for edge-sensitive claims. |
| immediate predicates | Bering `metadata.semantics.predicate_mode: immediate_response` maps to an edge-aware predicate over blocking synchronous edges only. Async dependencies do not affect immediate response success. | [docs/consumer-semantics-v2.md](consumer-semantics-v2.md), [docs/oracle-suite.md](oracle-suite.md) | Immediate response is not business completion. Use eventual predicates for async completion claims. |
| eventual predicates | Bering `metadata.semantics.predicate_mode: eventual_completion` maps to an edge-aware predicate over declared dependency modes. Async edges participate only when `dependency_modes` includes `async`. | [docs/consumer-semantics-v2.md](consumer-semantics-v2.md), [examples/outputs/snapshot-v1.3.0.sample.json](../examples/outputs/snapshot-v1.3.0.sample.json) | Rich temporal workflow models, ordering constraints, sagas, compensations, and time-window semantics are future work. |

## Compatibility Contract

Sheaft v1 is a strict downstream consumer of Bering artifacts. It accepts only pinned Bering contracts.

Current v1-compatible Bering contract line:

- `io.mb3r.bering.model@1.3.0`
- URI: `https://mb3r-lab.github.io/Bering/schema/model/v1.3.0/model.schema.json`
- digest: `sha256:2aa8a3550a25dc626ba6d2f5833569efca2f382b9e5c9c3405be93695d7d48ae`
- `io.mb3r.bering.snapshot@1.3.0`
- URI: `https://mb3r-lab.github.io/Bering/schema/snapshot/v1.3.0/snapshot.schema.json`
- digest: `sha256:cb778e5b0866d9ce5cfe7f23b8d98a339603593a0247cccd9cddaf05c7ae4bb1`

Pre-v1 Bering preview lines `1.0.0`, `1.1.0`, and `1.2.0` are retired on the current main line. Regenerate those artifacts on Bering `1.3.0` before using them with Sheaft v1. See [docs/compatibility-matrix.md](compatibility-matrix.md) and [compatibility-manifest.json](../compatibility-manifest.json).

## Migration From v0.2.x

Existing v0.2.x users can keep service-only predicates and `failure_probability`, but v1 analysis should move release-critical checks to explicit reliability and endpoint semantics.

Migration steps:

1. Keep `failure_probability` only as a legacy homogeneous node-failure shorthand.
2. Add `analysis.reliability.node_live_probability` for `theta`.
3. Add `analysis.reliability.edge_live_probability` for homogeneous `rho`.
4. Use per-service and per-edge reliability overrides only when the IDs are stable for every artifact under analysis.
5. Keep `all_of`, `any_of`, and `k_of_n` for service-only predicates.
6. Use `edge_aware` for edge-sensitive immediate or eventual predicates.
7. Prefer Bering `1.3.0` endpoint semantic hints when the producer can declare `immediate_response`, `eventual_completion`, or `external_predicate`.
8. Regenerate pre-v1 Bering artifacts on `1.3.0` before using them as release baselines.

## Out Of Scope

These areas are not v1 release claims:

- automatic probability calibration from live telemetry
- arbitrary non-product `P`
- live chaos execution or traffic generation
- rich temporal workflow models
- discovery pipeline ownership inside Sheaft
- proof of runtime safety, failover correctness, RBAC, or multi-tenant isolation

Use [docs/assumptions-and-limitations.md](assumptions-and-limitations.md) when deciding whether a report is strong enough for a blocking release gate.

## Release Evidence

`make release-dry-run` must validate:

- `go test ./...`
- checked-in smoke examples
- the synthetic oracle suite
- the fixed benchmark slice
- schema, example, compatibility-manifest, and v1 docs consistency through `validate-v1-release-docs`
- release assets and manifest validation

The oracle suite is the formal semantic evidence for the product-baseline stochastic connectivity model. The fixed benchmark slice proves the checked-in Bering example remains repeatable as a release fixture.

## Primary Files

- [docs/methodology.md](methodology.md)
- [docs/consumer-semantics-v2.md](consumer-semantics-v2.md)
- [docs/configuration.md](configuration.md)
- [docs/oracle-suite.md](oracle-suite.md)
- [docs/compatibility-matrix.md](compatibility-matrix.md)
- [docs/release-assets.md](release-assets.md)
- [api/schema/model.v1.3.0.schema.json](../api/schema/model.v1.3.0.schema.json)
- [api/schema/snapshot.v1.3.0.schema.json](../api/schema/snapshot.v1.3.0.schema.json)
- [api/schema/predicate-contract.schema.json](../api/schema/predicate-contract.schema.json)
- [api/schema/oracle-report.schema.json](../api/schema/oracle-report.schema.json)
- [configs/analysis.v1.1.example.yaml](../configs/analysis.v1.1.example.yaml)
- [configs/predicate-contract.example.yaml](../configs/predicate-contract.example.yaml)
- [examples/outputs/model-v1.3.0.sample.json](../examples/outputs/model-v1.3.0.sample.json)
- [examples/outputs/snapshot-v1.3.0.sample.json](../examples/outputs/snapshot-v1.3.0.sample.json)
