#!/usr/bin/env bash
# Start judge_service with OpenRouter + Llama Guard 4 (12B). Loads OPENROUTER_API_KEY from repo env file.
# Usage:
#   bash experiments/scripts/run_openrouter_llama_guard4_judge.sh
# Then in another terminal:
#   export PAPER_EVAL_JUDGE_URL=http://127.0.0.1:8765/judge
#   export JUDGE_MODEL_REVISION=meta-llama/llama-guard-4-12b   # optional explicit revision string for manifest
#   python3 experiments/scripts/relabel_tracka_llama_guard.py --json results/your.json --out results/your_lg4.json

set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"
if [[ -f "$ROOT/env" ]]; then
  set -a
  # shellcheck disable=SC1091
  source "$ROOT/env"
  set +a
fi

export JUDGE_BACKEND=chat_completion
export JUDGE_CHAT_BASE_URL="${JUDGE_CHAT_BASE_URL:-https://openrouter.ai/api/v1}"
export JUDGE_CHAT_MODEL="${JUDGE_CHAT_MODEL:-meta-llama/llama-guard-4-12b}"
export JUDGE_LLAMA_GUARD_NATIVE="${JUDGE_LLAMA_GUARD_NATIVE:-1}"
export JUDGE_CHAT_MAX_TOKENS="${JUDGE_CHAT_MAX_TOKENS:-64}"
# Prefer OpenRouter key from env file; fall back to JUDGE_CHAT_API_KEY if set
if [[ -z "${JUDGE_CHAT_API_KEY:-}" ]]; then
  export JUDGE_CHAT_API_KEY="${OPENROUTER_API_KEY:-}"
fi
export JUDGE_OPENROUTER_HTTP_REFERER="${JUDGE_OPENROUTER_HTTP_REFERER:-https://github.com/raccoonrat/dataset_mutator}"
export JUDGE_OPENROUTER_X_TITLE="${JUDGE_OPENROUTER_X_TITLE:-Decoupled-LLM-Gateway paper-eval}"
# urllib 不走 SOCKS；OpenRouter 经本地 SOCKS5 时用（可覆盖或置空禁用）
export JUDGE_SOCKS5_PROXY="${JUDGE_SOCKS5_PROXY:-127.0.0.1:1080}"

# Same python3 as server.py must have PySocks + requests; pip must not use SOCKS (chicken-and-egg).
judge_ensure_socks_deps() {
  if [[ -z "${JUDGE_SOCKS5_PROXY:-}" ]]; then
    return 0
  fi
  if python3 -c "import socks, requests" 2>/dev/null; then
    return 0
  fi
  echo "[judge] installing PySocks+requests for SOCKS (clearing proxy env for pip only)..." >&2
  env -u HTTPS_PROXY -u HTTP_PROXY -u ALL_PROXY -u http_proxy -u https_proxy -u all_proxy \
    pip install -q -r "$ROOT/experiments/judge_service/requirements.txt"
}
judge_ensure_socks_deps

exec python3 experiments/judge_service/server.py --host 127.0.0.1 --port "${JUDGE_PORT:-8765}"
