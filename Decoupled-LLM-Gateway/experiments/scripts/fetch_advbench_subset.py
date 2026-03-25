#!/usr/bin/env python3
"""
Fetch the first N goals from the public AdvBench harmful_behaviors CSV for paper-eval RSR suites.

Source: llm-attacks/llm-attacks (harmful_behaviors.csv). One goal per line written to stdout or -o.

Example:
  python3 experiments/scripts/fetch_advbench_subset.py -n 80 -o experiments/data/harmful_prompts_advbench_first80.txt
"""

from __future__ import annotations

import argparse
import csv
import io
import sys
import urllib.request

DEFAULT_URL = (
    "https://raw.githubusercontent.com/llm-attacks/llm-attacks/main/data/advbench/harmful_behaviors.csv"
)


def main() -> None:
    ap = argparse.ArgumentParser(description="Download AdvBench harmful_behaviors subset (goals column)")
    ap.add_argument("-n", type=int, default=80, help="number of goals (after header)")
    ap.add_argument("--url", default=DEFAULT_URL, help="CSV URL")
    ap.add_argument("-o", "--output", help="write file (UTF-8); default stdout")
    args = ap.parse_args()

    with urllib.request.urlopen(args.url, timeout=60) as resp:
        raw = resp.read().decode("utf-8", errors="replace")

    r = csv.DictReader(io.StringIO(raw))
    lines: list[str] = []
    for row in r:
        g = (row.get("goal") or row.get("Goal") or "").strip()
        if not g:
            continue
        lines.append(g)
        if len(lines) >= args.n:
            break

    header = (
        f"# AdvBench harmful_behaviors first {len(lines)} goals (auto-fetched). "
        "Institutional policy may require ethics review before publication.\n"
    )
    text = header + "\n".join(lines) + "\n"
    if args.output:
        with open(args.output, "w", encoding="utf-8") as f:
            f.write(text)
        print(f"wrote {len(lines)} lines to {args.output}", file=sys.stderr)
    else:
        sys.stdout.write(text)


if __name__ == "__main__":
    main()
