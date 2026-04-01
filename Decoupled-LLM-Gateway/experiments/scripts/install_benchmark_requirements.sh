#!/usr/bin/env bash
# Install experiments/requirements-benchmark.txt without HTTP(S)/ALL proxy env vars.
#
# If HTTPS_PROXY/ALL_PROXY is socks5h://..., pip uses it to reach PyPI before PySocks is
# installed → OSError: Missing dependencies for SOCKS support. This script clears those
# variables for the pip subprocess only (your shell keeps them for HF / judge).
#
# Usage (from Decoupled-LLM-Gateway/):
#   bash experiments/scripts/install_benchmark_requirements.sh
#   bash experiments/scripts/install_benchmark_requirements.sh --user

set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

exec env -u HTTPS_PROXY -u HTTP_PROXY -u ALL_PROXY -u http_proxy -u https_proxy -u all_proxy \
  pip install -r experiments/requirements-benchmark.txt "$@"
