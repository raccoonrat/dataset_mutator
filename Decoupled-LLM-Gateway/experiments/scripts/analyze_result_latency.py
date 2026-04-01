#!/usr/bin/env python3
"""Aggregate latency_ms from paper benchmark JSON / checkpoint (recursive tree walk)."""

from __future__ import annotations

import argparse
import json
import statistics
import sys
from pathlib import Path
from typing import Any, List


def collect_latencies(obj: Any, out: List[float]) -> None:
    if isinstance(obj, dict):
        v = obj.get("latency_ms")
        if isinstance(v, (int, float)):
            out.append(float(v))
        for x in obj.values():
            collect_latencies(x, out)
    elif isinstance(obj, list):
        for x in obj:
            collect_latencies(x, out)


def percentile_nearest(sorted_vals: List[float], p: float) -> float:
    if not sorted_vals:
        return 0.0
    k = max(0, min(len(sorted_vals) - 1, int(round((p / 100.0) * (len(sorted_vals) - 1)))))
    return sorted_vals[k]


def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("json_path", type=Path, help="result .json or .checkpoint.json")
    args = ap.parse_args()
    p = args.json_path
    if not p.is_file():
        print(f"not a file: {p}", file=sys.stderr)
        sys.exit(1)
    doc = json.loads(p.read_text(encoding="utf-8"))
    lat: List[float] = []
    if "runs" in doc:
        for r in doc["runs"]:
            collect_latencies(r.get("results", []), lat)
    else:
        collect_latencies(doc, lat)
    n = len(lat)
    if not n:
        print("no latency_ms values found")
        return
    s = sorted(lat)
    mean = statistics.fmean(lat)
    p50 = percentile_nearest(s, 50)
    p95 = percentile_nearest(s, 95)
    print(f"samples={n}")
    print(f"latency_ms mean={mean:.2f} p50={p50:.2f} p95={p95:.2f}")
    print(f"latency_s  mean={mean/1000:.4f} p50={p50/1000:.4f} p95={p95/1000:.4f}")


if __name__ == "__main__":
    main()
