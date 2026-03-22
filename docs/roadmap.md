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

This file captures the repository-side audit refreshed on 2026-03-22: current GitHub issue state versus what is actually implemented in this repository.

## Release Tracking State

- Latest public release shipped: [Sheaft v0.2.1 technical preview](https://github.com/MB3R-Lab/Sheaft/releases/tag/v0.2.1)
- Historical shipped milestones: `v0.1.0 technical preview`, `v0.1.1 technical preview`, `v0.2.0 technical preview`, `v0.2.1 technical preview`
- Active backlog milestone: `Post-v0.2.1 technical preview`
- Previous release-tracking issue: [#80](https://github.com/MB3R-Lab/Sheaft/issues/80)
- Current release-tracking issue: [#81](https://github.com/MB3R-Lab/Sheaft/issues/81)
- GitHub issue and milestone sync was refreshed after the `v0.2.1` patch release so the tracker now reflects the shipped release state.

## Audit Summary

| Epic | GitHub state | Repo reality | Notes |
| --- | --- | --- | --- |
| R1 | open | partial | Baseline semantics and the richer dual-line/fault-contract semantics are now implemented in-repo, but broader policy families such as rate-limit/fallback annotations are still incomplete. |
| R2 | closed | done | Artifact discovery/ingestion, provenance, incomplete telemetry handling, and diff-capable artifacts are present and covered by tests. |
| R3 | open | partial | Reproducible analysis and explicit parameter/calibration provenance are now in place, but pluginization and scale benchmarks are still backlog items. |
| R4 | open | partial | External benchmark contract and limitations docs exist, but the public benchmark suite and release-grade quality reports are not yet in-repo. |
| R5 | open | partial | CI gate, service mode, output artifacts, and validated cross-CI handoff smoke pipelines are implemented; chaos triage remains the main workflow gap. |
| R6 | closed | done | Strict contract pinning, conformance checks, vendored schemas, compatibility matrix, contract release workflow, and real dual-line `1.0.0`/`1.1.0` support are now implemented end to end. |
| R7 | open | gap | No open-core/export playbook material exists yet beyond issue-level planning. |
| R8 | open | gap | Security/privacy work is not yet implemented beyond lightweight assumptions/limitations guidance. |
| R9 | open | partial | Diff endpoints exist, but there is no why/debug UX or dependency-level explanation layer yet. |
| R10 | open | partial | Research references are published in `README.md`, but community comparison/adoption collateral is still thin. |

## Task-Level Audit

### R1. Normative model semantics

| Issue | GitHub state | Repo reality | Notes |
| --- | --- | --- | --- |
| R1.1 | closed | done | "Consumer Semantics v1" is documented in-repo with version scope, precedence rules, and 15 expected behavior examples. |
| R1.2 | closed | done | Fail-stop baseline semantics, three sampling modes, timeout/partial degradation handling, and the external Sheaft fault contract are implemented and tested. |
| R1.3 | open | partial | Retry/timeout/circuit-breaker inputs from Bering `1.1.0` and the Sheaft fault contract are now handled; fallback/rate-limit annotations still remain outside the implemented surface. |

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
| R4.1 | open | partial | `docs/benchmark-external.md` defines the external contract, but the actual benchmark suite is not in this repo. |
| R4.2 | open | gap | No release-quality metrics report or fixed benchmark quality publication exists. |
| R4.3 | open | partial | `docs/assumptions-and-limitations.md` documents some boundaries, but not yet the full "do-not-trust signals" catalogue with detector heuristics. |

### R5. Integration into engineering workflows

| Issue | GitHub state | Repo reality | Notes |
| --- | --- | --- | --- |
| R5.1 | closed | done | `run`, `gate`, exit codes, and CI-oriented batch flow are implemented and documented. |
| R5.2 | closed | done | `serve`/`watch`, status endpoints, history, and metrics cover the observability workflow. |
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
| R6.6 | closed | done | Project-level contract policy now coexists with real dual-line support, end-to-end `1.0.0`/`1.1.0` contract tests, and cross-line artifact baseline comparison coverage. |

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
| R9.1 | open | gap | CLI gate output is concise, but there is no dedicated why mode. |
| R9.2 | open | gap | No debugging toolkit exists for contract/path inspection beyond current tests and errors. |
| R9.3 | open | partial | `current-diff` exposes diffs, but they stop at profile/endpoint deltas and do not explain dependency-level causes. |

### R10. Community adoption

| Issue | GitHub state | Repo reality | Notes |
| --- | --- | --- | --- |
| R10.1 | closed | done | Research artifacts and references are linked from `README.md`. |
| R10.2 | open | gap | No explicit comparison with alternatives has been published yet. |

## Prioritized Backlog After Audit

1. R4.3: expand applicability boundaries into concrete do-not-trust signals and detector heuristics.
2. R1.3: finish the remaining fallback/rate-limit annotation surface or explicitly close it out of scope.
3. R9.1: add a why mode for gate decisions on top of current report/diff output.
4. R4.2: publish release-grade quality metrics for Sheaft-on-Bering.
5. R3.2: decide whether analysis extensibility should become a real plugin surface or stay intentionally in-core.

## Current Execution Note

- Repository-side audit refreshed on 2026-03-22 after dual-line and advanced-analysis implementation landed locally.
- After the `v0.2.0` release is published, the next GitHub sync should close `R6.6`, close `R1.2`, narrow `R1.3`, and open the next release-tracking issue for the following technical-preview tag.
- Added explicit release-tracking traceability on 2026-03-14 via [#78](https://github.com/MB3R-Lab/Sheaft/issues/78), which is closed because the public `v0.1.0` technical preview shipped.
- The next highest-priority repo task is **R4.3: expand applicability boundaries into concrete do-not-trust signals and detector heuristics**.
