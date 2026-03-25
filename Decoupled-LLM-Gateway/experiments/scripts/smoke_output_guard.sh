#!/usr/bin/env bash
# End-to-end smoke: echo-llm + heuristic judge_service + gateway output guard.
#
# - Without X-Gateway-Output-Guard: upstream echo text is returned (guard not run).
# - With X-Gateway-Output-Guard: 1: non-refusal echo text is replaced by template.
# - With guard header + echo refusal path: refusal text is kept (judge says is_refusal).
#
# Binds three ephemeral TCP ports (or set SMOKE_ECHO_PORT / SMOKE_JUDGE_PORT / SMOKE_GW_PORT).
#
# Usage:
#   bash experiments/scripts/smoke_output_guard.sh
#   # or from repo root:
#   make smoke-output-guard

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

free_tcp_port() {
  python3 -c "import socket;s=socket.socket();s.bind(('127.0.0.1',0));print(s.getsockname()[1]);s.close()"
}

# Align with Makefile: some environments export empty GOPATH/GOMODCACHE and break `go run`.
if [[ -z "${GOMODCACHE:-}" ]]; then
  if [[ -n "${GOPATH:-}" ]]; then
    export GOMODCACHE="${GOPATH}/pkg/mod"
  else
    export GOMODCACHE="${HOME}/go/pkg/mod"
  fi
fi
if [[ -z "${GOCACHE:-}" ]]; then
  export GOCACHE="${HOME}/.cache/go-build"
fi

# Default: three ephemeral ports (override with SMOKE_ECHO_PORT / SMOKE_JUDGE_PORT / SMOKE_GW_PORT).
ECHO_PORT="${SMOKE_ECHO_PORT:-$(free_tcp_port)}"
JUDGE_PORT="${SMOKE_JUDGE_PORT:-$(free_tcp_port)}"
GW_PORT="${SMOKE_GW_PORT:-$(free_tcp_port)}"
GUARD_TEMPLATE="${SMOKE_GUARD_TEMPLATE:-[GUARDED-BY-SMOKE]}"

ECHO_URL="http://127.0.0.1:${ECHO_PORT}"
JUDGE_URL="http://127.0.0.1:${JUDGE_PORT}/judge"
GW_URL="http://127.0.0.1:${GW_PORT}"

cleanup() {
  [[ -n "${GW_PID:-}" ]] && kill "$GW_PID" 2>/dev/null || true
  [[ -n "${JUDGE_PID:-}" ]] && kill "$JUDGE_PID" 2>/dev/null || true
  [[ -n "${ECHO_PID:-}" ]] && kill "$ECHO_PID" 2>/dev/null || true
}
trap cleanup EXIT

export ECHO_LISTEN="127.0.0.1:${ECHO_PORT}"
go run ./cmd/echo-llm &
ECHO_PID=$!

for _ in $(seq 1 40); do
  curl -sf "${ECHO_URL}/healthz" >/dev/null && break
  sleep 0.15
done
curl -sf "${ECHO_URL}/healthz" >/dev/null || {
  echo "smoke_output_guard: echo did not become ready" >&2
  exit 1
}

JUDGE_BACKEND=heuristic python3 experiments/judge_service/server.py --host 127.0.0.1 --port "$JUDGE_PORT" &
JUDGE_PID=$!

for _ in $(seq 1 40); do
  curl -sf "http://127.0.0.1:${JUDGE_PORT}/health" >/dev/null && break
  sleep 0.15
done
curl -sf "http://127.0.0.1:${JUDGE_PORT}/health" >/dev/null || {
  echo "smoke_output_guard: judge did not become ready" >&2
  exit 1
}

export GATEWAY_LISTEN="127.0.0.1:${GW_PORT}"
export GATEWAY_UPSTREAM="${ECHO_URL}"
export GATEWAY_OUTPUT_GUARD_URL="${JUDGE_URL}"
export GATEWAY_OUTPUT_GUARD_TEMPLATE="${GUARD_TEMPLATE}"
export GATEWAY_OUTPUT_GUARD_FAIL_OPEN="1"
export GATEWAY_ASYNC_LOG="0"

go run ./cmd/gateway &
GW_PID=$!

for _ in $(seq 1 40); do
  curl -sf -o /dev/null -X POST "${GW_URL}/v1/chat/completions" \
    -H 'Content-Type: application/json' \
    -d '{"model":"eval","messages":[{"role":"user","content":"ping"}]}' 2>/dev/null && break
  sleep 0.15
done

json_body='{"model":"eval","messages":[{"role":"user","content":"hello smoke"}]}'

# 1) No guard header → echo reply unchanged
out1=$(curl -sS "${GW_URL}/v1/chat/completions" \
  -H 'Content-Type: application/json' \
  -d "$json_body")
c1=$(echo "$out1" | python3 -c 'import json,sys; print(json.load(sys.stdin)["choices"][0]["message"]["content"])')
if [[ "$c1" != *"[echo] hello smoke"* ]]; then
  echo "FAIL: expected echo content without guard header, got: $c1" >&2
  exit 1
fi

# 2) Guard header → non-refusal replaced by template
out2=$(curl -sS "${GW_URL}/v1/chat/completions" \
  -H 'Content-Type: application/json' \
  -H 'X-Gateway-Output-Guard: 1' \
  -d "$json_body")
c2=$(echo "$out2" | python3 -c 'import json,sys; print(json.load(sys.stdin)["choices"][0]["message"]["content"])')
if [[ "$c2" != "$GUARD_TEMPLATE" ]]; then
  echo "FAIL: expected guard template with header, got: $c2" >&2
  exit 1
fi

# 3) Guard header + echo refusal → keep refusal (judge: is_refusal true)
json_refuse='{"model":"eval","messages":[{"role":"user","content":"MALICIOUS_TRIGGER payload"}]}'
out3=$(curl -sS "${GW_URL}/v1/chat/completions" \
  -H 'Content-Type: application/json' \
  -H 'X-Gateway-Output-Guard: 1' \
  -H 'X-Echo-Refuse-Substr: MALICIOUS_TRIGGER' \
  -d "$json_refuse")
c3=$(echo "$out3" | python3 -c 'import json,sys; print(json.load(sys.stdin)["choices"][0]["message"]["content"])')
if [[ "$c3" != *"REFUSAL_SINGLE_GUARD"* ]]; then
  echo "FAIL: expected echo refusal to bypass template, got: $c3" >&2
  exit 1
fi

echo "smoke_output_guard: ok (no-header echo, header→template, header+refuse→echo refusal)"
echo "  gateway ${GW_URL}  echo ${ECHO_URL}  judge ${JUDGE_URL}"
