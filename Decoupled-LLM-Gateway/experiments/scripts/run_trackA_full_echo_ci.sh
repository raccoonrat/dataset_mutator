#!/usr/bin/env bash
# Start echo-llm + gateway (upstream=echo), run full matrix → results/trackA_full_echo_seed3.json
# For CI / no API key. When PAPER_OUT_FALLBACK=1 (default) and no API key, also writes
# results/trackA_full_paper_seed3.json with manifest.repro_note (same numeric run as echo).

set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

ECHO_PORT="${ECHO_PORT:-9090}"
GW_PORT="${GW_PORT:-8080}"
GATEWAY_URL="http://127.0.0.1:${GW_PORT}"
UPSTREAM_URL="http://127.0.0.1:${ECHO_PORT}"
OUT_ECHO="${OUT_ECHO:-results/trackA_full_echo_seed3.json}"
OUT_PAPER="${OUT_PAPER:-results/trackA_full_paper_seed3.json}"
# Default 0: do not overwrite results/trackA_full_paper_seed3.json (real DeepSeek runs).
# Set PAPER_OUT_FALLBACK=1 to copy echo JSON there with repro_note when no API key (CI only).
PAPER_OUT_FALLBACK="${PAPER_OUT_FALLBACK:-0}"

cleanup() {
  [[ -n "${GW_PID:-}" ]] && kill "$GW_PID" 2>/dev/null || true
  [[ -n "${ECHO_PID:-}" ]] && kill "$ECHO_PID" 2>/dev/null || true
}
trap cleanup EXIT

export ECHO_LISTEN=":${ECHO_PORT}"
go run ./cmd/echo-llm &
ECHO_PID=$!
for _ in $(seq 1 30); do
  curl -sf "http://127.0.0.1:${ECHO_PORT}/healthz" >/dev/null 2>&1 && break
  sleep 0.2
done

export GATEWAY_UPSTREAM="$UPSTREAM_URL"
export GATEWAY_LISTEN=":${GW_PORT}"
export GATEWAY_ASYNC_LOG="${GATEWAY_ASYNC_LOG:-0}"
go run ./cmd/gateway &
GW_PID=$!
sleep 1.2

DEFS="unified,no_obfuscate,no_decoy,intent_only,direct_upstream,smooth_llm,structured_wrap,strong_system_guard,rag_semantic_only"

python3 experiments/run_paper_benchmark.py \
  --gateway-url "$GATEWAY_URL" \
  --upstream-url "$UPSTREAM_URL" \
  --defenses "$DEFS" \
  --suite full \
  --seeds 42,43,44 \
  --judge-mode heuristic \
  --openai-model echo-mock \
  -o "$OUT_ECHO"

echo "Wrote $OUT_ECHO"

if [[ "$PAPER_OUT_FALLBACK" == "1" ]] && [[ -z "${DEEPSEEK_API_KEY:-}" ]] && [[ -z "${OPENAI_API_KEY:-}" ]]; then
  python3 experiments/scripts/annotate_paper_manifest_note.py \
    --src "$OUT_ECHO" \
    --dst "$OUT_PAPER" \
    --note "Generated without DEEPSEEK_API_KEY/OPENAI_API_KEY: upstream is echo-llm (identical run to ${OUT_ECHO}). For production-aligned Track A numbers, start the gateway with GATEWAY_UPSTREAM pointing to the real API and run experiments/scripts/run_trackA_full_paper.sh."
  echo "Wrote $OUT_PAPER (fallback with repro_note)"
fi
