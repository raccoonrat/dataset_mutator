#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

export GATEWAY_UPSTREAM="${GATEWAY_UPSTREAM:-http://127.0.0.1:9090}"

cleanup() {
  kill "${ECHO_PID:-0}" 2>/dev/null || true
}
trap cleanup EXIT

go run ./cmd/echo-llm &
ECHO_PID=$!

for i in $(seq 1 50); do
  if curl -sf "http://127.0.0.1:9090/healthz" >/dev/null; then
    break
  fi
  sleep 0.05
done

exec go run ./cmd/gateway
