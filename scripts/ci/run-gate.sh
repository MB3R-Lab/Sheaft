#!/usr/bin/env sh
set -eu

MODEL_PATH="${1:-examples/outputs/model.sample.json}"
POLICY_PATH="${2:-configs/gate.policy.example.yaml}"
OUT_DIR="${3:-out}"
SEED="${4:-42}"
JOURNEYS_PATH="${5:-}"

mkdir -p "${OUT_DIR}"

if [ -n "${JOURNEYS_PATH}" ]; then
  sheaft run \
    --model "${MODEL_PATH}" \
    --journeys "${JOURNEYS_PATH}" \
    --policy "${POLICY_PATH}" \
    --out-dir "${OUT_DIR}" \
    --seed "${SEED}"
else
  sheaft run \
    --model "${MODEL_PATH}" \
    --policy "${POLICY_PATH}" \
    --out-dir "${OUT_DIR}" \
    --seed "${SEED}"
fi
