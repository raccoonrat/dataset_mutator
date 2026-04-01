#!/usr/bin/env python3
"""
Re-label Track A RSR/FPR file-suite rows using the same HTTP judge contract as run_paper_benchmark
(PAPER_EVAL_JUDGE_URL / judge_service with JUDGE_BACKEND=chat_completion + OpenRouter Llama Guard 4).

Requires JSON from run_paper_benchmark.py with --store-assistant-text so each per_prompt entry has
user_text and assistant_text.

Writes parallel rate keys (default suffix llama_guard4), e.g. harmful_rsr_rate_llama_guard4,
and per_prompt booleans is_refusal_llama_guard4. Updates manifest.secondary_judge_relabel.

Example:
  export PAPER_EVAL_JUDGE_URL=http://127.0.0.1:8765/judge
  python3 experiments/scripts/relabel_tracka_llama_guard.py \\
    --json results/trackA.json --out results/trackA_lg4.json
"""

from __future__ import annotations

import argparse
import json
import os
import sys
import time
import urllib.error
import urllib.request
from pathlib import Path
from typing import Any, Dict, List, Optional, Tuple


def http_judge_refusal(user_prompt: str, assistant_text: str, url: str, timeout_s: float) -> bool:
    payload = json.dumps(
        {"task": "refusal_binary", "user": user_prompt, "assistant": assistant_text},
        ensure_ascii=False,
    ).encode("utf-8")
    req = urllib.request.Request(url, data=payload, method="POST", headers={"Content-Type": "application/json"})
    bearer = (os.environ.get("PAPER_EVAL_JUDGE_BEARER") or "").strip()
    if bearer:
        req.add_header("Authorization", f"Bearer {bearer}")
    with urllib.request.urlopen(req, timeout=timeout_s) as resp:
        body = json.loads(resp.read().decode("utf-8", errors="replace"))
    if isinstance(body, dict):
        if "is_refusal" in body:
            return bool(body["is_refusal"])
        if "refusal" in body:
            return bool(body["refusal"])
    raise ValueError(f"judge: expected is_refusal or refusal, got {type(body).__name__}")


# scenario_id -> (aggregate_rate_key_to_write, per_prompt_flag_meaning is "refusal" for harmful suites)
_SUITE_AGG: Dict[str, str] = {
    "harmful_rsr_suite": "harmful_rsr_rate",
    "wild_rsr_suite": "wild_rsr_rate",
    "strongreject_rsr_suite": "strongreject_rsr_rate",
    "benign_fpr_suite": "improper_refusal_rate",
    "hpm_proxy": "hpm_rsr_rate",
}


def relabel_suite_row(row: Dict[str, Any], *, judge_url: str, timeout_s: float, suffix: str) -> Tuple[int, int]:
    sid = row.get("scenario_id")
    if sid not in _SUITE_AGG:
        return 0, 0
    base_key = _SUITE_AGG[sid]
    pps: List[Dict[str, Any]] = list(row.get("per_prompt") or [])
    if not pps:
        return 0, 0
    n_ok = 0
    n_skip = 0
    flag_key = f"is_refusal_{suffix}"
    flags: List[bool] = []
    for pp in pps:
        ut = (pp.get("user_text") or "").strip()
        if not ut:
            ut = (pp.get("prompt_preview") or "").strip()
        at = (pp.get("assistant_text") or "").strip()
        if not at:
            n_skip += 1
            continue
        try:
            ref = http_judge_refusal(ut, at, judge_url, timeout_s)
        except (urllib.error.URLError, urllib.error.HTTPError, ValueError, TimeoutError) as e:
            print(f"warn: judge failed scenario={sid} preview={pp.get('prompt_preview')!r}: {e}", file=sys.stderr)
            n_skip += 1
            continue
        pp[flag_key] = ref
        flags.append(bool(ref))
        n_ok += 1
    agg_key = f"{base_key}_{suffix}"
    if flags:
        row[agg_key] = sum(1 for f in flags if f) / len(flags)
    else:
        row.pop(agg_key, None)
    return n_ok, n_skip


def process_doc(doc: Dict[str, Any], *, judge_url: str, timeout_s: float, suffix: str) -> None:
    runs = doc.get("runs") or []
    total_ok = total_skip = 0
    for run in runs:
        for row in run.get("results") or []:
            ok, sk = relabel_suite_row(row, judge_url=judge_url, timeout_s=timeout_s, suffix=suffix)
            total_ok += ok
            total_skip += sk
    man = doc.setdefault("manifest", {})
    man["secondary_judge_relabel"] = {
        "suffix": suffix,
        "judge_url": judge_url,
        "judge_model_revision": (
            (man.get("eval_judge_chat") or {}).get("judge_model_revision")
            or os.environ.get("JUDGE_MODEL_REVISION", "").strip()
            or os.environ.get("JUDGE_CHAT_MODEL", "").strip()
        ),
        "relabeled_at_unix": time.time(),
        "per_prompt_labels_ok": total_ok,
        "per_prompt_labels_skipped": total_skip,
    }
    print(
        f"relabel done: labeled={total_ok} skipped={total_skip} suffix={suffix}",
        file=sys.stderr,
    )


def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--json", required=True, type=Path, help="input paper-eval JSON")
    ap.add_argument("--out", required=True, type=Path, help="output JSON (can overwrite input)")
    ap.add_argument(
        "--judge-url",
        default="",
        help="override PAPER_EVAL_JUDGE_URL",
    )
    ap.add_argument("--timeout-s", type=float, default=120.0)
    ap.add_argument(
        "--suffix",
        default="llama_guard4",
        help="suffix for new keys, e.g. harmful_rsr_rate_<suffix> (default llama_guard4)",
    )
    args = ap.parse_args()
    judge_url = (args.judge_url or os.environ.get("PAPER_EVAL_JUDGE_URL") or "").strip()
    if not judge_url:
        print("need --judge-url or PAPER_EVAL_JUDGE_URL", file=sys.stderr)
        sys.exit(2)
    doc = json.loads(args.json.read_text(encoding="utf-8"))
    process_doc(doc, judge_url=judge_url, timeout_s=args.timeout_s, suffix=args.suffix.strip() or "llama_guard4")
    args.out.parent.mkdir(parents=True, exist_ok=True)
    atomic = args.out.with_suffix(args.out.suffix + ".tmp")
    atomic.write_text(json.dumps(doc, ensure_ascii=False, indent=2), encoding="utf-8")
    os.replace(atomic, args.out)
    print("wrote", args.out, file=sys.stderr)


if __name__ == "__main__":
    main()
