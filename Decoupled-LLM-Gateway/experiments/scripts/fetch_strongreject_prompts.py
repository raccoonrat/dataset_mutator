#!/usr/bin/env python3
"""
Download StrongREJECT forbidden prompts (CSV) as a one-line-per-prompt file for harmful_rsr_suite.

Source (public): alexandrasouly/strongreject strongreject_dataset.csv
See also: dsbowen/strong_reject (evaluator code); paper uses StrongREJECT to mitigate empty-jailbreak ASR.

Example:
  python3 experiments/scripts/fetch_strongreject_prompts.py -o experiments/data/strongreject_forbidden_en.txt
"""

from __future__ import annotations

import argparse
import csv
import io
import sys
import urllib.request

DEFAULT_URL = (
    "https://raw.githubusercontent.com/alexandrasouly/strongreject/main/"
    "strongreject_dataset/strongreject_dataset.csv"
)


def main() -> None:
    ap = argparse.ArgumentParser(description="Fetch StrongREJECT forbidden_prompt column to text lines")
    ap.add_argument("--url", default=DEFAULT_URL, help="CSV URL")
    ap.add_argument("-n", type=int, default=0, help="if >0, keep only first N prompts after header")
    ap.add_argument("-o", "--output", required=True, help="output UTF-8 text file")
    args = ap.parse_args()

    req = urllib.request.Request(
        args.url,
        headers={"User-Agent": "dataset_mutator-fetch/1.0"},
    )
    with urllib.request.urlopen(req, timeout=120) as resp:
        raw = resp.read().decode("utf-8", errors="replace")

    r = csv.DictReader(io.StringIO(raw))
    lines: list[str] = []
    for row in r:
        p = (row.get("forbidden_prompt") or row.get("Forbidden_prompt") or "").strip()
        if not p:
            continue
        lines.append(p)
        if args.n and len(lines) >= args.n:
            break

    hdr = (
        f"# StrongREJECT forbidden_prompt (n={len(lines)}); source {args.url}\n"
        "# Ethics / institutional policy may apply before publication runs.\n"
    )
    from pathlib import Path

    out = Path(args.output)
    out.parent.mkdir(parents=True, exist_ok=True)
    out.write_text(hdr + "\n".join(lines) + "\n", encoding="utf-8")
    print(f"wrote {len(lines)} lines to {out}", file=sys.stderr)


if __name__ == "__main__":
    main()
