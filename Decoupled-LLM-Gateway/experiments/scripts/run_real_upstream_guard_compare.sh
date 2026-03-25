#!/usr/bin/env bash
# Two-run smoke on a REAL upstream: same scenarios, with vs without --gateway-output-guard.
# Requires: judge_service + gateway already running (see RUN_GUIDE §2.8); API key in env.
#
# Env overrides:
#   GATEWAY_URL              default http://127.0.0.1:8080
#   UPSTREAM_URL             default https://api.deepseek.com
#   OPENAI_MODEL             default deepseek-chat
#   MAX_HARMFUL MAX_BENIGN_FPR  caps for harmful_rsr_suite / benign_fpr_suite (default 3 / 4)
#
# Usage:
#   export DEEPSEEK_API_KEY='...'
#   bash experiments/scripts/run_real_upstream_guard_compare.sh

set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

if [[ -z "${DEEPSEEK_API_KEY:-}" && -z "${GATEWAY_UPSTREAM_API_KEY:-}" && -z "${OPENAI_API_KEY:-}" ]]; then
  echo "run_real_upstream_guard_compare: set DEEPSEEK_API_KEY, GATEWAY_UPSTREAM_API_KEY, or OPENAI_API_KEY" >&2
  exit 1
fi

GATEWAY_URL="${GATEWAY_URL:-http://127.0.0.1:8080}"
UPSTREAM_URL="${UPSTREAM_URL:-https://api.deepseek.com}"
OPENAI_MODEL="${OPENAI_MODEL:-deepseek-chat}"
MAX_H="${MAX_HARMFUL:-3}"
MAX_B="${MAX_BENIGN_FPR:-4}"

mkdir -p results
OUT_NO="results/real_smoke_noguard.json"
OUT_YES="results/real_smoke_guard.json"

common=(
  python3 experiments/run_paper_benchmark.py
  --gateway-url "$GATEWAY_URL"
  --upstream-url "$UPSTREAM_URL"
  --openai-model "$OPENAI_MODEL"
  --defenses unified,no_obfuscate,no_decoy,intent_only
  --scenarios harmful_rsr_suite,benign_fpr_suite
  --max-harmful-prompts "$MAX_H"
  --max-benign-fpr-prompts "$MAX_B"
  --seed 42
)

echo "Writing $OUT_NO (no --gateway-output-guard)..." >&2
"${common[@]}" -o "$OUT_NO"

echo "Writing $OUT_YES (--gateway-output-guard)..." >&2
"${common[@]}" --gateway-output-guard -o "$OUT_YES"

echo "=== jq: manifest (expect gateway_output_guard_header false vs true) ===" >&2
jq '.manifest | {gateway_output_guard_header, max_harmful_prompts, max_benign_fpr_prompts, protocol_version, openai_model_field}' "$OUT_NO" >&2
jq '.manifest | {gateway_output_guard_header, max_harmful_prompts, max_benign_fpr_prompts, protocol_version, openai_model_field}' "$OUT_YES" >&2

echo "=== diff manifests (excluding git_sha/hostname noise) ===" >&2
diff -u \
  <(jq '.manifest | del(.git_sha, .hostname)' "$OUT_NO") \
  <(jq '.manifest | del(.git_sha, .hostname)' "$OUT_YES") \
  || true

echo "Done. Inspect per_prompt in harmful_rsr_suite / benign_fpr_suite for subjective deltas." >&2
