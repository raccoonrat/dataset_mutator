#!/usr/bin/env bash
# Download JBB-Behaviors, StrongREJECT, and a small in-the-wild sample for paper-eval-5 (Update_dataset_Guide).
# Requires: pip install -r experiments/requirements-benchmark.txt (for JBB via Hugging Face).
#
# Usage (from Decoupled-LLM-Gateway/):
#   bash experiments/scripts/fetch_all_paper_datasets.sh

set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

if ! python3 -c "import datasets" 2>/dev/null; then
  echo "Installing benchmark dataset deps (datasets, pandas)..." >&2
  pip install -q -r experiments/requirements-benchmark.txt
fi

python3 experiments/scripts/fetch_jbb_behaviors.py \
  --misuse-out experiments/data/jbb_misuse_100_en.txt \
  --benign-out experiments/data/jbb_benign_100_en.txt

python3 experiments/scripts/fetch_strongreject_prompts.py \
  -o experiments/data/strongreject_forbidden_en.txt

python3 experiments/scripts/fetch_wild_jailbreak_sample.py \
  -n "${WILD_SAMPLE_N:-200}" \
  -o experiments/data/wild_jailbreak_first200_en.txt

echo "OK: paper datasets materialized under experiments/data/" >&2
