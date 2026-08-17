# Security Policy

## Supported versions

| Product version | Supported |
| --- | --- |
| `main` | Yes |
| `v1.2.0` | Yes |
| Older release lines | Best effort only |

The supported v1 path accepts the strict Bering artifact contracts `io.mb3r.bering.model@1.3.0` and `io.mb3r.bering.snapshot@1.3.0`. Unknown, mismatched, and retired preview contract lines are not supported and must be rejected rather than silently downgraded.

## Reporting a vulnerability

Please do **not** open a public GitHub issue for a suspected security vulnerability.

Use GitHub Private Vulnerability Reporting if it is available for this repository. Otherwise, report the issue by email to:

`contact@mb3r-lab.ru`

Please include, when possible:

- the affected Sheaft version or commit;
- the affected command (`simulate`, `gate`, `run`, `serve`) or interface;
- a minimized model/snapshot, policy, analysis configuration, or baseline fixture;
- reproduction steps;
- expected and observed behavior;
- your assessment of impact and attacker prerequisites.

Do not send real production topology, traces, credentials, customer data, or other sensitive material unless absolutely necessary. Prefer a minimized synthetic fixture.

We will review security reports as promptly as practical and coordinate remediation and disclosure with the reporter. Please avoid public disclosure until a fix or mitigation has been coordinated.

## Security-sensitive areas

Sheaft consumes model/snapshot artifacts and configuration and may return a CI/CD release-gate decision. Security-sensitive areas therefore include:

- model and snapshot artifact parsing;
- exact upstream contract/version validation;
- policy, analysis, contract-policy, and baseline parsing;
- simulation resource bounds and numeric edge cases;
- certified-tolerance and regression gate evaluation;
- report generation and `--why` decision explanations;
- baseline selection, fingerprints, and comparison integrity;
- filesystem/output/history handling;
- long-running `serve` artifact-watch path;
- HTTP status/report/diff/history/metrics endpoints;
- Helm, OCI, binary, checksum, and GitHub Actions release paths.

## Security expectations

The following are intended security properties of Sheaft:

1. Unknown or mismatched upstream contracts fail closed; there is no silent schema fallback.
2. Model, snapshot, policy, analysis, and baseline files are data and must never be executed.
3. Invalid, malformed, partially written, or incompatible input must not silently produce a trusted `PASS` verdict.
4. The gate verdict, report, summary, and `--why` explanation must be derived from the same validated analysis state.
5. Baseline comparison must not accept an incompatible or substituted baseline without detection by the existing compatibility/fingerprint checks.
6. A fixed valid input, configuration, and seed must preserve deterministic security-relevant decision behavior.
7. Crafted model size, simulation settings, or analysis parameters must not permit unbounded resource consumption in CI or service mode.
8. Service-mode report/history endpoints must not expose more operational information than the deployment intends.

## What counts as a security issue

Examples include, but are not limited to:

- parser-triggered code execution or arbitrary file access;
- schema/version bypass or downgrade;
- a malformed/incompatible artifact producing `PASS` instead of an error;
- gate bypass through policy/config parsing ambiguity;
- baseline substitution that evades documented compatibility/fingerprint checks;
- algorithmic denial of service through model complexity or simulation parameters;
- unsafe watch/history/output path handling;
- exposure of sensitive model, report, history, or metrics data;
- compromise of build/release artifacts or credentials.

A disagreement about whether the stochastic-connectivity model accurately predicts a real incident is normally a model-validity issue rather than a security vulnerability unless an attacker can exploit the behavior to bypass validation, manipulate a trusted gate, disclose data, or deny service.
