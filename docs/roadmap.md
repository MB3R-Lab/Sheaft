# Roadmap and Backlog Audit

GitHub issues remain the source of truth for roadmap tracking:

- Epic index: https://github.com/MB3R-Lab/Sheaft/issues/71
- R1: https://github.com/MB3R-Lab/Sheaft/issues/31
- R2: https://github.com/MB3R-Lab/Sheaft/issues/32
- R3: https://github.com/MB3R-Lab/Sheaft/issues/33
- R4: https://github.com/MB3R-Lab/Sheaft/issues/34
- R5: https://github.com/MB3R-Lab/Sheaft/issues/35
- R6: https://github.com/MB3R-Lab/Sheaft/issues/36
- R7: https://github.com/MB3R-Lab/Sheaft/issues/37
- R8: https://github.com/MB3R-Lab/Sheaft/issues/38
- R9: https://github.com/MB3R-Lab/Sheaft/issues/39
- R10: https://github.com/MB3R-Lab/Sheaft/issues/40

This file captures the repository-side audit refreshed for the `v1.0.0` release line: current GitHub issue state versus what is actually implemented in this repository.

## Release Tracking State

- Current release line: [Sheaft v1.0.0](https://github.com/MB3R-Lab/Sheaft/releases/tag/v1.0.0)
- Previous preview line: [Sheaft v0.2.4](https://github.com/MB3R-Lab/Sheaft/releases/tag/v0.2.4)
- Historical shipped preview milestones: `v0.1.0`, `v0.1.1`, `v0.2.0`, `v0.2.1`, `v0.2.2`, `v0.2.3`, `v0.2.4`
- Active v1 release-tracking issue: [#91](https://github.com/MB3R-Lab/Sheaft/issues/91)
- Previous release-tracking issue: [#82](https://github.com/MB3R-Lab/Sheaft/issues/82)
- Post-v1 product backlog milestone: `Post-v1.0.0 product backlog`
- Post-v1 product backlog index: [#71](https://github.com/MB3R-Lab/Sheaft/issues/71)
- Current upstream Bering app/package sync target: [Bering v1.0.0](https://github.com/MB3R-Lab/Bering/releases/tag/v1.0.0), with accepted `1.3.0` schema contract pins tracked in the compatibility matrix.
- GitHub issue and milestone sync was refreshed for the `v1.0.0` release preparation on 2026-07-02.

## Current V1 Release Checkpoint

- Release URL: https://github.com/MB3R-Lab/Sheaft/releases/tag/v1.0.0
- Release notes: [release/v1.0.0.md](../release/v1.0.0.md)
- V1 semantics: [docs/v1-major-semantics.md](v1-major-semantics.md)
- Bering app/package sync target: `v1.0.0`
- Accepted Bering contract line: `io.mb3r.bering.model@1.3.0` and `io.mb3r.bering.snapshot@1.3.0`
- Compatibility manifest: [compatibility-manifest.json](../compatibility-manifest.json)

## Previous Preview Checkpoint

The published `v0.2.4` release payload is the previous preview baseline:

- Release URL: https://github.com/MB3R-Lab/Sheaft/releases/tag/v0.2.4
- Release manifest asset: `release-manifest.json` (`sha256:e358eb1a06f22e6880f9bb7fc51031fec6d42c2f950abb2fb66249dc6f35bfea`)
- Compatibility manifest asset: `compatibility-manifest.json` (`sha256:d6a7b7413081eaa5771e67647908ac383316f661662ddd80e0a9c2269b7c050e`)
- Default config pack asset: `sheaft-default-config-pack_0.2.4.tar.gz` (`sha256:2c2b7854c1f7310fcc3488b72c82514aebf8bbbe66b217d5a29e394b4757f4d0`)
- Helm chart package asset: `sheaft-0.2.4.tgz` (`sha256:983a47c68df3ce4c7914dc98426c0d43f8af27ffee40d4b420dbb1162ce990f4`)
- Benchmark quality report asset: `quality-report.json` (`sha256:f27c866c38dfae3b28a121944ace47ee6abfdaca965569fcb6bf227dcb0ea14b`)
- OCI image: `ghcr.io/mb3r-lab/sheaft:v0.2.4` (`sha256:a434b5fb3f034a58455b20fa27a6976aa79d66340814f4cd81549b20bd8a5db8`)
- OCI chart: `oci://ghcr.io/mb3r-lab/charts/sheaft:0.2.4` (`sha256:f12887b3a34eede2b3426c0aeea3313b8978c324620475be74ad670f4f24b0a1`)

## Audit Summary

| Epic | GitHub state | Repo reality | Notes |
| --- | --- | --- | --- |
| R1 | open | partial | Baseline semantics and the richer dual-line/fault-contract semantics are now implemented in-repo, but broader policy families such as rate-limit/fallback annotations are still incomplete. |
| R2 | closed | done | Artifact discovery/ingestion, provenance, incomplete telemetry handling, and diff-capable artifacts are present and covered by tests. |
| R3 | open | partial | Reproducible analysis and explicit parameter/calibration provenance are now in place, but pluginization and scale benchmarks are still backlog items. |
| R4 | closed | done | The do-not-trust signal catalogue, fixed benchmark slice, quality report, and external benchmark contract are now in-repo. |
| R5 | open | partial | CI gate, `serve` watch-loop mode, output artifacts, and validated cross-CI handoff smoke pipelines are implemented; chaos triage remains the main workflow gap. |
| R6 | closed | done | Strict contract pinning, conformance checks, vendored schemas, compatibility matrix, and contract release workflow are implemented end to end. The current v1 line intentionally accepts only Bering `1.3.0`. |
| R7 | open | gap | No open-core/export playbook material exists yet beyond issue-level planning. |
| R8 | open | gap | Security/privacy work is not yet implemented beyond lightweight assumptions/limitations guidance. |
| R9 | open | partial | Gate why output now explains threshold/assertion causes, but the broader debugging toolkit and dependency-level explanation layer are still backlog items. |
| R10 | open | partial | Research references are published in `README.md`, but community comparison/adoption collateral is still thin. |

## Task-Level Audit

### R1. Normative model semantics

| Issue | GitHub state | Repo reality | Notes |
| --- | --- | --- | --- |
| R1.1 | closed | done | "Consumer Semantics v1" is documented in-repo with version scope, precedence rules, and 15 expected behavior examples. |
| R1.2 | closed | done | Fail-stop baseline semantics, four sampling modes, timeout/partial degradation handling, and the external Sheaft fault contract are implemented and tested. |
| R1.3 | open | partial | Retry/timeout/circuit-breaker inputs from Bering `1.3.0` and the Sheaft fault contract are now handled; fallback/rate-limit annotations still remain outside the implemented surface. |

### R2. Model discovery from artifacts

| Issue | GitHub state | Repo reality | Notes |
| --- | --- | --- | --- |
| R2.1 | closed | done | Supported artifact ingestion lives in `internal/artifact` and contract validation is strict. |
| R2.2 | closed | done | Provenance and confidence are carried through model metadata and report provenance fields. |
| R2.3 | closed | done | Incomplete telemetry tolerance is covered in the discovery path and tests. |
| R2.4 | closed | done | Current vs previous/baseline diffing is implemented in `internal/report` and exposed in service mode. |

### R3. Analysis/simulation as a product capability

| Issue | GitHub state | Repo reality | Notes |
| --- | --- | --- | --- |
| R3.1 | closed | done | Deterministic Monte Carlo execution and stable config normalization are implemented. |
| R3.2 | open | gap | There is no plugin interface or example plugin; all analyses are wired directly into the core pipeline. |
| R3.3 | closed | done | Reports now include resolved parameter values, source attribution (`default`/`policy`/`override`/`external`), and explicit fallback markers for missing calibration inputs. |
| R3.4 | open | gap | No published large-snapshot workload profile, SLA, or benchmark harness exists. |

### R4. Empirical validation and reproducibility

| Issue | GitHub state | Repo reality | Notes |
| --- | --- | --- | --- |
| R4.1 | closed | done | `benchmarks/fixed-slice/manifest.json`, `make benchmark-slice`, and `docs/benchmark-slice.md` now provide a reproducible public benchmark slice. |
| R4.2 | closed | done | The fixed slice now emits `.tmp/benchmark-slice/quality-report.json` with release-quality checks for repeatability, decision stability, confidence, advanced metric availability, baseline diff coverage, and cross-profile weighted availability. |
| R4.3 | closed | done | `docs/assumptions-and-limitations.md` now publishes a concrete do-not-trust signal catalogue with detector IDs, manual heuristics, severity, and remediation guidance. |

### R5. Integration into engineering workflows

| Issue | GitHub state | Repo reality | Notes |
| --- | --- | --- | --- |
| R5.1 | closed | done | `run`, `gate`, exit codes, and CI-oriented batch flow are implemented and documented. |
| R5.2 | closed | done | `serve`, its artifact watch loop, status endpoints, history, and metrics cover the observability workflow. |
| R5.3 | open | gap | No chaos experiment suggestion engine or triage output exists yet. |
| R5.4 | closed | done | `model.json`, `report.json`, and `summary.md` outputs are generated consistently. |
| R5.5 | closed | done | Example templates are now backed by a template convention checker, a native/docker smoke script, and a GitHub Actions smoke workflow. |
| R5.6 | closed | done | GitLab and Jenkins templates plus documented handoff/exit behavior now exist in-repo and are covered by the shared smoke validation flow. |

### R6. Standardization and interoperability

| Issue | GitHub state | Repo reality | Notes |
| --- | --- | --- | --- |
| R6.1 | closed | done | Open schemas live under `api/schema` and are validated via tests. |
| R6.2 | closed | done | Contract and integration tests cover model/snapshot consumption and output shape. |
| R6.3 | closed | done | The repository already functions as the open reference consumer implementation. |
| R6.4 | closed | done | Contract release workflow, release checklist, and CI verification against published Bering release metadata are now in-repo. |
| R6.5 | closed | done | Compatibility matrix is now published in-repo, linked from `README.md`, and guarded in CI when contract pin files change. |
| R6.6 | closed | done | Project-level contract policy now coexists with strict Bering `1.3.0` support and artifact baseline comparison coverage. |

### R7. Commercialization without lock-in

| Issue | GitHub state | Repo reality | Notes |
| --- | --- | --- | --- |
| R7.1 | open | gap | No explicit open-core boundary documentation exists yet. |
| R7.2 | open | gap | Export/portability guidance has not been documented. |
| R7.3 | open | gap | No pilot-to-production playbook exists yet. |

### R8. Security, privacy, and compliance

| Issue | GitHub state | Repo reality | Notes |
| --- | --- | --- | --- |
| R8.1 | open | gap | Data minimization guidance is not formalized beyond general limitations notes. |
| R8.2 | open | gap | No RBAC or multi-tenant service controls exist. |

### R9. UX and explainability

| Issue | GitHub state | Repo reality | Notes |
| --- | --- | --- | --- |
| R9.1 | closed | done | `policy_evaluation.reasons`, `summary.md` why output, and `sheaft gate/run --why` explain endpoint threshold, aggregate, and assertion causes. |
| R9.2 | open | gap | No debugging toolkit exists for contract/path inspection beyond current tests and errors. |
| R9.3 | open | partial | `current-diff` exposes diffs, but they stop at profile/endpoint deltas and do not explain dependency-level causes. |

### R10. Community adoption

| Issue | GitHub state | Repo reality | Notes |
| --- | --- | --- | --- |
| R10.1 | closed | done | Research artifacts and references are linked from `README.md`. |
| R10.2 | open | gap | No explicit comparison with alternatives has been published yet. |

## Product Capability Milestones

The next roadmap should prioritize trust and operator actionability before broadening the analysis surface. The current CLI and service flows already produce posture outputs; the product gap is proving when those outputs should be trusted, explaining why a decision happened, and turning elevated risk into a useful next action.

### Trustable Gate

Goal: make `sheaft gate` defensible in real delivery pipelines.

- R4.3: concrete do-not-trust signals and detector heuristics for low-quality Bering artifacts are now published.
- R4.1: reproducible public benchmark slice for Sheaft-on-Bering is now published.
- R4.2: release-grade quality metrics on that fixed benchmark slice are now published.
- R3.4: publish a large-snapshot workload profile, runtime SLA, and perf benchmark.

### Explainable Decision

Goal: make every elevated risk traceable from decision to evidence.

- R9.1: why mode for gate decisions is now available in reports, summaries, and CLI output.
- R9.2: add a debugging toolkit for contract, path, and policy inspection.
- R9.3: explain dependency-level causes for posture diffs and serve-mode regressions.

### Actionable Resilience

Goal: connect posture findings to engineering work.

- R1.3: either finish fallback/rate-limit interpretation or explicitly narrow it out of scope.
- R5.3: generate prioritized chaos-triage suggestions from Bering-driven risk outputs.

### Adoption And Enterprise Readiness

Goal: make the Bering -> Sheaft workflow repeatable beyond the first v1 baseline.

- R7.3: publish a pilot-to-production playbook with success metrics.
- R7.1: define the open-core boundary for the consumer product.
- R7.2: document an export/portability bundle.
- R8.1: implement consumer-side data minimization for reports, artifacts, and logs.
- R8.2: define RBAC, multi-tenancy, and audit-log requirements before promoting service mode beyond preview.

## Prioritized Backlog After Audit

1. R9.2: add a debugging toolkit for common contract, path, and policy failures.
2. R9.3: add dependency-level explanations for diffs and serve-mode regressions.
3. R3.4: publish a large-snapshot workload profile, runtime SLA, and perf benchmark.
4. R5.3: add chaos-triage suggestions from Bering-driven risk outputs.
5. R1.3: finish the remaining fallback/rate-limit annotation surface or explicitly close it out of scope.
6. R7.3: publish the pilot-to-production playbook.
7. R3.2: decide whether analysis extensibility should become a real plugin surface or stay intentionally in-core after the trust/explainability backlog is no longer blocking adoption.

## Current Execution Note

- Repository-side audit refreshed on 2026-07-02 for the `v1.0.0` release preparation and the product-capability backlog review.
- GitHub issue [#71](https://github.com/MB3R-Lab/Sheaft/issues/71) should stay aligned with this file as the post-v1 product backlog index, and milestone `Post-v1.0.0 product backlog` should contain non-release-blocking follow-up work.
- Release/package tracking now centers on [#91](https://github.com/MB3R-Lab/Sheaft/issues/91) for the v1 major line. Issue [#82](https://github.com/MB3R-Lab/Sheaft/issues/82) records the shipped `v0.2.4` payload, and [#83](https://github.com/MB3R-Lab/Sheaft/issues/83) is superseded by the v1 tracker.
- R4.3 landed on 2026-05-30 as the first trustable-gate documentation step: `docs/assumptions-and-limitations.md` now contains the do-not-trust signal catalogue and detector field shape.
- R4.1/R4.2 landed on 2026-05-31: `make benchmark-slice` now runs the fixed Sheaft-on-Bering benchmark and emits `quality-report.json`.
- R9.1 landed on 2026-05-31: gate decisions now include machine-readable why reasons and CLI `--why` output.
- Bering app/package sync for the v1 preparation line records Bering `v1.0.0` and strict schema contract support for `1.3.0`.
- The next highest-priority repo task is **R9.2: add a debugging toolkit for common contract, path, and policy failures**.
