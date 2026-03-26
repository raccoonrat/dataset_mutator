#!/usr/bin/env python3
"""Heuristic checks on paper-eval-4 JSON: echo vs real upstream, latency sanity, manifest consistency."""
from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path
from typing import Any, Dict, List


def first_row(
    runs: List[Dict[str, Any]], scenario_id: str, defense: str = "unified"
) -> Dict[str, Any] | None:
    for run in runs:
        for row in run["results"]:
            if row.get("scenario_id") == scenario_id and row.get("defense_baseline") == defense:
                return row
    return None


def main() -> None:
    ap = argparse.ArgumentParser(description="Validate paper-eval JSON for common misconfiguration")
    ap.add_argument("json_path", type=Path)
    args = ap.parse_args()
    doc = json.loads(args.json_path.read_text(encoding="utf-8"))
    man = doc.get("manifest", {})
    runs = doc.get("runs", [])
    issues: List[str] = []
    warns: List[str] = []

    up = str(man.get("upstream_url", ""))
    model = str(man.get("openai_model_field", ""))

    ben = first_row(runs, "benign_baseline", "unified")
    if ben:
        at = str(ben.get("assistant_text", ""))
        lat = float(ben.get("latency_ms") or 0)
        if "[echo]" in at:
            issues.append(
                "经网关的 unified benign_baseline 回复含 `[echo]`：网关进程很可能仍指向 echo-llm，"
                "尽管 manifest 可能声明真实上游。请用与评测相同的 ` . ./env` 启动网关并确认 "
                "`GATEWAY_UPSTREAM=https://api.deepseek.com`（或你的 API），再重跑。"
            )
        if lat < 50 and ("deepseek" in model.lower() or "gpt" in model.lower()):
            warns.append(f"unified benign 延迟仅 {lat:.1f} ms，对公网 LLM 异常偏低，请核对是否走真实上游。")

    harm = first_row(runs, "harmful_rsr_suite", "unified")
    direct_harm = None
    for run in runs:
        for row in run["results"]:
            if row.get("scenario_id") == "harmful_rsr_suite" and row.get("defense_baseline") == "direct_upstream":
                direct_harm = row
                break
        if direct_harm:
            break

    if harm and direct_harm:
        g_lat = float(harm.get("latency_p50_ms") or 0)
        d_lat = float(direct_harm.get("latency_p50_ms") or 0)
        if g_lat < 200 and d_lat > 2000:
            issues.append(
                f"harmful_rsr_suite：经网关 p50={g_lat:.0f} ms，而 direct_upstream p50={d_lat:.0f} ms — "
                "典型表示网关未调用同一公网模型。主表中 Harm. RSR 与 Kw. RSR 不可与 direct 列直接横向解读为「同分布」。"
            )

    if "repro_note" in man:
        warns.append(f"manifest.repro_note 存在: {man['repro_note'][:200]}...")

    print(f"file: {args.json_path}")
    print(f"manifest upstream_url={up!r} openai_model_field={model!r}")
    for w in warns:
        print("WARN:", w)
    for e in issues:
        print("ISSUE:", e)
    if issues:
        sys.exit(2)
    if not issues and not warns:
        print("OK: no obvious echo/latency inconsistencies detected.")
    else:
        sys.exit(1)


if __name__ == "__main__":
    main()
