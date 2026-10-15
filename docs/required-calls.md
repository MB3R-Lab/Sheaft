# Mandatory calls in a connectivity predicate

An `edge_aware` predicate normally requires reachability of every mandatory target.
When two callers reach the same service, these paths are alternatives. That does
not express an operation which must successfully execute both calls.

External predicate contract version `1.1` adds optional `required_edges`, a list
of exact Bering edge IDs. Every listed edge must succeed in addition to target
reachability. IDs are validated against the model and selected edge modes;
unknown edges or topologically unreachable callers fail before analysis. A caller
that becomes unreachable in a sampled state makes the predicate false. Version `1.0` remains
supported and rejects this new field. Older Sheaft versions reject contract
version `1.1` rather than silently ignoring the condition.

Use an endpoint with `predicate_mode: external_predicate` and a matching
`predicate_ref`, then select the contract with `predicate_contract` in the
analysis configuration. Upstream Bering schema versions and digests are unchanged.

```yaml
schema_version: '1.1'
predicates:
  recommendations:
    type: edge_aware
    entry_service: frontend
    mandatory_targets: [recommendation, product-catalog]
    edge_modes: [sync]
    required_edges:
      # Replace these examples with exact IDs emitted by discovery.
      - 'frontend|recommendation|sync|true'
      - 'recommendation|product-catalog|sync|true'
      - 'frontend|product-catalog|sync|true'
```

For fail-stop analysis an edge has one sampled connectivity state per trial,
shared across every call and path that uses that edge. Repeated RPCs over the same
channel are not independently resampled failures. This contract states mandatory
calls, not their order, payload validity, cache state, or conditional execution.
Those assumptions must be established for the selected operation scenario.

## Runnable release example and migration

Sheaft `v1.3.1` ships `configs/analysis.required-calls.example.yaml` and
`configs/predicate-contract.required-calls.example.yaml` in the default config
pack. Run them against `examples/outputs/model.required-calls.sample.json` as shown in the README.
This generic checkout example uses actual edge IDs in the packaged model; it is
not a reproduction of a paper experiment.

Existing configurations need no migration unless they intend to require calls.
For those endpoints, use predicate contract `1.1`, add the exact required IDs,
and select the contract in `analysis.predicate_contract`. Set the endpoint hint
to `external_predicate`: producer immediate/eventual semantic hints otherwise
take precedence. The packaged example differs from `model.sample.json` only in
this hint for `gateway:GET /explicit`. Keep the original graph intact.
Do not use `required_edges` as a substitute for handler-specific semantic review.

The same conjunctive call groups are used by availability, sweeps, and paired
Birnbaum importance. Unit tests enumerate all eight channel states of a triangle
with two callers to one mandatory service, and compare nontrivial importance
against an independent series-system formula.
