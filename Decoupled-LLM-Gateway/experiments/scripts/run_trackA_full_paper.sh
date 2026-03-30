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
#   RESUME               set to 1 to pass --resume; you MUST also set OUT to the *same* path as the interrupted run
#                        (checkpoint lives at <OUT-stem>.checkpoint.json next to OUT)
#   CHECKPOINT           optional explicit path for --checkpoint (default is next to OUT)
#   OPENAI_MODEL         default deepseek-chat
#   SKIP_PREFLIGHT       set to 1 to skip gateway probe (not recommended for real-paper runs)
#   SKIP_FETCH_DATASETS  set to 1 to skip HuggingFace/network dataset fetch (use existing files)
#   HARMFUL_PROMPTS_FILE / BENIGN_PROMPTS_FILE  override default JBB paths
#
# Local secrets / upstream (optional): place a file ./env in this repo root and run
#   cd Decoupled-LLM-Gateway && . ./env && ./experiments/scripts/run_trackA_full_paper.sh
# This script auto-sources ./env when present (exports all variables defined there).

set -euo pipefail
export PYTHONUNBUFFERED=1
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"
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
  --judge-mode heuristic \
  --openai-model "$MODEL" \
  "${CK_ARGS[@]}" \
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
