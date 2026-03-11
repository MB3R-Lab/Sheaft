# Collector-Fed Artifact Chain

One generic posture pipeline is:

1. A telemetry collector exports traces or derived topology data into a model-producer workflow.
2. The producer writes model or snapshot artifacts into a shared artifact directory.
3. Sheaft watches that directory with `sheaft serve --config configs/sheaft.example.yaml`.
4. Prometheus scrapes `/metrics` and Grafana visualizes current posture and diffs.

This keeps topology discovery upstream and lets Sheaft remain a downstream model consumer.
