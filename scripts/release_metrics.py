"""Create the two release evidence assets from measurements on one runner."""

import argparse
from collections import Counter
import json
from pathlib import Path
import statistics
from zipfile import ZIP_DEFLATED, ZipFile


REPEATS = 3
LOAD_KEYS = ("method", "profile_sha256", "generator_sha256", "initial_history", "generator_environment")
QUALITY_KEYS = ("schema", "profile", "dataset_sha256", "harness_sha256")


def read_json(path):
    return json.loads(path.read_text(encoding="utf-8")) if path.is_file() else None


def measurements(root, commit):
    quality = read_json(root / "evaluation/metrics.json")
    if quality and (quality["commit"] != commit or quality["runtime_dirty"]):
        raise ValueError(f"evaluation provenance does not match {commit}")
    load = []
    for number in range(1, REPEATS + 1):
        path = root / "load" / f"run-{number}" / "metrics.json"
        report = read_json(path)
        if report:
            if report["server_ref"] != commit:
                raise ValueError(f"load provenance does not match {commit}")
            samples = path.parent / "samples.jsonl"
            if not samples.is_file():
                raise ValueError(f"raw samples missing for {root.name} run {number}")
            with samples.open(encoding="utf-8") as raw:
                counts = Counter(json.loads(line)["phase"] for line in raw)
            expected = Counter({phase["phase"]: phase["started"] for phase in report["phases"]})
            if counts != expected:
                raise ValueError(f"raw sample counts do not match {root.name} run {number}")
            load.append((number, report))
    return quality, load


def equivalent(reports, keys):
    return bool(reports) and all(item[key] == reports[0][key] for item in reports for key in keys)


def complete(load):
    reports = [report for _, report in load]
    phases = [[phase["phase"] for phase in report["phases"]] for report in reports]
    return (len(reports) == REPEATS and equivalent(reports, LOAD_KEYS)
            and len({report["started_at"] for report in reports}) == REPEATS
            and bool(phases[0]) and all(names == phases[0] for names in phases))


def number(value):
    return "—" if value is None else f"{value:.2f}"


def rollup(load):
    summary = {}
    for _, report in load:
        for phase in report["phases"]:
            counts = ("goodput_rps", "failed", "timeouts", "dropped")
            item = summary.setdefault(phase["phase"], {key: [] for key in ("p95", "p99", *counts)})
            item["p95"].append(phase["latency_ms"]["p95"])
            item["p99"].append(phase["latency_ms"]["p99"])
            for key in counts:
                item[key].append(phase[key])
    return summary


def range_text(values):
    available = [value for value in values if value is not None]
    if not available:
        return "—"
    return f"{number(min(available))}–{number(max(available))} ({len(available)}/{len(values)})"


def delta(current, baseline):
    if any(value is None for value in current + baseline):
        return "UNVERIFIED"
    return f"{statistics.median(current) - statistics.median(baseline):+.2f}"


def create_report(tag, commit, baseline_tag, baseline_commit, evidence, output):
    environment = read_json(evidence / "environment.json") or {}
    candidate = evidence / "candidate"
    quality, load = measurements(candidate, commit)
    burst = read_json(candidate / "evaluation-burst/metrics.json")
    if burst and (burst["commit"] != commit or burst["runtime_dirty"] or burst["condition"] != "burst"
                  or not quality or not equivalent([quality, burst], QUALITY_KEYS)
                  or burst["baseline"]["commit"] != commit or burst["baseline"]["condition"] != "idle"):
        raise ValueError(f"paired burst provenance does not match {commit}")
    prior_quality, prior_load = measurements(evidence / "baseline", baseline_commit) if baseline_tag else (None, [])
    same_job = bool(environment.get("measurement_session"))
    quality_comparable = equivalent([quality, prior_quality], QUALITY_KEYS) if quality and prior_quality else False
    load_comparable = (same_job and complete(load) and complete(prior_load)
                       and equivalent([load[0][1], prior_load[0][1]], LOAD_KEYS)
                       and list(rollup(load)) == list(rollup(prior_load)))
    check_file = candidate / "e2e.status"
    e2e = check_file.read_text(encoding="utf-8").strip() if check_file.is_file() else "UNVERIFIED"
    if e2e not in {"PASS", "FAIL", "UNVERIFIED"}:
        raise ValueError(f"unexpected e2e status: {e2e}")
    budgets = {report["budget_status"] for _, report in load}
    if budgets - {"PASS", "FAIL"}:
        raise ValueError("unknown load budget status")
    budget = "FAIL" if "FAIL" in budgets else "PASS" if complete(load) else "UNVERIFIED"

    rows = [f"# Метрики релиза `{tag}`", "", f"SHA `{commit}`. Интеграционные тесты **{e2e}**. Бюджет пилота **{budget}**.",
            f"Сравнение с `{baseline_tag or '—'}`. Политика **{'COMPARABLE' if quality_comparable else 'UNVERIFIED'}**, "
            f"нагрузка **{'COMPARABLE' if load_comparable else 'UNVERIFIED'}**.",
            "Нагрузка измеряет один поток HTTP-проверок. В таблице дельты кандидата относительно базы.", "",
            "## Исходы политики", ""]
    if quality:
        rows.extend(["| Метрика | Случаи | Дельта, шт. |", "|---|---:|---:|"])
        for label, metric in quality["summary"].items():
            difference = metric["count"] - prior_quality["summary"][label]["count"] if quality_comparable else None
            rows.append(f"| {label} | {metric['count']} / {metric['total']} | {difference if difference is not None else '—'} |")
    else:
        rows.append("UNVERIFIED — отчёт оценки политики отсутствует.")

    rows.extend(["", "## Исходы сценариев при всплеске", ""])
    if burst:
        changed = burst["baseline"]["changed_cases"]
        rows.append(f"Покрытие заданной нагрузки **{burst['burst_coverage']}**. "
                    f"Изменившиеся исходы: {', '.join(changed) or 'нет среди наблюдений'}.")
        rows.extend(["", "| Сценарий | Задано, RPS | Успешных RPS в окне | Ошибки фона | Пропуски | Сценарий в окне |",
                     "|---|---:|---:|---:|---:|---|"])
        for case in burst["cases"]:
            background = case["load"]
            rate = 1000 * background["successful_in_window"] / background["duration_ms"]
            rows.append(f"| {case['id']} | {background['offered_rps']} | {number(rate)} | "
                        f"{background['sent'] - background['successful']} | {background['dropped']} | "
                        f"{'да' if background['foreground_in_window'] else 'нет'} |")
    else:
        rows.append("UNVERIFIED — парный прогон отсутствует.")

    rows.extend(["", "## Нагрузка по повторам", "", f"Получено {len(load)} из {REPEATS} отчётов.", "",
                 "| Повтор | Фаза | Задано, RPS | Фактически, RPS | p95, мс | p99, мс | Ошибки | Таймауты | Пропуски |",
                 "|---:|---|---:|---:|---:|---:|---:|---:|---:|"])
    for index, report in load:
        for phase in report["phases"]:
            rows.append(f"| {index} | {phase['phase']} | {phase['offered_rps']} | {number(phase['goodput_rps'])} | "
                        f"{number(phase['latency_ms']['p95'])} | {number(phase['latency_ms']['p99'])} | "
                        f"{phase['failed']} | {phase['timeouts']} | {phase['dropped']} |")

    rows.extend(["", "## Разброс и сравнение", "",
                 "Диапазон по повторам указан вместе с числом измерений. Дельты считаются по медианам повторов.", "",
                 "| Фаза | Диапазон p99, мс | Диапазон успешных RPS | Δ p95, мс | Δ p99, мс | Δ RPS | Δ ошибок | Δ таймаутов | Δ пропусков |",
                 "|---|---:|---:|---:|---:|---:|---:|---:|---:|"])
    prior = rollup(prior_load)
    for phase, values in rollup(load).items():
        changes = [delta(values[key], prior[phase][key]) if load_comparable else "—" for key in values]
        cells = [phase, range_text(values["p99"]), range_text(values["goodput_rps"]), *changes]
        rows.append("| " + " | ".join(cells) + " |")

    output.mkdir(parents=True, exist_ok=True)
    markdown = "\n".join(rows) + "\n"
    (output / "release-metrics.md").write_text(markdown, encoding="utf-8")
    manifest = {
        "schema": 1, "tag": tag, "commit": commit, "baseline_tag": baseline_tag, "baseline_commit": baseline_commit,
        "evaluation_comparable": quality_comparable, "load_comparable": load_comparable, "e2e": e2e,
        "burst_coverage": burst["burst_coverage"] if burst else "UNVERIFIED",
        "load_budget": budget, "environment": environment,
    }
    with ZipFile(output / "release-evidence.zip", "w", compression=ZIP_DEFLATED) as archive:
        archive.writestr("manifest.json", json.dumps(manifest, indent=2) + "\n")
        archive.writestr("release-metrics.md", markdown)
        patterns = ("compose.yaml", "e2e.status", "evaluation/metrics.json", "evaluation/metrics.md",
                    "evaluation-burst/metrics.json", "evaluation-burst/metrics.md",
                    "load/run-*/metrics.json", "load/run-*/metrics.md", "load/run-*/samples.jsonl")
        for variant in ("candidate", "baseline") if baseline_tag else ("candidate",):
            for pattern in patterns:
                for path in sorted((evidence / variant).glob(pattern)):
                    archive.write(path, path.relative_to(evidence).as_posix())


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--tag", required=True)
    parser.add_argument("--commit", required=True)
    parser.add_argument("--baseline-tag")
    parser.add_argument("--baseline-commit")
    parser.add_argument("--evidence", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    if bool(args.baseline_tag) != bool(args.baseline_commit):
        parser.error("baseline tag and commit must be supplied together")
    try:
        create_report(args.tag, args.commit, args.baseline_tag, args.baseline_commit, args.evidence, args.output)
    except (KeyError, OSError, TypeError, ValueError) as exc:
        parser.exit(1, f"release metrics: {exc}\n")


if __name__ == "__main__":
    main()
