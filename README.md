# Sheaft

Sheaft is a downstream resilience posture engine and CI/CD gate for model artifacts produced by Bering or another compatible upstream producer.

## What is Sheaft

Sheaft consumes already-produced model or snapshot artifacts, runs deterministic resilience analysis, emits posture reports, and can fail or warn a delivery pipeline based on policy.

It stays downstream of topology discovery. The public surface in this repository is the CLI and release assets around:

- batch commands: `simulate`, `gate`, `run`
- service commands: `serve`, `watch`

## Stability / Release Status

`v0.1.0` is an experimental public release and should be treated as a technical preview, not a stable GA release.

Stable within the `v0.1.0` technical preview:

- strict acceptance of `io.mb3r.bering.model@1.0.0`
- strict acceptance of `io.mb3r.bering.snapshot@1.0.0`
- batch CLI command names and core flow: `simulate`, `gate`, `run`
- deterministic batch execution for a fixed seed and config
- release archives for Linux and macOS on `amd64` and `arm64`

Experimental in `v0.1.0`:

- long-running `serve` / `watch` posture service
- richer analysis configuration beyond the legacy gate-policy subset
- Helm chart and OCI image operational packaging
- local `discover` helper, which is not the production discovery path

## Supported upstream contracts

Sheaft validates artifacts against an explicit whitelist.

- `io.mb3r.bering.model@1.0.0`
- `io.mb3r.bering.snapshot@1.0.0`

Pinned URIs, digests, and release-line support are tracked in [docs/compatibility-matrix.md](docs/compatibility-matrix.md). The machine-readable compatibility contract is [compatibility-manifest.json](compatibility-manifest.json).

Unknown or mismatched contracts are rejected. There is no silent fallback for unsupported upstream schemas.

## Installation

Preferred path for `v0.1.0`:

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
  --model examples/outputs/snapshot.sample.json \
  --analysis configs/analysis.example.yaml \
  --out-dir out/quickstart-analysis
```

The checked-in snapshot sample mirrors the current Bering `io.mb3r.bering.snapshot@1.0.0` envelope. The accompanying analysis example layers explicit predicate and weight overrides on top of that snapshot sample.

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

The long-running service remains experimental in `v0.1.0`, but it is included in the public technical preview.

The checked-in example is runnable without editing paths:

```bash
./sheaft serve --config configs/sheaft.example.yaml
```

That example:

- listens on `:8080`
- watches the checked-in sample artifact at `examples/outputs/snapshot.sample.json`
- uses `configs/analysis.example.yaml`
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
- Compatibility metadata is published in [compatibility-manifest.json](compatibility-manifest.json).
- Schema ownership stays with Bering; Sheaft does not redefine those schema versions.
- `--contract-policy` can narrow or deprecate accepted contracts for a specific project, but it cannot expand support beyond the built-in whitelist.

## Known limitations

- `v0.1.0` supports only `io.mb3r.bering.model@1.0.0` and `io.mb3r.bering.snapshot@1.0.0`.
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
- [Versioning](VERSIONING.md)
- [Releasing](RELEASING.md)
- [Changelog](CHANGELOG.md)
- [Service Mode](docs/observability-mode.md)
- [Assumptions and Limitations](docs/assumptions-and-limitations.md)

## Example artifacts and configs

- [examples/outputs/model.sample.json](examples/outputs/model.sample.json)
- [examples/outputs/snapshot.sample.json](examples/outputs/snapshot.sample.json)
- [examples/outputs/report.sample.json](examples/outputs/report.sample.json)
- [examples/outputs/posture-generated/report.json](examples/outputs/posture-generated/report.json)
- [examples/outputs/posture-generated/summary.md](examples/outputs/posture-generated/summary.md)
- [configs/gate.policy.example.yaml](configs/gate.policy.example.yaml)
- [configs/analysis.example.yaml](configs/analysis.example.yaml)
- [configs/predicate-contract.example.yaml](configs/predicate-contract.example.yaml)
- [configs/contract-policy.example.yaml](configs/contract-policy.example.yaml)
- [configs/sheaft.example.yaml](configs/sheaft.example.yaml)

## Exit codes

- `0`: success / pass / warn / report
- `2`: gate failure in `mode=fail`
- `1`: runtime, config, or input error

## License

MIT, see [LICENSE](LICENSE).
