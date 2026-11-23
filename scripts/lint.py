"""Check Antifraud's repository invariants without modifying source files."""

import argparse
from collections import Counter
import json
from pathlib import Path
import re
import subprocess
import sys


ROOT = Path(__file__).resolve().parent.parent
ACTION_USE = re.compile(r"^[ \t]*(?:-[ \t]*)?uses:[ \t]*([^\s#]+)", re.MULTILINE)
PINNED_ACTION = re.compile(r"[^@]+@[a-f0-9]{40}")
REQUIREMENT_ROW = re.compile(r"^\|\s*(REQ-[^|\s]+)\s*\|", re.MULTILINE)


def run(*args):
    return subprocess.check_output(args, cwd=ROOT, encoding="utf-8")


def read_json(path):
    return json.loads((ROOT / path).read_text(encoding="utf-8"))


def check_go_format():
    tracked_and_new = run(
        "git", "ls-files", "-z", "--cached", "--others", "--exclude-standard", "--", "*.go"
    )
    paths = sorted({name for name in tracked_and_new.split("\0") if name and (ROOT / name).is_file()})
    if not paths:
        return []
    goroot = Path(run("go", "env", "GOROOT").strip())
    formatter = goroot / "bin" / ("gofmt.exe" if sys.platform == "win32" else "gofmt")
    errors = []
    for path in paths:
        source = (ROOT / path).read_bytes().replace(b"\r\n", b"\n")
        result = subprocess.run([formatter], input=source, capture_output=True, cwd=ROOT)
        if result.returncode:
            detail = result.stderr.decode("utf-8", errors="replace").strip()
            errors.append(f"{path}: gofmt failed\n{detail}")
        elif result.stdout != source:
            errors.append(f"{path}: run gofmt -w on this file")
    return errors


def check_contracts():
    errors = []
    for path in ("contracts/external.json", "contracts/internal.json"):
        contract = read_json(path)
        schemas = contract.get("components", {}).get("schemas", {})
        for name, prefixes in (
            ("Evaluation", {"accountId": "demo-", "target": "sink-"}),
            ("Confirmation", {"accountId": "demo-"}),
        ):
            schema = schemas.get(name, {})
            if schema.get("additionalProperties") is not False:
                errors.append(f"{path}: {name} request must declare additionalProperties: false")
            for field, prefix in prefixes.items():
                domain = schema.get("properties", {}).get(field, {})
                errors.extend(check_synthetic_enum(path, f"{name}.{field}", domain, prefix))
        state = contract.get("paths", {}).get("/v1/state/{account}", {}).get("get", {})
        account = next(
            (param.get("schema", {}) for param in state.get("parameters", [])
             if param.get("name") == "account" and param.get("in") == "path"),
            {},
        )
        errors.extend(check_synthetic_enum(path, "State.account", account, "demo-"))
    return errors


def check_synthetic_enum(path, field, schema, prefix):
    values = schema.get("enum", [])
    pattern = re.compile(re.escape(prefix) + r"[a-z][a-z0-9]*(?:-[a-z0-9]+)*")
    if not isinstance(values, list) or not values or any(
        not isinstance(value, str) or not pattern.fullmatch(value) for value in values
    ):
        return [f"{path}: {field} must enumerate only synthetic {prefix}* values in scripts/contracts.py"]
    return []


def check_traceability():
    requirements = Counter(row["id"] for row in read_json("docs/sources/requirements.json"))
    path = "docs/traceability.md"
    indexed = Counter(REQUIREMENT_ROW.findall((ROOT / path).read_text(encoding="utf-8")))
    errors = []
    for label, duplicate_ids in (
        ("docs/sources/requirements.json", requirements),
        (path, indexed),
    ):
        errors.extend(
            f"{label}: duplicate requirement {requirement}"
            for requirement, count in sorted(duplicate_ids.items())
            if count > 1
        )
    errors.extend(f"{path}: missing {requirement}; run python scripts/traceability.py"
                  for requirement in sorted(requirements.keys() - indexed.keys()))
    errors.extend(f"{path}: unknown {requirement}; restore the ID from the source inventory"
                  for requirement in sorted(indexed.keys() - requirements.keys()))
    return errors


def check_action_pins():
    errors = []
    for path in sorted((ROOT / ".github/workflows").glob("*")):
        if path.suffix not in (".yml", ".yaml"):
            continue
        for reference in ACTION_USE.findall(path.read_text(encoding="utf-8")):
            reference = reference.strip("'\"")
            if reference.startswith("./"):
                continue
            if not PINNED_ACTION.fullmatch(reference):
                errors.append(f"{path.relative_to(ROOT).as_posix()}: pin {reference} to a 40-character commit SHA")
    return errors


def check_compose():
    config = json.loads(run("docker", "compose", "config", "--format", "json"))
    errors = []
    for name, service in sorted(config["services"].items()):
        network = service.get("network_mode", "")
        if network == "host":
            errors.append(f"compose.yaml: {name} must use an isolated network")
        ports = service.get("ports", [])
        if ports and name not in {"gateway", "postgres"}:
            errors.append(f"compose.yaml: {name} must not publish host ports")
        for port in ports:
            if port.get("host_ip") not in {"127.0.0.1", "::1"}:
                errors.append(f"compose.yaml: bind {name}'s published ports to loopback")
    return errors


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--compose", action="store_true", help="check the resolved Docker Compose isolation")
    args = parser.parse_args()
    checks = [check_compose] if args.compose else [
        check_go_format, check_contracts, check_traceability, check_action_pins,
    ]
    errors = []
    for check in checks:
        try:
            errors.extend(check())
        except (OSError, subprocess.CalledProcessError, ValueError, KeyError, TypeError) as error:
            errors.append(f"{check.__name__}: {error}")
    if errors:
        for error in errors:
            print(error, file=sys.stderr)
        return 1
    print("Compose isolation checks passed" if args.compose else "Repository lint checks passed")
    return 0


if __name__ == "__main__":
    sys.exit(main())
