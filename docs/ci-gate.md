# CI Gate

## Legacy Policy Flow

```bash
sheaft run \
  --model path/to/model.json \
  --policy configs/gate.policy.example.yaml \
  --out-dir out \
  --seed 42
```

Generated artifacts:

- `out/model.json`
- `out/report.json`
- `out/summary.md`

## Rich Analysis Flow

```bash
sheaft run \
  --model path/to/artifact.json \
  --analysis configs/analysis.v1.1.example.yaml \
  --out-dir out
```

Use the richer config when CI needs:

- multiple scenario profiles
- weighted aggregates
- baseline comparisons
- external predicate overlays
- explicit multi-profile gate rules

## Bering Artifact Handoff Contract

Use the same handoff layout in every CI system:

- upstream Bering job writes the selected artifact to `artifacts/input.json`
- Sheaft reads only that file via `--model artifacts/input.json`
- Sheaft outputs always land in `out/`
- retain `artifacts/input.json` for short-term debugging and `out/` for downstream review

Recommended retention windows:

- upstream artifact: 7 days
- Sheaft outputs: 14 days

This keeps all CI templates aligned across GitHub Actions, GitLab CI, and Jenkins.

In the example templates below, replace the sample `BERING_ARTIFACT_SOURCE` value with the actual fetch/copy step that hands off the Bering-produced artifact into the workspace.

## Strict Schema Checks

Sheaft is a strict downstream consumer. `sheaft run` and `sheaft simulate` will fail with exit code `1` when the incoming artifact declares an unsupported contract, mismatched schema URI, or mismatched digest. Supported Bering contract lines are `1.0.0` and `1.1.0`; `1.0.0` remains the baseline comparison line.

No extra CI flag is needed for strict checking: it is already part of artifact loading.

## Exit Codes

- `0`: pass / warn / report
- `2`: gate failure in `mode=fail`
- `1`: input, contract, config, or runtime error

## Reference Templates

- GitHub Actions: [examples/ci/github-actions.sheaft.yml](../examples/ci/github-actions.sheaft.yml)
- GitLab CI: [examples/ci/gitlab-ci.sheaft.yml](../examples/ci/gitlab-ci.sheaft.yml)
- Jenkins: [examples/ci/Jenkinsfile](../examples/ci/Jenkinsfile)

All three templates cover:

- Bering artifact handoff into `artifacts/input.json`
- `sheaft run` execution with the richer analysis config
- artifact publishing for both the original input and Sheaft outputs
- native CI failure propagation from Sheaft exit codes

## Smoke Validation

This repository validates the CI handoff contract in three layers:

- `sh scripts/ci/check-ci-handoff-templates.sh` verifies that the example templates and docs keep the agreed handoff paths, retention windows, artifact publishing steps, and smoke workflow references.
- `sh scripts/ci/smoke-ci-handoff.sh native` runs the same handoff layout locally via `go run ./cmd/sheaft`.
- `sh scripts/ci/smoke-ci-handoff.sh docker` exercises the Docker execution path used by the example templates, including local runs from Windows Git Bash.

The GitHub Actions workflow at `.github/workflows/ci-template-smoke.yml` runs both smoke modes on every pull request and on pushes to `main`.

## GitHub Actions Example

```yaml
name: sheaft-gate
on: [pull_request]
env:
  BERING_ARTIFACT_SOURCE: examples/outputs/snapshot-v1.1.0.sample.json
jobs:
  bering-artifact:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Produce or fetch Bering artifact
        run: |
          mkdir -p artifacts
          cp "$BERING_ARTIFACT_SOURCE" artifacts/input.json
      - name: Upload Bering artifact
        uses: actions/upload-artifact@v4
        with:
          name: bering-model
          path: artifacts/input.json
          if-no-files-found: error
          retention-days: 7
  sheaft-gate:
    needs: bering-artifact
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Download Bering artifact
        uses: actions/download-artifact@v4
        with:
          name: bering-model
          path: artifacts
      - name: Build image
        run: docker build -f build/Dockerfile -t sheaft:ci .
      - name: Run Sheaft
        run: |
          docker run --rm -v "$PWD:/workspace" -w /workspace sheaft:ci run \
            --model artifacts/input.json \
            --analysis configs/analysis.v1.1.example.yaml \
            --out-dir out
      - name: Upload Sheaft outputs
        uses: actions/upload-artifact@v4
        with:
          name: sheaft-report
          path: |
            artifacts/input.json
            out/
          if-no-files-found: error
          retention-days: 14
```

## GitLab CI Example

```yaml
stages:
  - bering
  - posture

variables:
  BERING_ARTIFACT_SOURCE: examples/outputs/snapshot-v1.1.0.sample.json
  SHEAFT_IMAGE: "$CI_REGISTRY_IMAGE/sheaft-ci:$CI_COMMIT_SHA"

bering_artifact:
  stage: bering
  image: alpine:3.20
  script:
    - mkdir -p artifacts
    - cp "$BERING_ARTIFACT_SOURCE" artifacts/input.json
  artifacts:
    when: always
    expire_in: 7 days
    paths:
      - artifacts/input.json

sheaft_gate:
  stage: posture
  image: docker:27-cli
  services:
    - docker:27-dind
  variables:
    DOCKER_HOST: tcp://docker:2375
    DOCKER_TLS_CERTDIR: ""
  needs:
    - job: bering_artifact
      artifacts: true
  script:
    - docker build -f build/Dockerfile -t "$SHEAFT_IMAGE" .
    - docker run --rm -v "$CI_PROJECT_DIR:/workspace" -w /workspace "$SHEAFT_IMAGE" run --model artifacts/input.json --analysis configs/analysis.v1.1.example.yaml --out-dir out
  artifacts:
    when: always
    expire_in: 14 days
    paths:
      - artifacts/input.json
      - out/
```

## Jenkins Example

```groovy
pipeline {
  agent any
  environment {
    BERING_ARTIFACT_SOURCE = 'examples/outputs/snapshot-v1.1.0.sample.json'
  }
  options {
    timestamps()
  }
  stages {
    stage('Checkout') {
      steps {
        checkout scm
      }
    }
    stage('Bering Artifact') {
      steps {
        sh '''
          mkdir -p artifacts
          cp "$BERING_ARTIFACT_SOURCE" artifacts/input.json
        '''
        stash name: 'bering-artifact', includes: 'artifacts/input.json'
      }
    }
    stage('Sheaft Gate') {
      steps {
        unstash 'bering-artifact'
        sh '''
          docker build -f build/Dockerfile -t sheaft:ci .
          docker run --rm -v "$WORKSPACE:/workspace" -w /workspace sheaft:ci run \
            --model artifacts/input.json \
            --analysis configs/analysis.v1.1.example.yaml \
            --out-dir out
        '''
      }
    }
  }
  post {
    always {
      archiveArtifacts artifacts: 'artifacts/input.json,out/**', fingerprint: true
    }
  }
}
```
