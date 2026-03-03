#!/usr/bin/env sh
set -eu

MODEL_PATH="${1:-examples/outputs/model.sample.json}"
POLICY_PATH="${2:-configs/gate.policy.example.yaml}"
OUT_DIR="${3:-out}"
SEED="${4:-42}"

mkdir -p "${OUT_DIR}"

sheaft run \
  --model "${MODEL_PATH}" \
  --policy "${POLICY_PATH}" \
  --out-dir "${OUT_DIR}" \
  --seed "${SEED}"
