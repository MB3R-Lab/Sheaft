# Install

Sheaft can be consumed as a binary, OCI image, or OCI Helm chart.

The machine-readable entrypoint for automation is `release-manifest.json`. Use that file to resolve exact archive names, checksums, image references, chart version, and the default config pack asset for a given release.

## Binary

1. Download the archive for your platform from the release payload.
2. Verify it against the checksum listed in `release-manifest.json` or the release checksums file.
3. Extract the `sheaft` binary and place it on `PATH`.

Example:

```bash
tar -xzf sheaft_X.Y.Z_linux_amd64.tar.gz
./sheaft help
```

## OCI Image

Pull the tagged image:

```bash
docker pull ghcr.io/mb3r-lab/sheaft:vX.Y.Z
docker run --rm ghcr.io/mb3r-lab/sheaft:vX.Y.Z help
```

The image keeps the current CLI entrypoint behavior:

```bash
docker run --rm ghcr.io/mb3r-lab/sheaft:vX.Y.Z run --model /data/input.json --analysis /config/analysis.yaml --out-dir /out
```

## OCI Helm Chart

Pull or install the chart from OCI:

```bash
helm pull oci://ghcr.io/mb3r-lab/charts/sheaft --version X.Y.Z
helm install sheaft oci://ghcr.io/mb3r-lab/charts/sheaft --version X.Y.Z
```

Chart modes:

- `mode=batch`: renders a `Job` that runs `sheaft run`
- `mode=serve`: renders a `Deployment` plus optional `Service` that runs `sheaft serve`

## Default Config Pack

Each release includes a versioned default config pack archive.

It is intended to be automation-friendly:

- example analysis config
- example gate policy
- example predicate contract
- example serve config
- example journeys override
- example reports and sample Bering-compatible artifacts

Typical flow:

1. extract the pack into your repo or CI workspace;
2. start from the provided config files;
3. override file contents or mount paths as needed for your environment.

## Kubernetes Notes

The OCI chart exposes:

- image repository, tag, and digest override
- mounted policy/model/artifact paths
- batch and serve mode selection
- resources
- env and envFrom
- service and metrics toggles
- pod and container security context

See [release-assets.md](release-assets.md) for the asset inventory and [compatibility.md](compatibility.md) for upstream contract usage.
