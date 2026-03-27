#!/usr/bin/env bash
# P2: output-guard ablation — same Track A matrix as full paper but with --gateway-output-guard.
#
# The *running* gateway process must have been started with the same GATEWAY_OUTPUT_GUARD_URL
# as this script expects (pointing at judge_service). This script starts judge_service locally
# and verifies the URL matches.
#
# Usage:
#   1. Add to ./env (do not commit):
#        export GATEWAY_OUTPUT_GUARD_URL='http://127.0.0.1:8766/judge'
#        export GATEWAY_OUTPUT_GUARD_TEMPLATE="I'm sorry, I can't help with that."
#        export GATEWAY_OUTPUT_GUARD_FAIL_OPEN=1
#   2. Restart gateway: set -a && . ./env && go run ./cmd/gateway
#   3. bash experiments/scripts/run_trackA_p2_output_guard.sh
#
# Env:
#   JUDGE_GUARD_PORT (default 8766), OUT, SKIP_PREFLIGHT, DEFS,
#   RUN_VALIDATE_AND_EXPORT (default 1), SEEDS
#   P2_SUITE (default full) — set to "minimal" with P2_SCENARIOS for smoke / cost control
#   P2_SCENARIOS — comma list when P2_SUITE=minimal
#   MAX_HARMFUL_PROMPTS, MAX_BENIGN_FPR_PROMPTS — optional caps

set -euo pipefail
export PYTHONUNBUFFERED=1
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"
# shellcheck source=/dev/null
source "$ROOT/experiments/scripts/paper_common.sh"
paper_source_env_if_present

JUDGE_GUARD_PORT="${JUDGE_GUARD_PORT:-8766}"
EXPECTED_GUARD="http://127.0.0.1:${JUDGE_GUARD_PORT}/judge"
GATEWAY_URL="${GATEWAY_URL:-http://127.0.0.1:8080}"
UP="${GATEWAY_UPSTREAM:-https://api.deepseek.com}"
MODEL="${OPENAI_MODEL:-deepseek-chat}"
OUT="${OUT:-results/trackA_full_paper_guard_seed3.json}"
SKIP_PREFLIGHT="${SKIP_PREFLIGHT:-0}"
SEEDS="${SEEDS:-42,43,44}"

DEFS="${DEFS:-unified,no_obfuscate,no_decoy,intent_only,direct_upstream,smooth_llm,structured_wrap,strong_system_guard,rag_semantic_only}"
EXTRA=()
[[ -n "${MAX_HARMFUL_PROMPTS:-}" ]] && EXTRA+=(--max-harmful-prompts "$MAX_HARMFUL_PROMPTS")
[[ -n "${MAX_BENIGN_FPR_PROMPTS:-}" ]] && EXTRA+=(--max-benign-fpr-prompts "$MAX_BENIGN_FPR_PROMPTS")

paper_refuse_echo_upstream_url "$UP" || exit 1

if [[ "${GATEWAY_OUTPUT_GUARD_URL:-}" != "$EXPECTED_GUARD" ]]; then
  echo "REFUSE: set GATEWAY_OUTPUT_GUARD_URL=$EXPECTED_GUARD in ./env (or export), restart gateway, then re-run." >&2
  echo "  Current: ${GATEWAY_OUTPUT_GUARD_URL:-<unset>}" >&2
  exit 1
fi

if [[ "$SKIP_PREFLIGHT" != "1" ]]; then
  export PAPER_PREFLIGHT_SHOW_SAMPLE=1
  paper_preflight_non_echo || exit 1
fi

cleanup() {
  [[ -n "${JUDGE_PID:-}" ]] && kill "$JUDGE_PID" 2>/dev/null || true
}
trap cleanup EXIT

export JUDGE_BACKEND="${JUDGE_BACKEND:-heuristic}"
python3 "$ROOT/experiments/judge_service/server.py" --port "$JUDGE_GUARD_PORT" &
JUDGE_PID=$!
for _ in $(seq 1 40); do
  curl -sf "http://127.0.0.1:${JUDGE_GUARD_PORT}/health" >/dev/null 2>&1 && break
  sleep 0.15
done

P2_SUITE="${P2_SUITE:-full}"
BENCH=(
  python3 experiments/run_paper_benchmark.py
  --gateway-url "$GATEWAY_URL"
  --upstream-url "$UP"
  --defenses "$DEFS"
  --judge-mode heuristic
  --gateway-output-guard
  --openai-model "$MODEL"
  --seeds "$SEEDS"
  "${EXTRA[@]}"
)

if [[ "$P2_SUITE" == "full" ]]; then
  "${BENCH[@]}" --suite full -o "$OUT"
else
  # minimal / custom scenarios (e.g. P2_SCENARIOS=harmful_rsr_suite,benign_fpr_suite for smoke)
  SC="${P2_SCENARIOS:-benign_baseline,refusal_keyword,extraction_leak,harmful_rsr_suite,benign_fpr_suite}"
  "${BENCH[@]}" --suite minimal --scenarios "$SC" -o "$OUT"
fi

echo "Wrote $OUT (gateway_output_guard_header=true)"
python3 experiments/scripts/validate_paper_json.py "$OUT"

RUN_VALIDATE_AND_EXPORT="${RUN_VALIDATE_AND_EXPORT:-1}"
if [[ "$RUN_VALIDATE_AND_EXPORT" == "1" ]]; then
  python3 experiments/scripts/export_trackA_table_latex.py \
    --json "$OUT" \
    --variant guard \
    --out-en "${ROOT}/../paper/generated/trackA_guard_table.tex" \
    --out-cn "${ROOT}/../paper/generated/trackA_guard_table_cn.tex"
  echo "Updated ../paper/generated/trackA_guard_table*.tex"
fi
