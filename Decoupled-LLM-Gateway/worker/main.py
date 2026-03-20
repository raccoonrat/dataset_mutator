#!/usr/bin/env python3
"""Milestone 3 async loop: consume gateway log events, hybrid judge (decoy leak), publish PolicyRule to Redis hash."""

from __future__ import annotations

import argparse
import json
import logging
import os
import re
import sys
from typing import Any, Dict, Optional

try:
    import redis
except ImportError:
    redis = None  # type: ignore

DEFAULT_STREAM = "decoupled:gateway:events"
DEFAULT_GROUP = "decoupled-gw-workers"
DEFAULT_HASH = "decoupled:policy:rules"
DEFAULT_CONSUMER = "worker-1"
RULE_ID_DECOY_LEAK = "m3-decoy-leak"

logging.basicConfig(level=logging.INFO, format="%(message)s")
log = logging.getLogger("worker")


def judge(ev: Dict[str, Any]) -> Optional[Dict[str, Any]]:
    """Hybrid judge MVP: detect decoy token echoed in model output."""
    if ev.get("degradation_triggered"):
        return None
    decoy = (ev.get("injected_decoy_id") or "").strip()
    resp = ev.get("llm_response") or ""
    if decoy and decoy in resp:
        obf = (ev.get("obfuscated_prompt") or "").strip()
        if len(obf) < 3:
            return None
        chunk = obf[:120]
        pattern = re.escape(chunk)
        return {
            "rule_id": RULE_ID_DECOY_LEAK,
            "action": "DEGRADE_TO_TEMPLATE",
            "trigger_pattern": pattern,
            "template_response": "异步环检测到诱饵泄露，已对该类请求启用前置降级。",
        }
    return None


def push_rule(client: Any, hash_key: str, rule: Dict[str, Any]) -> None:
    payload = json.dumps(rule, ensure_ascii=False)
    client.hset(hash_key, rule["rule_id"], payload)
    log.info("HSET %s %s", hash_key, rule["rule_id"])


def run_stdin() -> None:
    for line in sys.stdin:
        line = line.strip()
        if not line:
            continue
        try:
            ev = json.loads(line)
        except json.JSONDecodeError as e:
            log.warning("skip invalid json: %s", e)
            continue
        rule = judge(ev)
        if rule:
            print(json.dumps(rule, ensure_ascii=False))


def run_redis(url: str, stream: str, group: str, consumer: str, hash_key: str) -> None:
    if redis is None:
        raise SystemExit("install redis-py: pip install -r worker/requirements.txt")
    client = redis.Redis.from_url(url, decode_responses=True)
    try:
        client.xgroup_create(stream, group, id="0", mkstream=True)
    except redis.ResponseError as e:
        if "BUSYGROUP" not in str(e):
            raise
    log.info("xreadgroup stream=%s group=%s consumer=%s", stream, group, consumer)
    while True:
        resp = client.xreadgroup(group, consumer, {stream: ">"}, count=32, block=5000)
        if not resp:
            continue
        for _sname, entries in resp:
            for entry_id, fields in entries:
                raw = fields.get("payload")
                if raw is None:
                    log.warning("missing payload id=%s", entry_id)
                    client.xack(stream, group, entry_id)
                    continue
                try:
                    ev = json.loads(raw)
                except json.JSONDecodeError:
                    log.warning("bad json id=%s", entry_id)
                    client.xack(stream, group, entry_id)
                    continue
                rule = judge(ev)
                if rule:
                    push_rule(client, hash_key, rule)
                client.xack(stream, group, entry_id)


def main() -> None:
    ap = argparse.ArgumentParser(description="Decoupled LLM Gateway async worker (M3)")
    ap.add_argument("--redis-url", default=os.environ.get("WORKER_REDIS_URL", "").strip())
    ap.add_argument("--stream", default=os.environ.get("WORKER_STREAM", DEFAULT_STREAM))
    ap.add_argument("--group", default=os.environ.get("WORKER_GROUP", DEFAULT_GROUP))
    ap.add_argument("--consumer", default=os.environ.get("WORKER_CONSUMER", DEFAULT_CONSUMER))
    ap.add_argument("--policy-hash", default=os.environ.get("WORKER_POLICY_HASH", DEFAULT_HASH))
    ap.add_argument("--stdin", action="store_true", help="read NDJSON lines from stdin (no Redis)")
    args = ap.parse_args()

    if args.stdin:
        run_stdin()
        return
    if not args.redis_url:
        log.info("no --redis-url / WORKER_REDIS_URL; reading NDJSON from stdin")
        run_stdin()
        return
    run_redis(args.redis_url, args.stream, args.group, args.consumer, args.policy_hash)


if __name__ == "__main__":
    main()
