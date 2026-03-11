# Migration Notes

## From Batch-Only v0

Nothing changes for existing batch users:

- `simulate`, `gate`, and `run` still work
- `--policy` remains valid
- deterministic behavior for fixed seed and config is preserved

## When To Adopt The Richer Config

Move from `--policy` to `--analysis` when you need any of:

- multiple named profiles
- weighted aggregates
- baseline comparisons
- external predicate and workload overlays
- explicit multi-profile gate rules

## Service Adoption

To move from batch-only usage to continuous posture monitoring:

1. produce plain model or snapshot artifacts upstream
2. place them at a stable file path or in a watched directory
3. start `sheaft serve --config configs/sheaft.example.yaml`
4. scrape `/metrics` and consume `/status` or `/current-report`

## Contracts

Sheaft now validates against a supported-contract whitelist instead of a single hardcoded exact contract path in the model package. The currently supported contracts are listed in the README and enforced by explicit adapters.

## Reports

Existing consumers can keep reading the legacy top-level report fields:

- `simulation`
- `endpoint_results`
- `summary`
- `policy_evaluation`

New consumers can additionally use:

- `input_artifact`
- `provenance`
- `profiles`
- `diffs`
- `generated_at`
- `recompute_duration_ms`
