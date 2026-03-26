#!/usr/bin/env bash
# P1: SmoothLLM-style K>1 only for defense smooth_llm (others stay K=1 per harness).
# Full §5 matrix + 3 seeds; higher cost than main P0 run on smooth_llm cells only.
#
# Prerequisites: same as run_trackA_full_paper.sh (real gateway, . ./env).
# Usage:
#   cd Decoupled-LLM-Gateway && bash experiments/scripts/run_trackA_p1_smooth_k5.sh
#
# Env: OUT (default results/trackA_p1_smooth_k5_seed3.json), SMOOTH_K (default 5), SKIP_PREFLIGHT

set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"
# shellcheck source=/dev/null
source "$ROOT/experiments/scripts/paper_common.sh"
paper_source_env_if_present

GATEWAY_URL="${GATEWAY_URL:-http://127.0.0.1:8080}"
UP="${GATEWAY_UPSTREAM:-https://api.deepseek.com}"
MODEL="${OPENAI_MODEL:-deepseek-chat}"
OUT="${OUT:-results/trackA_p1_smooth_k5_seed3.json}"
SMOOTH_K="${SMOOTH_K:-5}"
SKIP_PREFLIGHT="${SKIP_PREFLIGHT:-0}"

paper_refuse_echo_upstream_url "$UP" || exit 1
if [[ "$SKIP_PREFLIGHT" != "1" ]]; then
  export PAPER_PREFLIGHT_SHOW_SAMPLE=0
  paper_preflight_non_echo || exit 1
fi

DEFS="unified,no_obfuscate,no_decoy,intent_only,direct_upstream,smooth_llm,structured_wrap,strong_system_guard,rag_semantic_only"

python3 experiments/run_paper_benchmark.py \
  --gateway-url "$GATEWAY_URL" \
  --upstream-url "$UP" \
  --defenses "$DEFS" \
  --suite full \
  --seeds 42,43,44 \
  --judge-mode heuristic \
  --smooth-llm-samples "$SMOOTH_K" \
  --openai-model "$MODEL" \
  -o "$OUT"

echo "Wrote $OUT (smooth_llm K=$SMOOTH_K; other defenses K=1)"
python3 experiments/scripts/validate_paper_json.py "$OUT"
echo "Compare smooth_llm row to trackA_full_paper_seed3.json (K=1) for distributional RSR / extraction."
