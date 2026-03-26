#!/usr/bin/env bash
# P1: paper-eval-4 with --judge-mode http (bundled judge_service).
# Subset: harmful_rsr_suite + benign_fpr_suite; defenses: unified, direct_upstream, smooth_llm (cost control).
#
# Prerequisites: gateway on GATEWAY_URL with real GATEWAY_UPSTREAM (see P0 checklist).
# Usage:
#   cd Decoupled-LLM-Gateway && bash experiments/scripts/run_trackA_p1_http_judge_subset.sh
#
# Env:
#   JUDGE_PORT (default 8765), JUDGE_BACKEND (default heuristic), OUT, SKIP_PREFLIGHT
#   MAX_HARMFUL_PROMPTS, MAX_BENIGN_FPR_PROMPTS — optional caps (smoke / cost control)

set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"
# shellcheck source=/dev/null
source "$ROOT/experiments/scripts/paper_common.sh"
paper_source_env_if_present

GATEWAY_URL="${GATEWAY_URL:-http://127.0.0.1:8080}"
UP="${GATEWAY_UPSTREAM:-https://api.deepseek.com}"
MODEL="${OPENAI_MODEL:-deepseek-chat}"
OUT="${OUT:-results/trackA_p1_http_judge_subset.json}"
SEEDS="${SEEDS:-42,43,44}"
JUDGE_PORT="${JUDGE_PORT:-8765}"
SKIP_PREFLIGHT="${SKIP_PREFLIGHT:-0}"
# POST path is ignored by judge_service.Handler; /judge matches self-check URL in server.py.
JUDGE_URL="http://127.0.0.1:${JUDGE_PORT}/judge"

paper_refuse_echo_upstream_url "$UP" || exit 1

if [[ "$SKIP_PREFLIGHT" != "1" ]]; then
  export PAPER_PREFLIGHT_SHOW_SAMPLE=0
  paper_preflight_non_echo || exit 1
fi

cleanup() {
  [[ -n "${JUDGE_PID:-}" ]] && kill "$JUDGE_PID" 2>/dev/null || true
}
trap cleanup EXIT

export JUDGE_BACKEND="${JUDGE_BACKEND:-heuristic}"
python3 "$ROOT/experiments/judge_service/server.py" --port "$JUDGE_PORT" &
JUDGE_PID=$!
for _ in $(seq 1 30); do
  curl -sf "http://127.0.0.1:${JUDGE_PORT}/health" >/dev/null 2>&1 && break
  sleep 0.2
done

export PAPER_EVAL_JUDGE_URL="$JUDGE_URL"

DEFS="unified,direct_upstream,smooth_llm"
SCENARIOS="harmful_rsr_suite,benign_fpr_suite"
EXTRA=()
[[ -n "${MAX_HARMFUL_PROMPTS:-}" ]] && EXTRA+=(--max-harmful-prompts "$MAX_HARMFUL_PROMPTS")
[[ -n "${MAX_BENIGN_FPR_PROMPTS:-}" ]] && EXTRA+=(--max-benign-fpr-prompts "$MAX_BENIGN_FPR_PROMPTS")

python3 experiments/run_paper_benchmark.py \
  --gateway-url "$GATEWAY_URL" \
  --upstream-url "$UP" \
  --defenses "$DEFS" \
  --scenarios "$SCENARIOS" \
  --seeds "$SEEDS" \
  --judge-mode http \
  --judge-url "$JUDGE_URL" \
  --openai-model "$MODEL" \
  "${EXTRA[@]}" \
  -o "$OUT"

echo "Wrote $OUT (judge_mode=http, backend=$JUDGE_BACKEND)"
python3 experiments/scripts/validate_paper_json.py "$OUT" || true
