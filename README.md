# Sheaft

[![release](https://img.shields.io/github/v/release/MB3R-Lab/Sheaft?label=release)](https://github.com/MB3R-Lab/Sheaft/releases)
[![checks](https://img.shields.io/github/actions/workflow/status/MB3R-Lab/Sheaft/release-dry-run.yml?branch=main&label=checks)](https://github.com/MB3R-Lab/Sheaft/actions/workflows/release-dry-run.yml)
[![bering schema](https://img.shields.io/badge/bering%20schema-1.3.0-blue)](https://github.com/MB3R-Lab/Sheaft/blob/main/docs/compatibility-matrix.md)

## Related MB3R repositories

Sheaft is downstream of [Bering](https://github.com/MB3R-Lab/Bering), which owns discovery and publishes the model and snapshot artifacts Sheaft consumes. [mb3r-stack](https://github.com/MB3R-Lab/mb3r-stack) bundles compatible Bering and Sheaft releases for integration and deployment.

Sheaft is a downstream resilience posture engine and CI/CD gate for model artifacts produced by Bering or another compatible upstream producer.

## What is Sheaft

Sheaft consumes already-produced model or snapshot artifacts, runs deterministic resilience analysis, emits posture reports, and can fail or warn a delivery pipeline based on policy.

It stays downstream of topology discovery. The public surface in this repository is the CLI and release assets around:

- batch commands: `simulate`, `gate`, `run`
- service command: `serve`, which runs the artifact watch loop; `watch` is kept as a compatibility alias for the same service path

## Stability / Release Status

The current public release line is `v1.0.0`. It is the first major Sheaft line for the stochastic-connectivity baseline over Bering `1.3.0` model and snapshot contracts.

Stable in `v1.0.0`:

- strict acceptance of the Bering v1 contract line: `io.mb3r.bering.model@1.3.0` and `io.mb3r.bering.snapshot@1.3.0`
- batch CLI command names and core flow: `simulate`, `gate`, `run`
- deterministic batch execution for a fixed seed and config
- baseline comparison through `analysis.baselines` for supported `1.3.0` artifacts
- additive advanced analysis when `1.3.0` metadata exists
- gate decision reasons in `report.json`, `summary.md`, and `sheaft gate/run --why`
- fixed benchmark slice and release-quality `quality-report.json` generation
- release archives for Linux and macOS on `amd64` and `arm64`

Outside the stable `v1.0.0` release claim:

- long-running `serve` posture service as a long-term operations contract
- richer analysis configuration ergonomics beyond the shipped examples
- Helm chart and OCI image behavior as a managed operations platform
- local `discover` helper, which is not the production discovery path

## Supported upstream contracts

Sheaft validates artifacts against an explicit whitelist.

These are alternative accepted upstream contract lines for incoming artifacts, not simultaneous version dependencies for a single artifact.

- `io.mb3r.bering.model@1.3.0`
- `io.mb3r.bering.snapshot@1.3.0`

Pinned URIs, digests, and release-line support are tracked in [docs/compatibility-matrix.md](docs/compatibility-matrix.md). The machine-readable compatibility contract is [compatibility-manifest.json](compatibility-manifest.json).

Unknown or mismatched contracts are rejected. There is no silent fallback for unsupported upstream schemas.

`1.3.0` is the strict v1 line for positive replica counts, reliability evidence, endpoint semantic hints, timeout, retry, placement, shared-resource, and edge-scoped analysis when the artifact provides it. Pre-v1 preview contract lines `1.0.0`, `1.1.0`, and `1.2.0` are retired on the current main line and must be regenerated before use with Sheaft v1.

The current `main` line has also been app-level synced against the Bering `v1.0.0` release/package. That sync does not change the accepted Bering schema contract pins.

The v1 major release claim is documented in [docs/v1-major-semantics.md](docs/v1-major-semantics.md): product-baseline `P_Node * P_Edge` stochastic connectivity over Bering topology `G`, replication map `R`, and endpoint predicates `Phi`, with explicit boundaries for what remains future work.

## Research and Evidence

- Formal model: [Stochastic Connectivity as the Foundation of a Runtime Model for Microservice Availability Analysis](https://www.alphaxiv.org/abs/2607.00740)
- DeathStarBench empirical anchor: [Model Discovery and Graph Simulation: A Lightweight Gateway to Chaos Engineering](https://www.alphaxiv.org/abs/2506.11176)
- OpenTelemetry Demo async-semantics case study: [Evaluating Asynchronous Semantics in Trace-Discovered Resilience Models: A Case Study on the OpenTelemetry Demo](https://www.alphaxiv.org/abs/2512.12314v1)

## Installation

Preferred path for the current v1 release:

1. Download the release binary archive for your platform.
2. Download the matching `sheaft-default-config-pack_X.Y.Z.tar.gz`.
3. Verify against `sheaft_X.Y.Z_checksums.txt`.
4. Extract and run the quickstart below.

Release binary archives:

- `sheaft_X.Y.Z_linux_amd64.tar.gz`
- `sheaft_X.Y.Z_linux_arm64.tar.gz`
- `sheaft_X.Y.Z_darwin_amd64.tar.gz`
- `sheaft_X.Y.Z_darwin_arm64.tar.gz`

Fallbacks:

- `go install github.com/MB3R-Lab/Sheaft/cmd/sheaft@vX.Y.Z`
- `go build ./cmd/sheaft`

See [docs/install.md](docs/install.md) for the full install matrix, including OCI image and Helm chart paths.

## Quickstart

If you extracted a release binary plus the default config pack, or if you are in a cloned checkout, this first run is intentionally copy-paste friendly:

```bash
./sheaft run \
  --model examples/outputs/model.sample.json \
  --policy configs/gate.policy.example.yaml \
  --out-dir out/quickstart \
  --seed 42
```

That writes:

- `out/quickstart/model.json`
- `out/quickstart/report.json`
- `out/quickstart/summary.md`

Analysis example:

```bash
./sheaft run \
  --model examples/outputs/snapshot-v1.3.0.sample.json \
  --analysis configs/analysis.v1.1.example.yaml \
  --out-dir out/quickstart-analysis
```

The checked-in snapshot samples use Bering `io.mb3r.bering.snapshot@1.3.0` and include typed edge IDs, reliability evidence, endpoint semantic hints, placements, shared resources, retries, timeouts, and observed latency/error metadata. The `configs/analysis.v1.1.example.yaml` example compares the `1.3.0` primary artifact against the checked-in `1.3.0` baseline through `analysis.baselines`.

On Windows from a source checkout, the same path is:

```powershell
go build ./cmd/sheaft
.\sheaft.exe run --model examples/outputs/model.sample.json --policy configs/gate.policy.example.yaml --out-dir out/quickstart --seed 42
```

## Batch mode

Core batch commands:

```bash
sheaft simulate --model <artifact.json> --policy <policy.yaml> --out <report.json> --seed 42
sheaft simulate --model <artifact.json> --analysis <analysis.yaml> --out <report.json>
sheaft gate --report <report.json> --policy <policy.yaml>
sheaft gate --report <report.json> --analysis <analysis.yaml>
sheaft run --model <artifact.json> --policy <policy.yaml> --out-dir out --seed 42
sheaft run --model <artifact.json> --analysis <analysis.yaml> --out-dir out
```

Optional project-level narrowing can be layered on with:

```bash
sheaft run --model <artifact.json> --analysis <analysis.yaml> --contract-policy configs/contract-policy.example.yaml --out-dir out
```

## Service mode

The long-running service is shipped for evaluation and automation experiments, but it is outside the stable v1 stochastic-connectivity release claim.

The checked-in example is runnable without editing paths:

```bash
./sheaft serve --config configs/sheaft.example.yaml
```

That example:

- listens on `:8080`
- watches the checked-in `1.3.0` sample artifact at `examples/outputs/snapshot.sample.json`
- uses the baseline analysis config `configs/analysis.example.yaml`
- writes history under `.sheaft/history`

HTTP endpoints:

- `/healthz`
- `/readyz`
- `/status`
- `/current-report`
- `/current-diff`
- `/history`
- `/metrics`

## Compatibility with Bering artifacts

Sheaft is intentionally downstream of Bering artifacts and schemas.

- It accepts only the checked-in contract pins listed above.
- `1.3.0` enables path-aware diagnostics and fault-profile analysis when metadata exists.
- Compatibility metadata is published in [compatibility-manifest.json](compatibility-manifest.json).
- The current app-level Bering release/package sync target is `v1.0.0`; schema acceptance still comes only from the pinned contracts above.
- Schema ownership stays with Bering; Sheaft does not redefine those schema versions.
- `--contract-policy` can narrow or deprecate accepted contracts for a specific project, but it cannot expand support beyond the built-in whitelist.

## Known limitations

- `1.3.0` analysis is only as rich as the artifact metadata. Missing retry, timeout, latency, placement, or shared-resource metadata is reported as unavailable rather than guessed.
- Legacy explicit predicates remain service-based even when `1.3.0` edge metadata is present. Edge faults and partial degradations affect journey-based analysis, diagnostics, and explicit `edge_aware` predicates, not old service predicate semantics.
- This release does not introduce or stabilize an upstream discovery pipeline. Discovery remains upstream; the local `discover` helper is experimental only.
- `serve` and its watch loop are suitable for evaluation and non-critical automation, not yet for a stable long-term operational contract.
- The richer analysis surface is available, but its configuration ergonomics and operational conventions may still change in later releases.
- Release automation is designed around GitHub Releases, release manifests, OCI image publication, and an OCI Helm chart; Windows release archives can be built, but Linux and macOS archives are the primary tested binary surface.
- Before using a Sheaft report as blocking release evidence, review the do-not-trust signal catalogue in [Assumptions and Limitations](docs/assumptions-and-limitations.md).

## Development

If GNU Make is available:

```bash
make build
make test
make lint
make smoke-examples
make oracle-suite
make benchmark-slice
```

Direct command equivalents:

```bash
go build ./cmd/sheaft
go test ./...
go vet ./...
```

## Docs

- [Install](docs/install.md)
- [Compatibility](docs/compatibility.md)
- [Compatibility Matrix](docs/compatibility-matrix.md)
- [V1 Major Semantics](docs/v1-major-semantics.md)
- [Release Assets](docs/release-assets.md)
- [Architecture](docs/architecture.md)
- [Methodology](docs/methodology.md)
- [Configuration and Schemas](docs/configuration.md)
- [CI Gate](docs/ci-gate.md)
- [Fixed Benchmark Slice](docs/benchmark-slice.md)
- [Synthetic Oracle Suite](docs/oracle-suite.md)
- [Consumer Semantics v1](docs/consumer-semantics-v1.md)
- [Consumer Semantics v2](docs/consumer-semantics-v2.md)
- [Versioning](VERSIONING.md)
- [Releasing](RELEASING.md)
- [Changelog](CHANGELOG.md)
- [Service Mode](docs/observability-mode.md)
- [Assumptions and Limitations](docs/assumptions-and-limitations.md)

## Example artifacts and configs

- [examples/outputs/model.sample.json](examples/outputs/model.sample.json)
- [examples/outputs/model-v1.3.0.sample.json](examples/outputs/model-v1.3.0.sample.json)
- [examples/outputs/snapshot.sample.json](examples/outputs/snapshot.sample.json)
- [examples/outputs/snapshot-v1.3.0.sample.json](examples/outputs/snapshot-v1.3.0.sample.json)
- [examples/outputs/report.sample.json](examples/outputs/report.sample.json)
- [examples/outputs/posture-generated/report.json](examples/outputs/posture-generated/report.json)
- [examples/outputs/posture-generated/summary.md](examples/outputs/posture-generated/summary.md)
- [configs/gate.policy.example.yaml](configs/gate.policy.example.yaml)
- [configs/analysis.example.yaml](configs/analysis.example.yaml)
- [configs/analysis.v1.1.example.yaml](configs/analysis.v1.1.example.yaml)
- [configs/fault-contract.example.yaml](configs/fault-contract.example.yaml)
- [configs/predicate-contract.example.yaml](configs/predicate-contract.example.yaml)
- [configs/contract-policy.example.yaml](configs/contract-policy.example.yaml)
- [configs/sheaft.example.yaml](configs/sheaft.example.yaml)

## Exit codes

- `0`: success / pass / warn / report
- `2`: gate failure in `mode=fail`
- `1`: runtime, config, or input error

## License

MIT, see [LICENSE](LICENSE).
