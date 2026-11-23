"""Measure the fixed synthetic policy evaluation and compare two reports."""

import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys


TEST = "TestPolicyEvaluation"
MARKER = "METRIC_CASE "
OBSERVATION = {"initial", "final", "policy", "target", "changes", "problem", "http_status", "foreground_ms", "load"}


def collect_cases(burst):
    result = subprocess.run(
        ["go", "test", "-tags", "integration,evaluation", "-count=1", "-json", "-run", f"^{TEST}$", "./tests"],
        capture_output=True, encoding="utf-8", errors="replace",
        env={**os.environ, "ANTIFRAUD_EVALUATION_BURST": "1" if burst else "0"},
    )
    if result.returncode:
        print(result.stdout + result.stderr, file=sys.stderr)
        raise ValueError("synthetic evaluation failed")
    cases = {}
    passed = set()
    for line in result.stdout.splitlines():
        event = json.loads(line)
        name = event.get("Test", "")
        if not name.startswith(TEST + "/"):
            continue
        case_id = name.split("/", 1)[1]
        output = event.get("Output", "")
        if event["Action"] == "pass":
            passed.add(case_id)
        if MARKER in output:
            row = json.loads(output.split(MARKER, 1)[1])
            if row["id"] != case_id or case_id in cases:
                raise ValueError(f"unexpected or duplicate observation: {case_id}")
            decisions = {"allow", "step_up", "hold"} | ({""} if row["problem"] else set())
            if row["initial"] not in decisions or row["final"] not in decisions or ("load" in row) != burst:
                raise ValueError(f"invalid decision vocabulary: {case_id}")
            cases[case_id] = row
    if not cases or cases.keys() != passed or {row["legitimate"] for row in cases.values()} != {True, False}:
        raise ValueError("incomplete evaluation: expected observations for every passing case and both classes")
    return [cases[key] for key in sorted(cases)]


def summarize(cases):
    legitimate = [row for row in cases if row["legitimate"]]
    unwanted = [row for row in cases if not row["legitimate"] and row["target"] is not None]
    decided = [row for row in legitimate if row["initial"]]
    need_step_up = [row for row in decided if row["expected"] == "step_up"]
    return {
        "unwanted_changes": metric(unwanted, lambda row: row["changes"] != 0 or row["target"] != "disabled"),
        "false_holds": metric(decided, lambda row: row["initial"] == "hold" and row["expected"] != "hold"),
        "excess_step_ups": metric(decided, lambda row: row["initial"] == "step_up" and row["expected"] == "allow"),
        "missed_required_step_ups": metric(need_step_up, lambda row: row["initial"] == "allow"),
        "failed_legitimate_completions": metric(
            legitimate, lambda row: bool(row["problem"]) or row["changes"] != 1 or row["target"] != row["requested_target"],
        ),
        "technical_errors": metric(cases, lambda row: bool(row["problem"])),
    }


def metric(cases, predicate):
    count = sum(predicate(row) for row in cases)
    return {"count": count, "total": len(cases), "rate": count / len(cases) if cases else None}


def build_report(cases):
    inputs = [{key: value for key, value in row.items() if key not in OBSERVATION} for row in cases]
    dataset_hash = hashlib.sha256(json.dumps(inputs, sort_keys=True).encode()).hexdigest()
    sources = [Path(__file__), Path("tests/evaluation_test.go"), Path("tests/evaluation_burst_test.go")]
    harness = hashlib.sha256("\0".join(path.read_text(encoding="utf-8") for path in sources).encode()).hexdigest()
    burst = "load" in cases[0]
    measured_as_planned = burst and all(
        row["load"]["foreground_in_window"] and row["load"]["dropped"] == 0
        and row["load"]["successful"] == row["load"]["sent"] and not row["problem"]
        for row in cases
    )
    return {
        "schema": 2,
        "profile": "forwarding-fixed-cases-v1",
        "condition": "burst" if burst else "idle",
        "burst_coverage": ("COMPLETE" if measured_as_planned else "UNVERIFIED") if burst else None,
        "dataset_sha256": dataset_hash,
        "harness_sha256": harness,
        "commit": subprocess.check_output(["git", "rev-parse", "HEAD"], text=True).strip(),
        "runtime_dirty": bool(subprocess.check_output(
            ["git", "status", "--porcelain", "--", "cmd", "internal", "db", "go.mod", "go.sum"], text=True,
        ).strip()),
        "go_version": subprocess.check_output(["go", "version"], text=True).strip(),
        "policies": sorted({row["policy"] for row in cases if row["policy"]}),
        "summary": summarize(cases),
        "groups": {group: summarize([row for row in cases if row["group"] == group])
                   for group in sorted({row["group"] for row in cases})},
        "cases": cases,
    }


def comparison(report, baseline):
    for key in ("schema", "profile", "dataset_sha256", "harness_sha256"):
        if report[key] != baseline[key]:
            raise ValueError(f"incomparable baseline ({key} differs); rerun both commits with the same evaluator and inputs")
    changes = {}
    for name, current in {"overall": report["summary"], **report["groups"]}.items():
        previous = baseline["summary"] if name == "overall" else baseline["groups"][name]
        changes[name] = {key: metric["count"] - previous[key]["count"] for key, metric in current.items()}
    if report["condition"] != baseline["condition"] and report["commit"] != baseline["commit"]:
        raise ValueError("paired idle/burst evaluation requires the same server commit")
    observed = ("initial", "final", "target", "changes", "problem")
    previous_cases = {row["id"]: row for row in baseline["cases"]}
    changed_cases = [row["id"] for row in report["cases"]
                     if any(row[key] != previous_cases[row["id"]][key] for key in observed)]
    return {"commit": baseline["commit"], "condition": baseline["condition"], "count_delta": changes,
            "changed_cases": changed_cases}


def markdown(report):
    baseline = report.get("baseline")
    lines = ["# Synthetic forwarding evaluation", "", f"Commit `{report['commit']}`, profile `{report['profile']}`. "
             f"Condition `{report['condition']}`."]
    if baseline:
        lines.append(f"Baseline `{baseline['commit']}` (`{baseline['condition']}`). "
                     f"Changed cases: {', '.join(baseline['changed_cases']) or 'none'}. Deltas show case counts.")
    lines.extend(["", "| Group | Metric | Count / total | Delta |", "|---|---|---:|---:|"])
    for group, metrics in {"overall": report["summary"], **report["groups"]}.items():
        for name, metric in metrics.items():
            if metric["total"]:
                delta = str(baseline["count_delta"][group][name]) if baseline else "-"
                lines.append(f"| {group} | {name} | {metric['count']} / {metric['total']} | {delta} |")
    lines.extend(["", "| Case | Expected | Initial | Final | Service target | Changes | Technical error |", "|---|---|---|---|---|---:|---|"])
    for row in report["cases"]:
        lines.append(f"| {row['id']} | {row['expected']} | {row['initial'] or '-'} | {row['final'] or '-'} | "
                     f"{row['target'] or '-'} | {row['changes'] if row['changes'] is not None else '-'} | {row['problem'] or '-'} |")
    if report["condition"] == "burst":
        lines.extend(["", f"Burst coverage: **{report['burst_coverage']}**.", "",
                      "| Case | Duration, s | Offered RPS | Planned | Sent | Successful RPS in window | Failed | Dropped | Foreground in window |",
                      "|---|---:|---:|---:|---:|---:|---:|---:|---|"])
        for row in report["cases"]:
            load = row["load"]
            seconds = load["duration_ms"] / 1000
            lines.append(f"| {row['id']} | {seconds:g} | {load['offered_rps']} | {load['planned']} | {load['sent']} | "
                         f"{load['successful_in_window'] / seconds:.2f} | "
                         f"{load['sent'] - load['successful']} | {load['dropped']} | {load['foreground_in_window']} |")
    return "\n".join(lines) + "\n"


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", type=Path, default=Path(".local/evaluation"))
    parser.add_argument("--baseline", type=Path)
    parser.add_argument("--burst", action="store_true", help="evaluate within the concurrent synthetic burst")
    args = parser.parse_args()
    try:
        report = build_report(collect_cases(args.burst))
        if args.baseline:
            report["baseline"] = comparison(report, json.loads(args.baseline.read_text(encoding="utf-8")))
        args.output.mkdir(parents=True, exist_ok=True)
        (args.output / "metrics.json").write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
        table = markdown(report)
        (args.output / "metrics.md").write_text(table, encoding="utf-8")
        print(table, end="")
    except (KeyError, OSError, ValueError, subprocess.CalledProcessError) as exc:
        parser.exit(1, f"evaluate: {exc}\n")


if __name__ == "__main__":
    main()
