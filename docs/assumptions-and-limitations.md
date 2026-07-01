# Assumptions and Limitations

## Assumptions

- Input topology and endpoint metadata are produced upstream.
- Supported contracts are explicitly versioned and whitelisted.
- `1.0.0` semantics are fail-stop and profile-driven.
- `1.2.0` advanced analysis depends on explicit retry, timeout, latency, placement, and shared-resource metadata.
- Weighted aggregates reflect configured workload mix, not observed runtime traffic by default.

## Current Limitations

- No live chaos execution or traffic generation.
- No automatic discovery ownership in production flow.
- No automatic probability calibration from telemetry.
- No arbitrary non-product `P`; the v1 stochastic baseline is `P_Node * P_Edge` plus explicit reviewed fault overlays.
- No rich temporal workflow models for sagas, compensations, ordering constraints, or time-window completion.
- Baseline comparison accepts prior reports and raw supported artifacts, but only overlapping metrics are directly comparable.
- Missing advanced metadata is reported as unavailable instead of being synthesized.
- Legacy explicit predicates remain service-based. Baseline edge reliability requires journey data, path diagnostics, or `edge_aware` predicates.
- Directory watch mode selects the newest matching file; it does not merge multiple artifacts.

## Guidance

- Use Sheaft as a cheap, repeatable consumer-side posture check.
- Treat low-confidence or degraded posture results as escalation inputs, not proof of runtime safety.
- Keep thresholds and profile definitions environment-specific and explicit.

## Do-Not-Trust Signal Catalogue

These signals define when a Sheaft result should not be used as a blocking release gate without human review. They are intentionally concrete enough for release reviewers to check manually today and structured enough to become machine-readable report fields later.

Severity means:

- `block`: do not use the report as a release gate until the input or policy is fixed.
- `review`: keep the report, but require an explicit reviewer note before treating it as release evidence.
- `warn`: useful context that should not block by itself.

| ID | Signal | Manual detector heuristic | Severity | Recommended action |
| --- | --- | --- | --- | --- |
| `DNT-SCHEMA-001` | Artifact contract is unsupported, deprecated by project policy, or not pinned to the expected Bering schema line. | Sheaft rejects unsupported contracts at load time; for accepted contracts, compare `schema_name`, `schema_version`, and the project `contract-policy` decision against the release under review. | `block` | Regenerate the Bering artifact on a supported contract or narrow the project contract policy explicitly. Do not override by editing the artifact. |
| `DNT-META-001` | Upstream discovery confidence is too low for release gating. | Treat `metadata.confidence < 0.80` as review-required by default. For snapshots, also review low `service_support_min`, `edge_support_min`, or `endpoint_support_min` values when present. | `review` | Re-run upstream discovery with better telemetry coverage or document why the missing support does not affect the release path. |
| `DNT-COV-001` | High-weight endpoint or path has weak coverage. | Inspect configured endpoint weights. Any endpoint with weight `>= 0.10`, or any endpoint in the top 20% of configured workload weight, should have an explicit predicate, journey, or path diagnostic source. | `review` | Add explicit journeys or predicate overlays for the release-critical endpoints before using the gate as a blocker. |
| `DNT-ADV-001` | A gate decision depends on an endpoint or profile whose advanced metric is unavailable. | In `report.json`, look for advanced metrics with `available: false`, `provenance: "unavailable"`, or non-comparable diff entries on the same profile or endpoint that triggered the gate. | `review` | Add retry, timeout, latency, placement, shared-resource, or fault-contract metadata, or narrow the gate to metrics that are actually available. |
| `DNT-BASE-001` | Baseline comparison has no meaningful overlap. | If the diff marks the primary metrics as missing, non-comparable, or compares only unrelated advanced metrics, the baseline is not release evidence. This is common when comparing a rich `1.2.0` artifact to a sparse `1.0.0` baseline without shared metrics. | `review` | Pick a baseline artifact with overlapping metrics or publish the comparison as informational only. |
| `DNT-WATCH-001` | Serve/watch mode is pointed at a directory that operators expect to merge. | Directory watch mode selects the newest matching artifact. If the handoff produces multiple partial files that should be combined, the served report reflects only one file. | `block` | Publish a single merged Bering artifact before Sheaft consumes it, or point Sheaft at the exact generated artifact file. |
| `DNT-POLICY-001` | Gate policy is copied across environments without environment-specific assumptions. | Compare policy thresholds, profile weights, sampling mode, and fault assumptions against checked-in examples. Defaults or copy-pasted staging thresholds used for production release gates require review. | `review` | Create environment-specific policy files and record the release assumptions in the PR or release issue. |
| `DNT-STALE-001` | Artifact provenance does not match the release candidate. | Compare artifact provenance, generated-at/emitted-at timestamps, source refs, and the release issue or CI build ID. Treat artifacts older than the release candidate build, or from a different source ref, as stale. | `block` | Regenerate the artifact from the release candidate pipeline and attach the matching report to the release evidence. |
| `DNT-FAULT-001` | External fault contract dominates the result but is not tied to reviewed evidence. | If the report depends on `external_contract` or `artifact+external_contract` provenance for degraded success, timeout viability, or blast radius, verify that the fault-contract file is versioned and reviewed for the release. | `review` | Pin the fault contract beside the policy and include its source in the release-tracking issue. |
| `DNT-SCOPE-001` | The result is being used to claim runtime safety outside Sheaft's model scope. | Any claim involving live chaos execution, traffic generation, runtime failover validation, RBAC, or multi-tenant service isolation is outside current Sheaft output scope. | `warn` | Phrase the release evidence as consumer-side posture analysis, not proof of runtime safety. Use separate runtime tests for those claims. |

## Detector Field Shape

When these checks become machine-readable report output, each detector should emit:

- `id`: one of the stable IDs above.
- `severity`: `block`, `review`, or `warn`.
- `scope`: affected artifact, profile, endpoint, path, or policy file.
- `evidence`: the concrete field or value that triggered the detector.
- `remediation`: one actionable fix.

Until then, release reviewers should record any triggered `block` or `review` signals in the release-tracking issue before treating a Sheaft report as release evidence.
