#!/usr/bin/env bash
# Full Track A matrix (paper-eval-5): JBB-Behaviors misuse/benign (paired FPR) + optional wild + StrongREJECT;
# 9 defenses × full scenario suite × 3 seeds, no prompt caps on core suites.
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
#   OUT                  default results/trackA_full_paper_paper_eval5_<YYYYMMDD_HHMMSS>.json (unique per run)
#   TRACKA_LATEST_LINK   default results/trackA_full_paper_latest.json — symlink updated after a successful benchmark run
#   TRACKA_SKIP_LATEST_SYMLINK  set to 1 to skip creating/updating the symlink
#   RESUME               set to 1 to pass --resume; you MUST also set OUT to the *same* path as the interrupted run
#                        (checkpoint lives at <OUT-stem>.checkpoint.json next to OUT)
#   CHECKPOINT           optional explicit path for --checkpoint (default is next to OUT)
#   OPENAI_MODEL         default deepseek-chat
#   SKIP_PREFLIGHT       set to 1 to skip gateway probe (not recommended for real-paper runs)
#   SKIP_FETCH_DATASETS  set to 1 to skip HuggingFace/network dataset fetch (use existing files)
#   HARMFUL_PROMPTS_FILE / BENIGN_PROMPTS_FILE  override default JBB paths
#   SKIP_CONDA_ACTIVATE  set to 1 to skip auto conda activate dataset_mutator
#   TRACKA_PROGRESS_INTERVAL_SEC  seconds between [trackA] progress lines (default 300); 0 disables periodic logs
#
# Refusal judge (主表 LG4 / HTTP): 默认 --judge-mode http，需先启动 judge（见 scripts/run_openrouter_llama_guard4_judge.sh）。
#   PAPER_EVAL_JUDGE_URL  默认 http://127.0.0.1:${JUDGE_PORT:-8765}/judge
#   TRACKA_JUDGE_MODE     设为 heuristic 可恢复纯关键词裁判（不写 manifest LG4）
#   跑分前在**同一 shell** export JUDGE_* / JUDGE_MODEL_REVISION，manifest.eval_judge_chat 会记录 revision
#
# Local secrets / upstream (optional): place a file ./env in this repo root and run
#   cd Decoupled-LLM-Gateway && . ./env && ./experiments/scripts/run_trackA_full_paper.sh
# This script auto-sources ./env when present (exports all variables defined there).

set -euo pipefail
export PYTHONUNBUFFERED=1
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

paper_activate_dataset_mutator() {
  if [[ "${SKIP_CONDA_ACTIVATE:-0}" == "1" ]]; then
    return 0
  fi
  if [[ "${CONDA_DEFAULT_ENV:-}" == "dataset_mutator" ]]; then
    echo "[trackA] conda: already active dataset_mutator python=$(command -v python3)"
    return 0
  fi
  local base sh
  base=""
  if command -v conda >/dev/null 2>&1; then
    base="$(conda info --base 2>/dev/null)" || true
  fi
  if [[ -z "$base" && -f "${HOME}/anaconda3/etc/profile.d/conda.sh" ]]; then
    base="${HOME}/anaconda3"
  fi
  if [[ -z "$base" && -f "${HOME}/miniconda3/etc/profile.d/conda.sh" ]]; then
    base="${HOME}/miniconda3"
  fi
  sh="${base}/etc/profile.d/conda.sh"
  if [[ ! -f "$sh" ]]; then
    echo "ERROR: need conda env dataset_mutator; conda.sh not found. Run: conda activate dataset_mutator" >&2
    exit 1
  fi
  # shellcheck source=/dev/null
  source "$sh"
  conda activate dataset_mutator || {
    echo "ERROR: conda activate dataset_mutator failed" >&2
    exit 1
  }
  echo "[trackA] conda: activated dataset_mutator python=$(command -v python3)"
}

paper_activate_dataset_mutator

# shellcheck source=/dev/null
source "$ROOT/experiments/scripts/paper_common.sh"
paper_source_env_if_present
GATEWAY_URL="${GATEWAY_URL:-http://127.0.0.1:8080}"
UP="${GATEWAY_UPSTREAM:-https://api.deepseek.com}"
RUN_TS="$(date +%Y%m%d_%H%M%S)"
if [[ "${RESUME:-0}" == "1" ]]; then
  if [[ -z "${OUT:-}" ]]; then
    echo "RESUME=1 requires OUT=results/....json pointing at the same file as the first run." >&2
    exit 1
  fi
else
  OUT="${OUT:-results/trackA_full_paper_paper_eval5_${RUN_TS}.json}"
fi
MODEL="${OPENAI_MODEL:-deepseek-chat}"
SKIP_PREFLIGHT="${SKIP_PREFLIGHT:-0}"

paper_refuse_echo_upstream_url "$UP" || exit 1

if [[ "${SKIP_FETCH_DATASETS:-0}" != "1" ]]; then
  if ! python3 -c "import datasets" 2>/dev/null; then
    pip install -q -r experiments/requirements-benchmark.txt || {
      echo "WARN: could not pip install datasets; run: bash experiments/scripts/fetch_all_paper_datasets.sh" >&2
    }
  fi
  python3 experiments/scripts/fetch_jbb_behaviors.py \
    --misuse-out "$ROOT/experiments/data/jbb_misuse_100_en.txt" \
    --benign-out "$ROOT/experiments/data/jbb_benign_100_en.txt" || {
    echo "WARN: JBB fetch failed; falling back to repo harmful/benign lists if present." >&2
  }
  python3 experiments/scripts/fetch_strongreject_prompts.py -o "$ROOT/experiments/data/strongreject_forbidden_en.txt" || true
  python3 experiments/scripts/fetch_wild_jailbreak_sample.py -n "${WILD_SAMPLE_N:-200}" \
    -o "$ROOT/experiments/data/wild_jailbreak_first200_en.txt" || true
fi

HARMFUL="${HARMFUL_PROMPTS_FILE:-$ROOT/experiments/data/jbb_misuse_100_en.txt}"
BENIGN="${BENIGN_PROMPTS_FILE:-$ROOT/experiments/data/jbb_benign_100_en.txt}"
if [[ ! -f "$HARMFUL" ]]; then HARMFUL="$ROOT/experiments/data/harmful_prompts_trackA_en.txt"; fi
if [[ ! -f "$BENIGN" ]]; then BENIGN="$ROOT/experiments/data/benign_prompts_en.txt"; fi

SCENARIOS="benign_baseline,refusal_keyword,extraction_leak,multi_round_extraction,harmful_rsr_suite,hpm_proxy,benign_fpr_suite,decoy_dos_sla"
if [[ -f "$ROOT/experiments/data/wild_jailbreak_first200_en.txt" ]]; then
  SCENARIOS="${SCENARIOS},wild_rsr_suite"
fi
if [[ -f "$ROOT/experiments/data/strongreject_forbidden_en.txt" ]]; then
  SCENARIOS="${SCENARIOS},strongreject_rsr_suite"
fi

if [[ "$SKIP_PREFLIGHT" != "1" ]]; then
  export PAPER_PREFLIGHT_SHOW_SAMPLE=1
  paper_preflight_non_echo || exit 1
fi

DEFS="unified,no_obfuscate,no_decoy,intent_only,direct_upstream,smooth_llm,structured_wrap,strong_system_guard,rag_semantic_only"

CK_ARGS=()
if [[ "${RESUME:-0}" == "1" ]]; then
  CK_ARGS+=(--resume)
fi
if [[ -n "${CHECKPOINT:-}" ]]; then
  CK_ARGS+=(--checkpoint "$CHECKPOINT")
fi

PROGRESS_SEC="${TRACKA_PROGRESS_INTERVAL_SEC:-300}"

paper_core_progress_line() {
  local pid="$1" bench_start="$2" tag="$3"
  local now es h m s pe peh pem pes
  now=$(date +%s)
  es=$((now - bench_start))
  h=$((es / 3600))
  m=$(((es % 3600) / 60))
  s=$((es % 60))
  printf '[trackA] %s %s core_pid=%s script_elapsed_s=%d script_elapsed_hms=%02d:%02d:%02d' \
    "$(date -Is)" "$tag" "${pid:-?}" "$es" "$h" "$m" "$s"
  if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
    pe=$(ps -o etimes= -p "$pid" 2>/dev/null | tr -d ' ' || true)
    if [[ -n "$pe" && "$pe" =~ ^[0-9]+$ ]]; then
      peh=$((pe / 3600))
      pem=$(((pe % 3600) / 60))
      pes=$((pe % 60))
      printf ' process_elapsed_s=%s process_elapsed_hms=%02d:%02d:%02d' "$pe" "$peh" "$pem" "$pes"
    fi
  fi
  printf '\n'
}

JUDGE_PORT="${JUDGE_PORT:-8765}"
TRACKA_JUDGE_MODE="${TRACKA_JUDGE_MODE:-http}"
PAPER_EVAL_JUDGE_URL="${PAPER_EVAL_JUDGE_URL:-http://127.0.0.1:${JUDGE_PORT}/judge}"
export PAPER_EVAL_JUDGE_URL

JUDGE_ARGS=(--judge-mode "$TRACKA_JUDGE_MODE")
if [[ "${TRACKA_JUDGE_MODE}" == "http" ]]; then
  JUDGE_ARGS+=(--judge-url "$PAPER_EVAL_JUDGE_URL")
fi

echo "[trackA] $(date -Is) benchmark_main_start out=$OUT judge_mode=${TRACKA_JUDGE_MODE} judge_url=${PAPER_EVAL_JUDGE_URL:-}"
BENCH_START=$(date +%s)
python3 experiments/run_paper_benchmark.py \
  --gateway-url "$GATEWAY_URL" \
  --upstream-url "$UP" \
  --defenses "$DEFS" \
  --suite minimal \
  --scenarios "$SCENARIOS" \
  --harmful-prompts-file "$HARMFUL" \
  --benign-prompts-file "$BENIGN" \
  --wild-prompts-file "$ROOT/experiments/data/wild_jailbreak_first200_en.txt" \
  --strongreject-prompts-file "$ROOT/experiments/data/strongreject_forbidden_en.txt" \
  --dataset-profile jbb_paired_sota \
  --max-wild-prompts "${MAX_WILD_PROMPTS:-200}" \
  --max-strongreject-prompts "${MAX_STRONGREJECT_PROMPTS:-100}" \
  --seeds 42,43,44 \
  "${JUDGE_ARGS[@]}" \
  --openai-model "$MODEL" \
  "${CK_ARGS[@]}" \
  -o "$OUT" &
BENCH_PID=$!

if [[ "$PROGRESS_SEC" =~ ^[0-9]+$ ]] && [[ "$PROGRESS_SEC" -gt 0 ]]; then
  (
    while kill -0 "$BENCH_PID" 2>/dev/null; do
      sleep "$PROGRESS_SEC"
      kill -0 "$BENCH_PID" 2>/dev/null || break
      paper_core_progress_line "$BENCH_PID" "$BENCH_START" "core_progress"
    done
  ) &
  PROG_PID=$!
else
  PROG_PID=""
fi

set +e
wait "$BENCH_PID"
BENCH_EXIT=$?
set -e

if [[ -n "${PROG_PID:-}" ]]; then
  kill "$PROG_PID" 2>/dev/null || true
  wait "$PROG_PID" 2>/dev/null || true
fi

now=$(date +%s)
es=$((now - BENCH_START))
h=$((es / 3600))
m=$(((es % 3600) / 60))
s=$((es % 60))
printf '[trackA] %s benchmark_done core_pid=%s exit=%d script_elapsed_s=%d script_elapsed_hms=%02d:%02d:%02d\n' \
  "$(date -Is)" "$BENCH_PID" "$BENCH_EXIT" "$es" "$h" "$m" "$s"

if [[ "$BENCH_EXIT" -ne 0 ]]; then
  exit "$BENCH_EXIT"
fi

echo "Wrote $OUT"

# Stable alias: fixed path -> current timestamped artifact (absolute target so cwd-independent).
if [[ "${TRACKA_SKIP_LATEST_SYMLINK:-0}" != "1" ]]; then
  LATEST_REL="${TRACKA_LATEST_LINK:-results/trackA_full_paper_latest.json}"
  if [[ "$LATEST_REL" != /* ]]; then
    LATEST_ABS="$ROOT/$LATEST_REL"
  else
    LATEST_ABS="$LATEST_REL"
  fi
  OUT_DIR="$(cd "$(dirname "$OUT")" && pwd)"
  ABS_OUT="$OUT_DIR/$(basename "$OUT")"
  mkdir -p "$(dirname "$LATEST_ABS")"
  ln -sf "$ABS_OUT" "$LATEST_ABS"
  echo "Symlink $LATEST_ABS -> $ABS_OUT"
fi

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
