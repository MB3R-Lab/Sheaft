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

This file captures the repository-side audit performed on 2026-03-11: current GitHub issue state versus what is actually implemented in this repository.

## Audit Summary

| Epic | GitHub state | Repo reality | Notes |
| --- | --- | --- | --- |
| R1 | open | partial | Core semantics are implemented across `internal/artifact`, `internal/analyzer`, `internal/simulation`, and `docs/methodology.md`, but the normative consumer semantics doc and resiliency pattern contract are still missing. |
| R2 | closed | done | Artifact discovery/ingestion, provenance, incomplete telemetry handling, and diff-capable artifacts are present and covered by tests. |
| R3 | open | partial | Reproducible analysis is in place, but pluginization, explicit calibration, and scale benchmarks are still backlog items. |
| R4 | open | partial | External benchmark contract and limitations docs exist, but the public benchmark suite and release-grade quality reports are not yet in-repo. |
| R5 | open | partial | CI gate, service mode, and output artifacts are implemented; chaos triage and cross-CI handoff templates were still the main workflow gap before this audit. |
| R6 | open | partial | Strict contract pinning, conformance checks, and vendored schemas exist; release workflow, compatibility matrix, and multi-version support remain open. |
| R7 | open | gap | No open-core/export playbook material exists yet beyond issue-level planning. |
| R8 | open | gap | Security/privacy work is not yet implemented beyond lightweight assumptions/limitations guidance. |
| R9 | open | partial | Diff endpoints exist, but there is no why/debug UX or dependency-level explanation layer yet. |
| R10 | open | partial | Research references are published in `README.md`, but community comparison/adoption collateral is still thin. |

## Task-Level Audit

### R1. Normative model semantics

| Issue | GitHub state | Repo reality | Notes |
| --- | --- | --- | --- |
| R1.1 | open | partial | Contract handling, predicate/weight precedence, snapshots, and journey overrides are implemented, but there is no dedicated "Consumer Semantics v1" spec with the required examples. |
| R1.2 | open | partial | Fail-stop semantics and three sampling modes are implemented and tested; timeout/partial/gray extension contract is still missing. |
| R1.3 | open | gap | No input contract or end-to-end handling for retry/timeout/circuit-breaker/fallback/rate-limit annotations exists yet. |

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
| R3.3 | open | partial | Predicate/weight provenance is reported, but parameter sourcing/calibration provenance is not explicit end-to-end. |
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
| R5.5 | open | partial | Cross-CI handoff templates and conventions are now in-repo, but they are still sample templates rather than validated example repos. |
| R5.6 | open | done locally | GitLab and Jenkins templates plus documented handoff/exit behavior now exist in-repo; GitHub issue can be closed after sync. |

### R6. Standardization and interoperability

| Issue | GitHub state | Repo reality | Notes |
| --- | --- | --- | --- |
| R6.1 | closed | done | Open schemas live under `api/schema` and are validated via tests. |
| R6.2 | closed | done | Contract and integration tests cover model/snapshot consumption and output shape. |
| R6.3 | closed | done | The repository already functions as the open reference consumer implementation. |
| R6.4 | open | partial | Remote schema sync checks exist in `.github/workflows/schema-contract.yml` and `scripts/ci/check-remote-schema-sync.sh`, but there is no documented cross-repo release policy or matrix tie-in yet. |
| R6.5 | open | gap | No compatibility matrix is published or linked from `README.md` yet. |
| R6.6 | open | gap | Only a single supported schema version is pinned today; project-level multi-version pinning is absent. |

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

1. R5.5: finish the CI/CD handoff path end-to-end by keeping templates current and validating them in example repos or smoke pipelines.
2. R6.5: publish a Bering-Sheaft compatibility matrix and wire it into release/update checks.
3. R6.4: document the contract release policy so schema pin bumps have a defined workflow.
4. R1.1: publish the missing normative consumer semantics document while the implementation surface is still compact.
5. R3.3: add explicit parameter sourcing/calibration provenance to reports and summaries.

## Selected Task

The next task taken into active work from this audit is **R5.5: Bering artifact handoff templates for CI/CD**.

Why this is first:

- it is `priority: p0` and directly blocks production workflow adoption;
- the repo already has the runtime primitives (`run`, strict contract validation, output artifacts, exit codes);
- it also closes the narrower `R5.6` gap as part of the same documentation/templates pass.
