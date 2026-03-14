# Install

The current public technical preview is `v0.1.1`. Prefer release assets for evaluation and automation; use `go install` or `go build` as fallback paths.

The machine-readable entrypoint for release consumers is `release-manifest.json`. It records exact archive names, checksums, image references, chart version, and the default config pack asset for a given release.

## Preferred: Release Binary + Default Config Pack

1. Download the archive for your platform.
2. Download the matching default config pack archive.
3. Verify both against `sheaft_X.Y.Z_checksums.txt` and, if needed, `release-manifest.json`.
4. Extract the binary and the config pack.

Example asset names:

- `sheaft_X.Y.Z_linux_amd64.tar.gz`
- `sheaft_X.Y.Z_linux_arm64.tar.gz`
- `sheaft_X.Y.Z_darwin_amd64.tar.gz`
- `sheaft_X.Y.Z_darwin_arm64.tar.gz`
- `sheaft-default-config-pack_X.Y.Z.tar.gz`

Minimal first run after extraction:

```bash
tar -xzf sheaft_X.Y.Z_linux_amd64.tar.gz
tar -xzf sheaft-default-config-pack_X.Y.Z.tar.gz
./sheaft run --model examples/outputs/model.sample.json --policy configs/gate.policy.example.yaml --out-dir out/quickstart --seed 42
```

## Fallback: `go install`

```bash
go install github.com/MB3R-Lab/Sheaft/cmd/sheaft@vX.Y.Z
sheaft help
```

## Fallback: `go build`

From a clean checkout:

```bash
go build ./cmd/sheaft
./sheaft run --model examples/outputs/model.sample.json --policy configs/gate.policy.example.yaml --out-dir out/quickstart --seed 42
```

## OCI Image

```bash
docker pull ghcr.io/mb3r-lab/sheaft:vX.Y.Z
docker run --rm ghcr.io/mb3r-lab/sheaft:vX.Y.Z help
```

The image keeps the same CLI entrypoint behavior:

```bash
docker run --rm ghcr.io/mb3r-lab/sheaft:vX.Y.Z run --model /data/input.json --analysis /config/analysis.yaml --out-dir /out
```

## OCI Helm Chart

```bash
helm pull oci://ghcr.io/mb3r-lab/charts/sheaft --version X.Y.Z
helm install sheaft oci://ghcr.io/mb3r-lab/charts/sheaft --version X.Y.Z
```

Chart modes:

- `mode=batch`: renders a `Job` that runs `sheaft run`
- `mode=serve`: renders a `Deployment` plus optional `Service` that runs `sheaft serve`

## Default Config Pack

Each release includes a versioned default config pack archive with checked-in examples:

- example analysis config
- example gate policy
- example predicate contract
- example serve config
- example reports and sample Bering-compatible artifacts

Typical flow:

1. extract the pack into a repo, CI workspace, or scratch directory;
2. run the sample batch path unchanged once;
3. replace sample artifact and config paths with project-specific inputs.

See [release-assets.md](release-assets.md) for the asset inventory and [compatibility.md](compatibility.md) for upstream contract usage.
