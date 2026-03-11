# Service Mode

Use `sheaft serve` when posture needs to be refreshed continuously from externally produced artifacts.

## What It Watches

- one plain model file
- one snapshot envelope file
- a directory of artifacts, where Sheaft selects the newest matching file
- a stable file path that is replaced or rewritten over time

## Behavior

- initial recompute on startup
- optional polling
- optional filesystem event reloads
- bounded in-memory report history
- optional on-disk report history
- Prometheus/OpenMetrics metrics
- JSON status and diff endpoints for automation

## Example

```bash
sheaft serve --config configs/sheaft.example.yaml
```

Example config: [configs/sheaft.example.yaml](../configs/sheaft.example.yaml)

## HTTP Endpoints

- `/healthz`: process liveness
- `/readyz`: ready after the first successful recompute
- `/status`: concise status JSON for dashboards and automation
- `/current-report`: full current report
- `/current-diff`: diff versus previous and configured baselines
- `/history`: bounded report history
- `/metrics`: Prometheus/OpenMetrics output

## Operational Notes

- Service mode does not perform topology discovery.
- The same seed and config produce deterministic simulation output for the same artifact.
- Artifact changes are detected by path selection plus content digest changes.
