# Threat Model

## 1. Scope

This threat model covers Sheaft `v1.2.0` and the current `main` line as the downstream resilience posture engine and CI/CD gate in the MB3R toolchain:

```text
Procrustes -> Bering -> model/snapshot artifact
                           |
                           v
                        Sheaft
                  +--------+--------+
                  |                 |
                  v                 v
           resilience report   PASS/WARN/FAIL
                                      |
                                      v
                                   CI/CD
```

Sheaft consumes already-produced model or snapshot artifacts. It does not own production model discovery. The supported public batch surface includes `simulate`, `gate`, and `run`; the repository also ships `serve`/`watch` for evaluation and automation experiments.

The primary security question is:

> Can an attacker controlling an upstream artifact, policy/analysis configuration, baseline, watched file, output path, or release pipeline make Sheaft execute code, consume unreasonable resources, disclose sensitive model data, or return a trusted release verdict that bypasses the intended validation and policy boundary?

## 2. Security-relevant behavior

The current stable v1 line includes:

- strict acceptance of `io.mb3r.bering.model@1.3.0` and `io.mb3r.bering.snapshot@1.3.0`;
- no silent fallback for unknown or mismatched upstream schemas;
- deterministic batch execution for fixed seed and configuration;
- failure-tolerance sweeps and Wilson confidence intervals;
- conservative certified tolerance at a configured confidence level;
- release gates for minimum certified tolerance and maximum regression against compatible baselines;
- deterministic sweep fingerprints and paired raw-artifact baseline evaluation;
- gate decision reasons in `report.json`, `summary.md`, and `--why` output;
- optional advanced analysis and contract-policy narrowing;
- a long-running artifact watch service exposing health, status, current report/diff, history, and metrics endpoints.

Capacity-aware redistribution, queue saturation, retry feedback, overload-cascade prediction, and a stable managed service-mode operations contract are explicitly outside the stable v1 claim.

## 3. Assets

The assets to protect are:

1. **Integrity of the release-gate verdict** — invalid or attacker-controlled input must not silently produce an authoritative `PASS`.
2. **Integrity of analysis results** — certified tolerance, regression comparisons, fingerprints, reasons, and reports must correspond to the validated inputs actually evaluated.
3. **Integrity of baseline comparison** — a baseline must not be silently substituted or interpreted under an incompatible contract.
4. **Availability of CI and service mode** — crafted models/configuration must not cause unbounded simulation or output work.
5. **Confidentiality of model and report data** — topology, endpoint semantics, placements, shared resources, retry/timeout metadata, and historical reports may be sensitive.
6. **Filesystem integrity** — outputs and `.sheaft/history` must not escape their intended paths or overwrite unintended files.
7. **Integrity of distributed binaries, config packs, OCI images, Helm charts, checksums, and GitHub Actions workflows**.

## 4. Trust boundaries

### TB1 — Bering or compatible producer -> Sheaft

Model/snapshot JSON is the primary external input. Even a syntactically valid artifact must be treated as untrusted until contract and semantic validation succeeds.

### TB2 — Policy/analysis configuration -> Sheaft

Gate policy, analysis configuration, contract policy, and simulation parameters directly influence work performed and the final verdict. A malicious or mistaken configuration can be as security-sensitive as a malicious model.

### TB3 — Baseline artifact/config -> comparison engine

Regression gates compare current input against a baseline. Baseline substitution, incompatible contract interpretation, or comparison mismatch can manipulate the final decision.

### TB4 — Sheaft -> CI/CD

Exit status and PASS/WARN/FAIL semantics cross into a delivery system that may promote or block a release. Parse/validation errors must remain distinguishable from `PASS`.

### TB5 — Watched filesystem -> `serve`

Service mode watches an artifact and writes history. The watched file may be replaced while being read, be partially written, be a symlink, or be modified by a lower-trust process.

### TB6 — Sheaft service -> operators/monitoring

`/status`, `/current-report`, `/current-diff`, `/history`, and `/metrics` expose current and historical operational information. Their network exposure is a confidentiality boundary.

### TB7 — Source/build pipeline -> distributed Sheaft artifacts

Go dependencies, GitHub Actions, GoReleaser, default config packs, OCI publishing, Helm packaging, checksums, and release credentials form the software supply-chain boundary.

## 5. Threat actors

Relevant actors include:

- an attacker able to modify or substitute a Bering artifact before CI consumes it;
- a malicious contributor controlling a model/config fixture used by CI;
- a compromised process writing the artifact watched by `serve`;
- an attacker with network access to service-mode report/history endpoints;
- a malicious operator controlling output/history paths or simulation parameters;
- a compromised dependency, GitHub Action, package registry, or release credential;
- accidental pathological input with equivalent impact.

## 6. Threats and abuse cases

### T1 — Upstream contract bypass or schema downgrade

A crafted JSON artifact may claim an accepted contract while using retired/ambiguous semantics, or exploit product-version vs schema-version confusion.

**Desired controls:** exact name/version whitelist, schema digest/URI pinning where already part of compatibility metadata, semantic validation after schema validation, negative tests for all retired preview lines.

### T2 — Fail-open gate behavior

A parse error, validation error, missing field, numeric exception, empty analysis result, or baseline error could accidentally fall through to a default or stale `PASS` state.

**Desired controls:** explicit error state distinct from PASS/WARN/FAIL, fail-closed exit behavior for invalid required input, tests that every validation failure blocks trusted gate emission.

### T3 — Policy/configuration ambiguity

Duplicate keys, numeric edge cases, unknown fields, conflicting rules, or unexpected defaults in YAML/configuration may change gate semantics.

**Desired controls:** strict parsing where appropriate, documented defaults, duplicate/conflict detection, bounded numeric ranges, canonicalized effective configuration in reports/fingerprints.

### T4 — Algorithmic denial of service

A huge graph, extreme replica count, enormous endpoint set, pathological sweep configuration, excessive Monte Carlo samples, or expensive baseline combination may exhaust CI time or service resources.

**Desired controls:** validated upper bounds, context cancellation, execution/time budgets, graph/endpoint/sample limits, complexity benchmarks, resource metrics.

### T5 — Numeric instability or special-value abuse

NaN, infinities, negative/overflowing counts, extreme probabilities, or floating-point boundary behavior could corrupt confidence intervals, tolerance calculations, comparisons, or verdicts.

**Desired controls:** reject non-finite values, validate probability/count domains, test exact boundary conditions, use stable comparison rules, fuzz numeric fields.

### T6 — Baseline substitution or mismatch

An attacker may replace the expected baseline, point analysis at a different valid artifact, or exploit incompatibility so that a regression is hidden.

**Desired controls:** paired-artifact fingerprints, explicit baseline identity in output, compatibility validation, optional external digest pinning in CI, report both evaluated primary and baseline fingerprints.

### T7 — TOCTOU / partial watched artifact

In service mode, a producer may rewrite a watched file non-atomically. Sheaft could read a partial or mixed version, or an attacker could swap a symlink between checks and reads.

**Desired controls:** producer-side atomic replacement, Sheaft-side validate-after-read, stable file identity/fingerprint, debounce/retry only on explicit transient errors, symlink policy.

### T8 — History/output path traversal or overwrite

A crafted path or filename derived from input may cause writes outside `--out-dir` or `.sheaft/history`, overwrite unrelated files, or follow unsafe symlinks.

**Desired controls:** fixed output names under validated roots, canonicalization, safe create/replace behavior, least-privilege runtime user, path/symlink tests.

### T9 — Stale report/gate confusion

A failed new analysis could leave a previous successful report/current status visible, causing an operator or automation to interpret stale state as current.

**Desired controls:** explicit generation IDs/timestamps/fingerprints, error state on failed refresh, never retain stale `PASS` as current without marking staleness.

### T10 — Sensitive-data exposure through service endpoints

Current report, diff, history, or metrics may expose internal topology, SLO thresholds, service names, placements, failure-domain metadata, or operational history.

**Desired controls:** secure deployment guidance, network policy/authentication at ingress if not built in, minimize high-cardinality/sensitive metric labels, configurable retention, least-privilege filesystem permissions.

### T11 — Report/terminal/Markdown injection

Untrusted service names, endpoint names, paths, or metadata copied into `summary.md`, logs, or `--why` output may contain control characters or misleading markup.

**Desired controls:** context-appropriate escaping and sanitization, tests for control characters and markup-sensitive content.

### T12 — Supply-chain compromise

A compromised dependency, GitHub Action, release credential, config pack, OCI image, or Helm chart could alter the executable or default policy surface.

**Desired controls:** least-privilege workflow permissions, pinned third-party Actions, `govulncheck`, dependency review, checksums/provenance, protected release credentials, config-pack integrity checks.

## 7. Gate-specific security invariants

The following invariants are especially important because Sheaft can influence release automation:

1. **Unsupported contracts never produce a trusted verdict.**
2. **Validation errors are not PASS.** A malformed, incompatible, or partially read artifact must terminate in an explicit error state.
3. **No silent fallback.** Retired preview schema semantics cannot be selected implicitly.
4. **Same analysis state, same explanation.** `report.json`, `summary.md`, gate exit behavior, and `--why` must agree on the evaluated verdict and reasons.
5. **Baseline identity is explicit.** Regression decisions must identify the actual baseline artifact/configuration/fingerprint evaluated.
6. **Determinism is preserved for fixed valid inputs.** Fixed artifact, configuration, seed, and executable version should preserve decision behavior.
7. **Stale state is explicit.** A failed service-mode refresh must not leave a prior PASS looking current.
8. **Resource use is bounded.** User-supplied model/configuration cannot request effectively unbounded work without an explicit trusted override.

## 8. Recommended security testing

High-value work includes:

- Go fuzz targets for model/snapshot JSON, policies, analysis configs, contract policies, reports, and baseline parsing;
- negative corpus for unknown, mismatched, and retired Bering contract lines;
- property tests asserting “invalid input never returns PASS”;
- property tests for report/summary/`--why`/exit-code consistency;
- numeric fuzzing for NaN/Inf/extreme probabilities/counts/confidence values;
- model-complexity and sample-count resource benchmarks;
- baseline substitution/mismatch tests and fingerprint assertions;
- watch-loop TOCTOU, partial-write, rename, symlink, and stale-state tests;
- HTTP endpoint information-disclosure review;
- output/history traversal and unsafe-symlink tests;
- `govulncheck`, static analysis, dependency review, and GitHub Actions permission review;
- release archive/config-pack/OCI/Helm checksum and provenance validation.

## 9. Deployment assumptions

The long-running `serve` path is currently shipped for evaluation and automation experiments rather than as the stable v1 managed-operations contract. Deployments should therefore place appropriate authentication, authorization, TLS, network policy, resource limits, filesystem permissions, and retention controls around the service where required.

The stable security claim should remain centered on strict artifact validation and correct fail-closed behavior at the batch/CI boundary.

## 10. Out of scope

The following are not security vulnerabilities by themselves:

- the known modeling boundary around capacity-aware redistribution, queues, retry feedback, or overload cascades;
- ordinary Monte Carlo/statistical uncertainty represented by the product's analysis outputs;
- a scientifically explainable disagreement between a valid model result and a live system;
- topology-discovery correctness inside Bering.

They become security-relevant only if an attacker can exploit them to bypass validation or policy, manipulate a trusted release decision, disclose information, or deny service.
