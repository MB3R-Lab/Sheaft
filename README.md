# Sheaft

[![Release](https://img.shields.io/github/v/release/MB3R-Lab/Sheaft)](https://github.com/MB3R-Lab/Sheaft/releases)
[![ci-template-smoke](https://img.shields.io/github/actions/workflow/status/MB3R-Lab/Sheaft/ci-template-smoke.yml?branch=main&label=ci-template-smoke)](https://github.com/MB3R-Lab/Sheaft/actions/workflows/ci-template-smoke.yml)
[![schema-contract](https://img.shields.io/github/actions/workflow/status/MB3R-Lab/Sheaft/schema-contract.yml?branch=main&label=schema-contract)](https://github.com/MB3R-Lab/Sheaft/actions/workflows/schema-contract.yml)
[![Go version](https://img.shields.io/github/go-mod/go-version/MB3R-Lab/Sheaft)](https://github.com/MB3R-Lab/Sheaft/blob/main/go.mod)
[![Status](https://img.shields.io/badge/status-technical_preview-orange)](https://github.com/MB3R-Lab/Sheaft/releases/tag/v0.2.0)
[![Bering support](https://img.shields.io/badge/Bering-1.0%20%7C%201.1-blue)](https://github.com/MB3R-Lab/Sheaft/blob/main/docs/compatibility-matrix.md)

Sheaft is a downstream resilience posture engine and CI/CD gate for model artifacts produced by Bering or another compatible upstream producer.

## What is Sheaft

Sheaft consumes already-produced model or snapshot artifacts, runs deterministic resilience analysis, emits posture reports, and can fail or warn a delivery pipeline based on policy.

It stays downstream of topology discovery. The public surface in this repository is the CLI and release assets around:

- batch commands: `simulate`, `gate`, `run`
- service commands: `serve`, `watch`

## Stability / Release Status

The current public release is `v0.2.0`. The `v0.2.x` line is an experimental public release and should be treated as a technical preview, not a stable GA release.

Stable within the `v0.2.0` technical preview:

- strict acceptance of the baseline Bering contract line: `io.mb3r.bering.model@1.0.0` and `io.mb3r.bering.snapshot@1.0.0`
- strict acceptance of the advanced Bering contract line: `io.mb3r.bering.model@1.1.0` and `io.mb3r.bering.snapshot@1.1.0`
- batch CLI command names and core flow: `simulate`, `gate`, `run`
- deterministic batch execution for a fixed seed and config
- cross-line baseline comparison through `analysis.baselines`
- additive advanced analysis when `1.1.0` metadata exists
- release archives for Linux and macOS on `amd64` and `arm64`

Experimental in `v0.2.x`:

- long-running `serve` / `watch` posture service
- richer analysis configuration beyond the legacy gate-policy subset
- Helm chart and OCI image operational packaging
- local `discover` helper, which is not the production discovery path

## Supported upstream contracts

Sheaft validates artifacts against an explicit whitelist.

These are alternative accepted upstream contract lines for incoming artifacts, not simultaneous version dependencies for a single artifact.

- `io.mb3r.bering.model@1.0.0`
- `io.mb3r.bering.snapshot@1.0.0`
- `io.mb3r.bering.model@1.1.0`
- `io.mb3r.bering.snapshot@1.1.0`

Pinned URIs, digests, and release-line support are tracked in [docs/compatibility-matrix.md](docs/compatibility-matrix.md). The machine-readable compatibility contract is [compatibility-manifest.json](compatibility-manifest.json).

Unknown or mismatched contracts are rejected. There is no silent fallback for unsupported upstream schemas.

`1.0.0` remains the baseline semantics line and the reference artifact line for cross-version comparisons. `1.1.0` adds richer typed metadata for timeout, retry, placement, shared-resource, and edge-scoped analysis when the artifact provides it.

## Installation

Preferred path for the current technical preview release:

1. Download the release binary archive for your platform.
2. Download the matching `sheaft-default-config-pack_X.Y.Z.tar.gz`.
3. Verify against `sheaft_X.Y.Z_checksums.txt`.
4. Extract and run the quickstart below.

Minimum planned binary archives:

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
  --model examples/outputs/snapshot-v1.1.0.sample.json \
  --analysis configs/analysis.v1.1.example.yaml \
  --out-dir out/quickstart-analysis
```

The checked-in baseline snapshot sample mirrors the Bering `io.mb3r.bering.snapshot@1.0.0` envelope. The `1.1.0` sample adds typed edge IDs, placements, shared resources, retries, timeouts, and observed latency/error metadata. The `configs/analysis.v1.1.example.yaml` example compares that `1.1.0` primary artifact directly against the `1.0.0` baseline artifact through `analysis.baselines`.

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

The long-running service remains experimental in `v0.1.x`, but it is included in the public technical preview.

The checked-in example is runnable without editing paths:

```bash
./sheaft serve --config configs/sheaft.example.yaml
```

That example:

- listens on `:8080`
- watches the checked-in baseline `1.0.0` sample artifact at `examples/outputs/snapshot.sample.json`
- uses the legacy baseline analysis config `configs/analysis.example.yaml`
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
- `1.0.0` is kept as the stable fail-stop baseline semantics line.
- `1.1.0` enables additive path-aware diagnostics and fault-profile analysis when metadata exists.
- Compatibility metadata is published in [compatibility-manifest.json](compatibility-manifest.json).
- Schema ownership stays with Bering; Sheaft does not redefine those schema versions.
- `--contract-policy` can narrow or deprecate accepted contracts for a specific project, but it cannot expand support beyond the built-in whitelist.

## Known limitations

- `1.1.0` analysis is only as rich as the artifact metadata. Missing retry, timeout, latency, placement, or shared-resource metadata is reported as unavailable rather than guessed.
- Legacy explicit predicates remain service-based even when `1.1.0` edge metadata is present. Edge faults and partial degradations affect journey-based analysis and diagnostics, not old explicit predicate semantics.
- This release does not introduce or stabilize an upstream discovery pipeline. Discovery remains upstream; the local `discover` helper is experimental only.
- `serve` / `watch` are suitable for technical-preview evaluation, not yet for a stable long-term operational contract.
- The richer analysis surface is available, but its configuration ergonomics and operational conventions may still change in later `0.x` releases.
- Release automation is designed around GitHub Releases, release manifests, OCI image publication, and an OCI Helm chart; Windows release archives can be built, but they are not the primary tested surface in this preview.

## Development

If GNU Make is available:

```bash
make build
make test
make lint
make smoke-examples
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
- [Release Assets](docs/release-assets.md)
- [Architecture](docs/architecture.md)
- [Methodology](docs/methodology.md)
- [Configuration and Schemas](docs/configuration.md)
- [CI Gate](docs/ci-gate.md)
- [Consumer Semantics v1](docs/consumer-semantics-v1.md)
- [Consumer Semantics v2](docs/consumer-semantics-v2.md)
- [Versioning](VERSIONING.md)
- [Releasing](RELEASING.md)
- [Changelog](CHANGELOG.md)
- [Service Mode](docs/observability-mode.md)
- [Assumptions and Limitations](docs/assumptions-and-limitations.md)

## Example artifacts and configs

- [examples/outputs/model.sample.json](examples/outputs/model.sample.json)
- [examples/outputs/model-v1.1.0.sample.json](examples/outputs/model-v1.1.0.sample.json)
- [examples/outputs/snapshot.sample.json](examples/outputs/snapshot.sample.json)
- [examples/outputs/snapshot-v1.0.0.sample.json](examples/outputs/snapshot-v1.0.0.sample.json)
- [examples/outputs/snapshot-v1.1.0.sample.json](examples/outputs/snapshot-v1.1.0.sample.json)
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
