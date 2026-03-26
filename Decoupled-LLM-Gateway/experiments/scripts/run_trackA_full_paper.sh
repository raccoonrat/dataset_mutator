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
#   GATEWAY_UPSTREAM     default https://api.deepseek.com (must match the *running* gateway process)
#   OUT                  default results/trackA_full_paper_seed3.json
#   OPENAI_MODEL         default deepseek-chat
#   SKIP_PREFLIGHT       set to 1 to skip gateway probe (not recommended for real-paper runs)
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
SKIP_PREFLIGHT="${SKIP_PREFLIGHT:-0}"

preflight_gateway() {
  local body resp
  body=$(printf '%s' "{\"model\":\"${MODEL}\",\"messages\":[{\"role\":\"user\",\"content\":\"Reply with exactly: OK\"}]}")
  if ! resp=$(curl -sS -m 180 "${GATEWAY_URL}/v1/chat/completions" \
    -H 'Content-Type: application/json' -d "$body"); then
    echo "PREFLIGHT FAIL: cannot reach ${GATEWAY_URL}/v1/chat/completions (start gateway first?)" >&2
    return 1
  fi
  if echo "$resp" | grep -q '\[echo\]'; then
    echo "PREFLIGHT FAIL: gateway reply contains [echo] — GATEWAY_UPSTREAM is probably still echo-llm." >&2
    echo "Fix: in the same shell as the gateway, set GATEWAY_UPSTREAM to your real API (see .env), restart gateway, re-run." >&2
    return 1
  fi
  echo "PREFLIGHT OK: gateway returned non-echo completion (first ~120 chars):"
  echo "$resp" | head -c 120
  echo "..."
}

case "$UP" in
  *127.0.0.1:9090*|*localhost:9090*|:9090*)
    echo "REFUSE: GATEWAY_UPSTREAM=$UP looks like echo-llm. For paper JSON, point at a real OpenAI-compatible API." >&2
    exit 1
    ;;
esac

if [[ "$SKIP_PREFLIGHT" != "1" ]]; then
  preflight_gateway
fi

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

# Validate JSON (echo vs real mix) and refresh paper LaTeX fragments unless disabled.
RUN_VALIDATE_AND_EXPORT="${RUN_VALIDATE_AND_EXPORT:-1}"
if [[ "$RUN_VALIDATE_AND_EXPORT" == "1" ]]; then
  python3 experiments/scripts/validate_paper_json.py "$OUT"
  python3 experiments/scripts/export_trackA_table_latex.py \
    --json "$OUT" \
    --out-en "${ROOT}/../paper/generated/trackA_main_table.tex" \
    --out-cn "${ROOT}/../paper/generated/trackA_main_table_cn.tex"
  echo "Updated ../paper/generated/trackA_main_table*.tex"
fi
