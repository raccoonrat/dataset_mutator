#!/usr/bin/env python3
"""
Sample the first N prompts from verazuo/jailbreak_llms in-the-wild jailbreak CSV (streaming-friendly).

Large files: we only read until N data rows are collected (does not download entire 3MB+ if N small).

Example:
  python3 experiments/scripts/fetch_wild_jailbreak_sample.py -n 200 -o experiments/data/wild_jailbreak_first200_en.txt
"""

from __future__ import annotations

import argparse
import csv
import io
import sys
import urllib.request

# Smaller snapshot; column name may vary — try common names.
DEFAULT_URL = (
    "https://raw.githubusercontent.com/verazuo/jailbreak_llms/main/"
    "data/prompts/jailbreak_prompts_2023_05_07.csv"
)


def main() -> None:
    ap = argparse.ArgumentParser(description="Sample in-the-wild jailbreak prompts to plain lines")
    ap.add_argument("--url", default=DEFAULT_URL, help="CSV URL (jailbreak prompts)")
    ap.add_argument("-n", type=int, default=200, help="number of prompts to keep")
    ap.add_argument("-o", "--output", required=True)
    args = ap.parse_args()

    req = urllib.request.Request(args.url, headers={"User-Agent": "dataset_mutator-fetch/1.0"})
    with urllib.request.urlopen(req, timeout=300) as resp:
        raw = resp.read().decode("utf-8", errors="replace")

    r = csv.DictReader(io.StringIO(raw))
    if not r.fieldnames:
        raise SystemExit("empty CSV")
    # Guess prompt column
    candidates = ("prompt", "Prompt", "text", "jailbreak", "content", "user_prompt")
    col = next((c for c in candidates if c in r.fieldnames), None)
    if col is None:
        # use first column that looks like text
        col = r.fieldnames[0]

    lines: list[str] = []
    for row in r:
        p = (row.get(col) or "").strip()
        if not p:
            continue
        lines.append(p)
        if len(lines) >= args.n:
            break

    hdr = (
        f"# In-the-wild jailbreak sample n={len(lines)}; column={col!r}; url={args.url}\n"
    )
    from pathlib import Path

    out = Path(args.output)
    out.parent.mkdir(parents=True, exist_ok=True)
    out.write_text(hdr + "\n".join(lines) + "\n", encoding="utf-8")
    print(f"wrote {len(lines)} lines to {out}", file=sys.stderr)


if __name__ == "__main__":
    main()
