#!/usr/bin/env bash
# Shared helpers for paper Track A shell scripts (sourced, not executed).
# Usage: ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)" && cd "$ROOT" && source experiments/scripts/paper_common.sh

paper_source_env_if_present() {
  if [[ -f ./env ]]; then
    set -a
    # shellcheck source=/dev/null
    . ./env
    set +a
  fi
}

# Hugging Face Hub / datasets / huggingface.co: use SOCKS5 (urllib3+PySocks via requests).
# Does nothing if HTTPS_PROXY or ALL_PROXY is already set.
#
# Set one of (first non-empty): TRACKA_HF_SOCKS5, HF_SOCKS5_PROXY, or reuse JUDGE_SOCKS5_PROXY.
# Value: host:port (e.g. 127.0.0.1:1080) or full URI socks5h://127.0.0.1:1080
#
# NO_PROXY / no_proxy extended with loopback so local GATEWAY_URL / judge skip the proxy.
paper_apply_hf_proxy_env() {
  if [[ -n "${HTTPS_PROXY:-}" || -n "${ALL_PROXY:-}" ]]; then
    return 0
  fi
  local raw="${TRACKA_HF_SOCKS5:-${HF_SOCKS5_PROXY:-${JUDGE_SOCKS5_PROXY:-}}}"
  if [[ -z "$raw" ]]; then
    return 0
  fi
  local url="$raw"
  if [[ "$raw" != *"://"* ]]; then
    url="socks5h://${raw}"
  fi
  export HTTPS_PROXY="$url"
  export HTTP_PROXY="$url"
  export ALL_PROXY="$url"
  local loop="127.0.0.1,localhost,::1"
  if [[ -n "${NO_PROXY:-}" ]]; then
    export NO_PROXY="${NO_PROXY},${loop}"
  else
    export NO_PROXY="$loop"
  fi
  export no_proxy="$NO_PROXY"
  echo "[paper] Hugging Face / HTTPS via ${url} (NO_PROXY includes ${loop})" >&2
}

# Args: GATEWAY_URL MODEL — echo response must not contain [echo] marker from echo-llm.
paper_preflight_non_echo() {
  local body resp
  body=$(printf '%s' "{\"model\":\"${MODEL}\",\"messages\":[{\"role\":\"user\",\"content\":\"Reply with exactly: OK\"}]}")
  if ! resp=$(curl -sS -m 180 "${GATEWAY_URL}/v1/chat/completions" \
    -H 'Content-Type: application/json' -d "$body"); then
    echo "PREFLIGHT FAIL: cannot reach ${GATEWAY_URL}/v1/chat/completions (start gateway with . ./env?)" >&2
    return 1
  fi
  if echo "$resp" | grep -q '\[echo\]'; then
    echo "PREFLIGHT FAIL: gateway still returns [echo]; set GATEWAY_UPSTREAM to real API and restart gateway." >&2
    return 1
  fi
  echo "PREFLIGHT OK (non-echo gateway)."
  if [[ "${PAPER_PREFLIGHT_SHOW_SAMPLE:-1}" == "1" ]]; then
    echo "PREFLIGHT sample (first ~120 chars):"
    echo "$resp" | head -c 120
    echo "..."
  fi
  return 0
}

paper_refuse_echo_upstream_url() {
  local up="$1"
  case "$up" in
    *127.0.0.1:9090*|*localhost:9090*|:9090*)
      echo "REFUSE: upstream $up looks like echo-llm." >&2
      return 1
      ;;
  esac
  return 0
}
