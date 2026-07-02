# Install

The current public release line is `v1.0.0`. Prefer release assets for evaluation and automation; use `go install` or `go build` as fallback paths.

The machine-readable entrypoint for release consumers is the `release-manifest.json` asset attached to each GitHub Release. It records exact archive names, checksums, image references, chart version, and the default config pack asset for that release.

## Preferred: Release Binary + Default Config Pack

1. Download the archive for your platform.
2. Download the matching default config pack archive.
3. Verify both against `sheaft_X.Y.Z_checksums.txt` and, if needed, `release-manifest.json`.
4. Extract the binary and the config pack.

Example `v1.0.0` asset names:

- `sheaft_1.0.0_linux_amd64.tar.gz`
- `sheaft_1.0.0_linux_arm64.tar.gz`
- `sheaft_1.0.0_darwin_amd64.tar.gz`
- `sheaft_1.0.0_darwin_arm64.tar.gz`
- `sheaft-default-config-pack_1.0.0.tar.gz`

Minimal first run after extraction:

```bash
tar -xzf sheaft_1.0.0_linux_amd64.tar.gz
tar -xzf sheaft-default-config-pack_1.0.0.tar.gz
./sheaft run --model examples/outputs/model.sample.json --policy configs/gate.policy.example.yaml --out-dir out/quickstart --seed 42
```

## Fallback: `go install`

```bash
go install github.com/MB3R-Lab/Sheaft/cmd/sheaft@v1.0.0
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
docker pull ghcr.io/mb3r-lab/sheaft:v1.0.0
docker run --rm ghcr.io/mb3r-lab/sheaft:v1.0.0 help
```

The image keeps the same CLI entrypoint behavior:

```bash
docker run --rm ghcr.io/mb3r-lab/sheaft:v1.0.0 run --model /data/input.json --analysis /config/analysis.yaml --out-dir /out
```

## OCI Helm Chart

```bash
helm pull oci://ghcr.io/mb3r-lab/charts/sheaft --version 1.0.0
helm install sheaft oci://ghcr.io/mb3r-lab/charts/sheaft --version 1.0.0
```

Chart modes:

- `mode=batch`: renders a `Job` that runs `sheaft run`
- `mode=serve`: renders a `Deployment` plus optional `Service` that runs `sheaft serve`

## Default Config Pack

Each release includes a versioned default config pack archive with checked-in examples:

- example baseline analysis config
- example versioned advanced analysis config
- example fault contract
- example gate policy
- example predicate contract
- example serve config
- example reports and sample Bering-compatible artifacts
- fixed benchmark slice manifest

Typical flow:

1. extract the pack into a repo, CI workspace, or scratch directory;
2. run the sample batch path unchanged once;
3. replace sample artifact and config paths with project-specific inputs.

See [release-assets.md](release-assets.md) for the asset inventory and [compatibility.md](compatibility.md) for upstream contract usage.
