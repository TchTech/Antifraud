"""Measure synthetic protected checks against a dedicated loopback gateway."""

import argparse
import hashlib
import http.client
import json
import math
import os
from pathlib import Path
import platform
import subprocess
import threading
import time
import uuid
from collections import Counter
from concurrent.futures import ThreadPoolExecutor
from datetime import datetime, timezone


METHOD = "loopback-gateway/open-arrival/one-connection-per-request/v1"
RESULTS = {"allow", "step_up", "hold"}


def endpoint(port, timeout, method, path, body=None):
    connection = http.client.HTTPConnection("127.0.0.1", port, timeout=timeout)
    try:
        connection.request(method, path, body, {"Content-Type": "application/json"})
        response = connection.getresponse()
        return response.status, response.read()
    finally:
        connection.close()


def protected_check(port, timeout, due):
    operation = str(uuid.uuid4())
    body = json.dumps({"accountId": "demo-clean", "operationId": operation, "target": "sink-a",
                       "device": "known", "attestation": "demo-valid"})
    started = time.perf_counter()
    sample = {"dispatch_delay_ms": (started - due) * 1000}
    try:
        status, raw = endpoint(port, timeout, "POST", "/v1/evaluate", body)
        sample["http_status"] = status
        reply = json.loads(raw)
        if (status == 200 and isinstance(reply, dict) and reply.get("operationId") == operation
                and isinstance(reply.get("result"), str) and reply["result"] in RESULTS
                and isinstance(reply.get("policyVersion"), str)
                and reply["policyVersion"]):
            sample.update(result="ok", decision=reply["result"], policy=reply["policyVersion"])
        else:
            sample["result"] = "bad_response"
    except TimeoutError:
        sample["result"] = "timeout"
    except (OSError, http.client.HTTPException, ValueError):
        sample["result"] = "invalid_or_unavailable"
    finished = time.perf_counter()
    sample["latency_ms"] = (finished - started) * 1000
    return finished, sample


def workload(profile, port):
    timeout = profile["timeout_ms"] / 1000
    status, raw = endpoint(port, timeout, "GET", "/v1/state/demo-clean")
    state = json.loads(raw)
    if status != 200 or not isinstance(state, dict) or "changes" not in state:
        raise ValueError("dedicated synthetic gateway did not return demo-clean state")
    initial = {"target": state["target"], "changes": state["changes"],
               "decisions": len(state["decisions"] or []), "audit": len(state["audit"] or [])}
    limit = threading.BoundedSemaphore(profile["max_in_flight"])
    phases = []
    with ThreadPoolExecutor(max_workers=profile["max_in_flight"]) as pool:
        origin = time.perf_counter()
        stage_offset = 0
        for phase in profile["phases"]:
            rate = phase["requests_per_second"]
            duration = phase["seconds"]
            stage_start = origin + stage_offset
            jobs = []
            drops = Counter()
            for index in range(duration * rate):
                due = stage_start + index / rate
                wait = due - time.perf_counter()
                if wait > 0:
                    time.sleep(wait)
                if time.perf_counter() >= due + 1 / rate:
                    drops["generator_late"] += 1
                elif not limit.acquire(blocking=False):
                    drops["in_flight_limit"] += 1
                else:
                    job = pool.submit(protected_check, port, timeout, due)
                    job.add_done_callback(lambda _: limit.release())
                    jobs.append(job)
            phases.append((phase, stage_start + duration, jobs, drops))
            stage_offset += duration

    summaries, samples = [], []
    for phase, end, jobs, drops in phases:
        observations = [job.result() for job in jobs]
        summaries.append(summarize(phase, end, observations, drops))
        samples.extend({"phase": phase["name"], **sample} for _, sample in observations)
    return initial, summaries, samples


def summarize(phase, end, observations, drops):
    duration = phase["seconds"]
    planned = duration * phase["requests_per_second"]
    outcomes = Counter(sample["result"] for _, sample in observations)
    successes = [(finished, sample) for finished, sample in observations if sample["result"] == "ok"]
    on_time = sum(finished < end for finished, _ in successes)
    latency = sorted(sample["latency_ms"] for _, sample in successes)
    failures = len(observations) - len(successes)
    assert planned == len(observations) + sum(drops.values())
    return {
        "phase": phase["name"], "seconds": duration, "offered_rps": phase["requests_per_second"],
        "planned": planned, "started": len(observations), "successful": len(successes),
        "started_rps": len(observations) / duration, "goodput_rps": on_time / duration,
        "success_after_phase": len(successes) - on_time,
        "failed": failures, "failure_rate": failures / len(observations) if observations else None,
        "timeouts": outcomes["timeout"], "dropped": sum(drops.values()),
        "drop_reasons": {key: drops[key] for key in ("generator_late", "in_flight_limit")},
        "outcomes": dict(outcomes), "decisions": dict(Counter(sample["decision"] for _, sample in successes)),
        "http_statuses": dict(Counter(str(sample["http_status"]) for _, sample in observations if "http_status" in sample)),
        "latency_ms": {f"p{p}": latency[math.ceil(p / 100 * len(latency)) - 1] if latency else None
                       for p in (50, 95, 99, 100)},
        "max_dispatch_delay_ms": max((sample["dispatch_delay_ms"] for _, sample in observations), default=None),
    }


def compare(report, base):
    for key in ("method", "profile_sha256", "generator_sha256", "initial_history", "generator_environment"):
        if report[key] != base[key]:
            raise ValueError(f"incomparable baseline: {key} differs")
    if not report["server_ref"] or not base["server_ref"]:
        raise ValueError("supply --server-ref for both runs before comparing server commits")
    return {
        "server_ref": base["server_ref"],
        "phase_deltas": {current["phase"]: {
            "p99_ms": current["latency_ms"]["p99"] - prior["latency_ms"]["p99"]
            if current["latency_ms"]["p99"] is not None and prior["latency_ms"]["p99"] is not None else None,
            **{key: current[key] - prior[key] for key in ("goodput_rps", "failed", "timeouts", "dropped")},
        } for current, prior in zip(report["phases"], base["phases"], strict=True)},
    }


def markdown(report):
    rows = ["# Protected-check gateway load", "", f"Profile `{report['profile']['name']}`. "
            f"Server `{report['server_ref'] or 'unrecorded'}`. Budget **{report['budget_status']}**.", "",
            "| Phase | Offered RPS | Started RPS | Goodput RPS | p50 ms | p95 ms | p99 ms | Max ms | Failed | Timeouts | Dropped |",
            "|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|"]
    for phase in report["phases"]:
        fields = [phase[key] for key in ("offered_rps", "started_rps", "goodput_rps")]
        fields.extend(phase["latency_ms"][key] for key in ("p50", "p95", "p99", "p100"))
        fields.extend(phase[key] for key in ("failed", "timeouts", "dropped"))
        rows.append("| " + " | ".join([phase["phase"], *["-" if val is None else f"{val:.2f}" for val in fields]]) + " |")
    if "baseline" in report:
        rows.extend(["", f"Deltas against `{report['baseline']['server_ref']}` (current minus baseline).", "",
                     "| Phase | p99 ms | Goodput RPS | Failed | Timeouts | Dropped |",
                     "|---|---:|---:|---:|---:|---:|"])
        for phase, delta in report["baseline"]["phase_deltas"].items():
            rows.append("| " + " | ".join([phase, *["-" if val is None else f"{val:+.2f}" for val in delta.values()]]) + " |")
    return "\n".join(rows) + "\n"


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--port", type=int, default=18080, help="dedicated localhost gateway port")
    parser.add_argument("--profile", type=Path, default=Path("docs/load/gateway-profile.json"))
    parser.add_argument("--output", type=Path, default=Path(".local/load"))
    parser.add_argument("--baseline", type=Path)
    parser.add_argument("--server-ref", default=os.environ.get("LOAD_SERVER_REF"), help="revision of the running server")
    parser.add_argument("--check", action="store_true", help="exit nonzero on a measured budget violation")
    args = parser.parse_args()
    try:
        data = args.profile.read_text(encoding="utf-8")
        profile = json.loads(data)
        phases = profile["phases"]
        names = [phase["name"] for phase in phases]
        positive = [profile[key] for key in ("timeout_ms", "max_in_flight")]
        positive.extend(phase[key] for phase in phases for key in ("seconds", "requests_per_second"))
        if len(set(names)) != len(names) or not set(names) - {"warmup"}:
            raise ValueError("profile requires unique phases and at least one measured phase")
        if any(type(value) is not int or value <= 0 for value in positive) or not 1 <= args.port <= 65535:
            raise ValueError("profile rates, durations, timeout, worker limit and port must be positive integers in range")
        budget = profile["budget"]
        if type(budget["max_p99_ms"]) not in (int, float) or not math.isfinite(budget["max_p99_ms"]) or budget["max_p99_ms"] <= 0:
            raise ValueError("max_p99_ms must be a positive finite number")
        if any(type(budget[key]) is not int or budget[key] < 0 for key in ("max_failures", "max_dropped")):
            raise ValueError("failure and drop budgets must be nonnegative integers")
        started_at = datetime.now(timezone.utc).isoformat()
        initial, summaries, samples = workload(profile, args.port)
        violations = [s["phase"] for s in summaries if s["phase"] != "warmup" and (
            s["latency_ms"]["p99"] is None or s["latency_ms"]["p99"] > budget["max_p99_ms"]
            or s["failed"] > budget["max_failures"] or s["dropped"] > budget["max_dropped"]
        )]
        report = {"method": METHOD, "server_ref": args.server_ref, "started_at": started_at, "profile": profile,
                  "profile_sha256": hashlib.sha256(data.encode()).hexdigest(),
                  "generator_sha256": hashlib.sha256(Path(__file__).read_text(encoding="utf-8").encode()).hexdigest(),
                  "generator_environment": {"os": platform.platform(), "machine": platform.machine(),
                                            "python": platform.python_version(), "logical_cpus": os.cpu_count()},
                  "initial_history": initial, "phases": summaries,
                  "budget_status": "FAIL" if violations else "PASS", "budget_violations": violations}
        if args.baseline:
            report["baseline"] = compare(report, json.loads(args.baseline.read_text(encoding="utf-8")))
        args.output.mkdir(parents=True, exist_ok=True)
        (args.output / "metrics.json").write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
        with (args.output / "samples.jsonl").open("w", encoding="utf-8") as raw:
            for sample in samples:
                raw.write(json.dumps(sample) + "\n")
        table = markdown(report)
        (args.output / "metrics.md").write_text(table, encoding="utf-8")
        print(table, end="")
        if args.check and violations:
            parser.exit(1, "load: measured budget failed in " + ", ".join(violations) + "\n")
    except (OSError, ValueError, KeyError, TypeError, http.client.HTTPException) as exc:
        parser.exit(1, f"load: {exc}\n")


if __name__ == "__main__":
    main()
