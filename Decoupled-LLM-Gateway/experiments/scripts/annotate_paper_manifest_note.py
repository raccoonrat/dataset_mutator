#!/usr/bin/env python3
"""Copy benchmark JSON and set manifest.repro_note (for CI fallback paper artifact)."""
from __future__ import annotations

import argparse
import json
from pathlib import Path


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("--src", required=True)
    ap.add_argument("--dst", required=True)
    ap.add_argument("--note", required=True)
    args = ap.parse_args()
    doc = json.loads(Path(args.src).read_text(encoding="utf-8"))
    doc.setdefault("manifest", {})["repro_note"] = args.note
    Path(args.dst).write_text(json.dumps(doc, ensure_ascii=False, indent=2), encoding="utf-8")


if __name__ == "__main__":
    main()
