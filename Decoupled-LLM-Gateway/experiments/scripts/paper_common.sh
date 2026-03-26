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
