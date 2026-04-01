#!/usr/bin/env bash
# 小规模主表烟测：与 full paper 对齐的 judge-mode / prompt_workers / checkpoint 语义，压低 API 成本。
#
# 与 run_trackA_full_paper.sh 的差异：
#   - 仅 3 个 defense（unified, direct_upstream, smooth_llm），单 seed
#   - 场景不含 hpm / decoy / wild / strongreject（避免长列表与压测）
#   - harmful / benign FPR 截断；multi_round 缩短
#   - 默认本脚本内启动本地 judge_service（heuristic）；LG4 见下方
#
# Usage:
#   cd Decoupled-LLM-Gateway && . ./env   # 可选：密钥与上游
#   bash experiments/scripts/run_trackA_smoke_main_table.sh
#
# LG4（OpenRouter）烟测：先起 judge，再禁用自动起裁判：
#   bash experiments/scripts/run_openrouter_llama_guard4_judge.sh   # 另一终端
#   export SKIP_JUDGE_AUTOSTART=1 PAPER_EVAL_JUDGE_URL=http://127.0.0.1:8765/judge
#   export JUDGE_MODEL_REVISION=meta-llama/llama-guard-4-12b
#   bash experiments/scripts/run_trackA_smoke_main_table.sh
#
# Env:
#   GATEWAY_URL, GATEWAY_UPSTREAM, OPENAI_MODEL — 与 full paper 相同
#   SKIP_PREFLIGHT=1 — 跳过非 echo 探针（仅调试）
#   SKIP_CONDA_ACTIVATE=1 — 默认已开启；需要 dataset_mutator 环境可设为 0
#   SMOKE_OUT — 输出 JSON 路径
#   SMOKE_PROMPT_WORKERS — 默认 2
#   MAX_HARMFUL_PROMPTS, MAX_BENIGN_FPR_PROMPTS, SMOKE_MAX_ROUNDS — 可调

set -euo pipefail
export PYTHONUNBUFFERED=1
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

if [[ "${SKIP_CONDA_ACTIVATE:-1}" == "1" ]]; then
  :
else
  if [[ "${CONDA_DEFAULT_ENV:-}" != "dataset_mutator" ]] && command -v conda >/dev/null 2>&1; then
    # shellcheck source=/dev/null
    source "$(conda info --base)/etc/profile.d/conda.sh" && conda activate dataset_mutator
  fi
fi

# shellcheck source=/dev/null
source "$ROOT/experiments/scripts/paper_common.sh"
paper_source_env_if_present
paper_apply_hf_proxy_env

GATEWAY_URL="${GATEWAY_URL:-http://127.0.0.1:8080}"
UP="${GATEWAY_UPSTREAM:-https://api.deepseek.com}"
MODEL="${OPENAI_MODEL:-deepseek-chat}"
SKIP_PREFLIGHT="${SKIP_PREFLIGHT:-0}"
JUDGE_PORT="${JUDGE_PORT:-8765}"
SMOKE_OUT="${SMOKE_OUT:-results/trackA_smoke_main_table_$(date +%Y%m%d_%H%M%S).json}"
SMOKE_PROMPT_WORKERS="${SMOKE_PROMPT_WORKERS:-2}"
MAX_HARMFUL_PROMPTS="${MAX_HARMFUL_PROMPTS:-8}"
MAX_BENIGN_FPR_PROMPTS="${MAX_BENIGN_FPR_PROMPTS:-8}"
SMOKE_MAX_ROUNDS="${SMOKE_MAX_ROUNDS:-2}"
SKIP_JUDGE_AUTOSTART="${SKIP_JUDGE_AUTOSTART:-0}"

HARMFUL="${HARMFUL_PROMPTS_FILE:-$ROOT/experiments/data/jbb_misuse_100_en.txt}"
BENIGN="${BENIGN_PROMPTS_FILE:-$ROOT/experiments/data/jbb_benign_100_en.txt}"
if [[ ! -f "$HARMFUL" ]]; then HARMFUL="$ROOT/experiments/data/harmful_prompts_trackA_en.txt"; fi
if [[ ! -f "$BENIGN" ]]; then BENIGN="$ROOT/experiments/data/benign_prompts_en.txt"; fi

paper_refuse_echo_upstream_url "$UP" || exit 1

if [[ "$SKIP_PREFLIGHT" != "1" ]]; then
  export PAPER_PREFLIGHT_SHOW_SAMPLE=1
  paper_preflight_non_echo || exit 1
fi

cleanup() {
  [[ -n "${JUDGE_PID:-}" ]] && kill "$JUDGE_PID" 2>/dev/null || true
}
trap cleanup EXIT

if [[ "$SKIP_JUDGE_AUTOSTART" == "1" ]]; then
  PAPER_EVAL_JUDGE_URL="${PAPER_EVAL_JUDGE_URL:-http://127.0.0.1:${JUDGE_PORT}/judge}"
  export PAPER_EVAL_JUDGE_URL
  echo "[smoke] using existing judge at $PAPER_EVAL_JUDGE_URL"
elif curl -sf "http://127.0.0.1:${JUDGE_PORT}/health" >/dev/null 2>&1; then
  export PAPER_EVAL_JUDGE_URL="http://127.0.0.1:${JUDGE_PORT}/judge"
  echo "[smoke] port ${JUDGE_PORT} already has a judge (/health ok), reuse $PAPER_EVAL_JUDGE_URL (set SKIP_JUDGE_AUTOSTART=1 to silence)"
else
  export JUDGE_BACKEND="${JUDGE_BACKEND:-heuristic}"
  python3 "$ROOT/experiments/judge_service/server.py" --host 127.0.0.1 --port "$JUDGE_PORT" &
  JUDGE_PID=$!
  for _ in $(seq 1 40); do
    curl -sf "http://127.0.0.1:${JUDGE_PORT}/health" >/dev/null 2>&1 && break
    sleep 0.15
  done
  export PAPER_EVAL_JUDGE_URL="http://127.0.0.1:${JUDGE_PORT}/judge"
  echo "[smoke] started local judge_service backend=$JUDGE_BACKEND url=$PAPER_EVAL_JUDGE_URL"
fi

DEFS="unified,direct_upstream,smooth_llm"
SCENARIOS="benign_baseline,refusal_keyword,extraction_leak,multi_round_extraction,harmful_rsr_suite,benign_fpr_suite"
TRACKA_JUDGE_MODE="${TRACKA_JUDGE_MODE:-http}"

JUDGE_ARGS=(--judge-mode "$TRACKA_JUDGE_MODE")
if [[ "${TRACKA_JUDGE_MODE}" == "http" ]]; then
  JUDGE_ARGS+=(--judge-url "$PAPER_EVAL_JUDGE_URL")
fi

echo "[smoke] out=$SMOKE_OUT defenses=$DEFS prompt_workers=$SMOKE_PROMPT_WORKERS"
python3 experiments/run_paper_benchmark.py \
  --gateway-url "$GATEWAY_URL" \
  --upstream-url "$UP" \
  --defenses "$DEFS" \
  --suite minimal \
  --scenarios "$SCENARIOS" \
  --harmful-prompts-file "$HARMFUL" \
  --benign-prompts-file "$BENIGN" \
  --max-harmful-prompts "$MAX_HARMFUL_PROMPTS" \
  --max-benign-fpr-prompts "$MAX_BENIGN_FPR_PROMPTS" \
  --max-rounds "$SMOKE_MAX_ROUNDS" \
  --seeds 42 \
  --prompt-workers "$SMOKE_PROMPT_WORKERS" \
  "${JUDGE_ARGS[@]}" \
  --openai-model "$MODEL" \
  -o "$SMOKE_OUT"

echo "[smoke] wrote $SMOKE_OUT"
python3 experiments/scripts/validate_paper_json.py "$SMOKE_OUT" || true
echo "[smoke] manifest.judge_mode=$(python3 -c "import json;print(json.load(open('$SMOKE_OUT'))['manifest'].get('judge_mode'))") prompt_workers=$(python3 -c "import json;print(json.load(open('$SMOKE_OUT'))['manifest'].get('prompt_workers'))")"
