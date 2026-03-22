# Sheaft Consumer Semantics v2

This document defines the richer dual-line analysis behavior used when Sheaft consumes Bering `1.1.0` artifacts or when a `1.0.0` artifact is analyzed with an opt-in Sheaft fault contract.

## Version Scope

- `io.mb3r.bering.model@1.1.0`
- `io.mb3r.bering.snapshot@1.1.0`

`1.0.0` remains supported, but it stays the baseline semantics line from [consumer-semantics-v1.md](consumer-semantics-v1.md). Sheaft does not silently reinterpret old `1.0.0` artifacts under richer `1.1.0` semantics.

## Semantics Split

- `1.0.0`: stable fail-stop baseline semantics
- `1.1.0`: additive typed metadata for edge IDs, placements, shared resources, retries, timeouts, and observed latency/error summaries
- `1.0.0` plus Sheaft fault contract: only the honest subset that can be expressed with the artifact plus the external contract is enabled

## Execution Order

For each profile and trial, Sheaft applies:

1. deterministic profile seed resolution
2. baseline service availability sampling from the selected sampling mode
3. correlated service or placement outages plus shared-resource outages
4. edge fail-stop faults
5. edge and service partial degradations
6. endpoint/path evaluation plus assertions and gate logic
7. baseline diffs

## Endpoint Forms

### Predicate-based endpoint

- explicit predicates stay service-based
- edge fail-stop and edge brownout faults do not mutate the success meaning of old explicit predicates
- advanced graph diagnostics can still be emitted alongside the predicate result

### Journey-based endpoint

- legacy fallback and explicit journey overrides resolve to ordered service paths
- each path also carries ordered stable edge IDs
- path execution can depend on:
  - service availability
  - edge availability
  - retry policy
  - timeout viability
  - injected or observed error/latency behavior

## Legacy Discovery Rule

Legacy fallback path discovery is unchanged:

- only `sync` + `blocking=true` edges participate
- `async` edges remain excluded
- `blocking=false` edges remain excluded

## Advanced Metrics

When required metadata exists, Sheaft emits:

- timeout mismatch counts on blocking synchronous paths
- retry amplification factors on paths, endpoints, and edges
- blast-radius counts for correlated failures
- path-level expected success under brownout faults

When required metadata does not exist, the metric is reported as unavailable with a reason. Sheaft does not guess retries, timeouts, circuit-breaker caps, placements, or latency values.

## Placement-Aware Availability

If `1.1.0` placement buckets exist:

- correlated placement faults can kill only the matching buckets
- service liveness for legacy predicate evaluation remains "at least one bucket survives"
- the existing sampling modes compose with placement buckets instead of being replaced

If no placement metadata exists, Sheaft falls back to the coarse service-level model.

## Fault Contract

The Sheaft-owned fault contract adds:

- correlated failure domains by service IDs, labels, placement labels, or shared resources
- edge fail-stop faults
- edge and service partial degradations
- structured assertions on endpoints, paths, edges, and profiles

Assertion failures feed the normal gate result instead of creating a parallel gate subsystem.

## Baselines

The existing `analysis.baselines` flow now accepts either:

- a prior Sheaft report, or
- a raw supported Bering artifact

This allows primary `1.1.0` artifacts to be compared directly against `1.0.0` baseline artifacts in CI. Overlapping metrics produce diffs. Missing advanced metrics remain in the diff as non-comparable entries with explicit reasons.

## Out of Scope

This profile still does not add:

- live chaos execution
- traffic generation
- a new discovery pipeline
- guessed defaults for missing advanced metadata
