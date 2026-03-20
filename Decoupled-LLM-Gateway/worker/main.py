#!/usr/bin/env python3
"""Async governance loop — Milestone 1 stub.

Reads newline-delimited JSON (GatewayLogEvent) from stdin.
Milestone 3: hybrid judge + Redis policy push to the gateway process.
"""

from __future__ import annotations

import json
import sys


def main() -> None:
    for line in sys.stdin:
        line = line.strip()
        if not line:
            continue
        try:
            ev = json.loads(line)
        except json.JSONDecodeError as e:
            print(f"skip invalid json: {e}", file=sys.stderr)
            continue
        tid = ev.get("trace_id", "")
        degraded = ev.get("degradation_triggered", False)
        print(f"[worker] trace={tid} degraded={degraded}", file=sys.stderr)


if __name__ == "__main__":
    main()
