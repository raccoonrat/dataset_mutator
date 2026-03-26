#!/usr/bin/env bash
# Full Track A matrix (paper-eval-4): 9 defenses × full suite × 3 seeds, no prompt caps.
#
# Usage (from repo root or Decoupled-LLM-Gateway/):
#   ./experiments/scripts/run_trackA_full_paper.sh
#
# Prerequisites:
#   - Gateway on GATEWAY_URL (default http://127.0.0.1:8080) with GATEWAY_UPSTREAM pointing
#     to your OpenAI-compatible API (e.g. https://api.deepseek.com) and a key in the gateway
#     process env (DEEPSEEK_API_KEY / OPENAI_API_KEY / GATEWAY_UPSTREAM_API_KEY).
#   - For direct_upstream, this script passes --upstream-url (same as GATEWAY_UPSTREAM unless
#     overridden) and the benchmark adds Bearer for the client.
#
# Environment:
#   GATEWAY_URL          default http://127.0.0.1:8080
#   GATEWAY_UPSTREAM     default https://api.deepseek.com
#   OUT                  default results/trackA_full_paper_seed3.json
#   OPENAI_MODEL         default deepseek-chat
#
# Local secrets / upstream (optional): place a file ./env in this repo root and run
#   cd Decoupled-LLM-Gateway && . ./env && ./experiments/scripts/run_trackA_full_paper.sh
# This script auto-sources ./env when present (exports all variables defined there).

set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"
if [[ -f ./env ]]; then
  set -a
  # shellcheck source=/dev/null
  . ./env
  set +a
fi
GATEWAY_URL="${GATEWAY_URL:-http://127.0.0.1:8080}"
UP="${GATEWAY_UPSTREAM:-https://api.deepseek.com}"
OUT="${OUT:-results/trackA_full_paper_seed3.json}"
MODEL="${OPENAI_MODEL:-deepseek-chat}"

DEFS="unified,no_obfuscate,no_decoy,intent_only,direct_upstream,smooth_llm,structured_wrap,strong_system_guard,rag_semantic_only"

python3 experiments/run_paper_benchmark.py \
  --gateway-url "$GATEWAY_URL" \
  --upstream-url "$UP" \
  --defenses "$DEFS" \
  --suite full \
  --seeds 42,43,44 \
  --judge-mode heuristic \
  --openai-model "$MODEL" \
  -o "$OUT"

echo "Wrote $OUT"
