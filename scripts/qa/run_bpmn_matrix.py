#!/usr/bin/env python3
"""Run and report the FlowGo BPMN scenario matrix.

The catalog intentionally uses a small YAML subset so the runner can avoid
external Python dependencies.
"""

from __future__ import annotations

import argparse
import datetime as dt
import hashlib
import json
import subprocess
import time
from pathlib import Path
from typing import Any


def parse_scalar(value: str) -> Any:
    value = value.strip()
    if value == "":
        return ""
    if value in ("true", "false"):
        return value == "true"
    if "," in value and not (value.startswith("'") or value.startswith('"')):
        return [item.strip() for item in value.split(",") if item.strip()]
    if (value.startswith('"') and value.endswith('"')) or (value.startswith("'") and value.endswith("'")):
        return value[1:-1]
    return value


def load_scenarios(path: Path) -> list[dict[str, Any]]:
    scenarios: list[dict[str, Any]] = []
    current: dict[str, Any] | None = None
    for raw_line in path.read_text(encoding="utf-8").splitlines():
        line = raw_line.rstrip()
        stripped = line.strip()
        if not stripped or stripped.startswith("#") or stripped == "scenarios:":
            continue
        if stripped.startswith("- "):
            if current:
                scenarios.append(current)
            current = {}
            stripped = stripped[2:].strip()
            if stripped:
                key, _, value = stripped.partition(":")
                current[key.strip()] = parse_scalar(value)
            continue
        if current is not None and ":" in stripped:
            key, _, value = stripped.partition(":")
            current[key.strip()] = parse_scalar(value)
    if current:
        scenarios.append(current)
    return scenarios


def run_command(command: str, cwd: Path, artifact: Path) -> dict[str, Any]:
    started = dt.datetime.now(dt.timezone.utc).isoformat(timespec="seconds")
    start = time.monotonic()
    artifact.parent.mkdir(parents=True, exist_ok=True)
    proc = subprocess.run(
        command,
        shell=True,
        cwd=str(cwd),
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
    )
    duration_ms = int((time.monotonic() - start) * 1000)
    artifact.write_text(proc.stdout, encoding="utf-8", errors="replace")
    return {
        "command": command,
        "exit_code": proc.returncode,
        "status": "pass" if proc.returncode == 0 else "fail",
        "duration_ms": duration_ms,
        "started_at": started,
        "ended_at": dt.datetime.now(dt.timezone.utc).isoformat(timespec="seconds"),
        "artifact": str(artifact),
        "failure_excerpt": "" if proc.returncode == 0 else proc.stdout[-4000:],
    }


def scenario_status(scenario: dict[str, Any], command_results: dict[str, dict[str, Any]]) -> dict[str, Any]:
    command = str(scenario.get("command") or "").strip()
    if not command:
        return {
            "status": "warn",
            "exit_code": None,
            "duration_ms": 0,
            "artifacts": [],
            "failure_excerpt": "",
            "skip_reason": "No automated command is mapped yet; scenario is tracked as a coverage gap.",
        }
    result = command_results[command]
    return {
        "status": result["status"],
        "exit_code": result["exit_code"],
        "duration_ms": result["duration_ms"],
        "artifacts": [result["artifact"]],
        "failure_excerpt": result["failure_excerpt"],
        "skip_reason": "",
    }


def write_markdown(report: dict[str, Any], path: Path) -> None:
    lines = [
        "# BPMN Matrix Report",
        "",
        f"- Overall status: **{report['overall']['status'].upper()}**",
        f"- Scenarios: {len(report['scenarios'])}",
        "",
        "| Scenario | Category | Support | Status |",
        "| :--- | :--- | :--- | :--- |",
    ]
    for scenario in report["scenarios"]:
        lines.append(
            f"| `{scenario['id']}` | {scenario.get('category', '')} | "
            f"{scenario.get('support_status', '')} | {scenario['status'].upper()} |"
        )
    lines.extend(["", "## Coverage Gaps", ""])
    gaps = [s for s in report["scenarios"] if s["status"] in ("skip", "warn") and not str(s.get("command") or "").strip()]
    if gaps:
        for scenario in gaps:
            lines.append(f"- `{scenario['id']}`: {scenario.get('behavior', '')}")
    else:
        lines.append("- No uncovered scenarios recorded.")
    path.write_text("\n".join(lines) + "\n", encoding="utf-8")


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--catalog", default="tests/bpmn/matrix/scenarios.yml")
    parser.add_argument("--reports-dir", default="reports")
    parser.add_argument("--output-json", default="reports/bpmn-matrix-report.json")
    parser.add_argument("--output-md", default="reports/bpmn-matrix-report.md")
    parser.add_argument("--dry-run", action="store_true", help="Do not execute commands; mark executable scenarios as skipped.")
    args = parser.parse_args()

    cwd = Path.cwd()
    reports_dir = Path(args.reports_dir)
    command_artifacts = reports_dir / "bpmn-matrix"
    scenarios = load_scenarios(Path(args.catalog))
    command_results: dict[str, dict[str, Any]] = {}

    if not args.dry_run:
        for command in sorted({str(s.get("command") or "").strip() for s in scenarios if str(s.get("command") or "").strip()}):
            artifact_id = hashlib.sha256(command.encode("utf-8")).hexdigest()[:16]
            artifact = command_artifacts / f"{artifact_id}.txt"
            command_results[command] = run_command(command, cwd, artifact)

    normalized: list[dict[str, Any]] = []
    for scenario in scenarios:
        command = str(scenario.get("command") or "").strip()
        if args.dry_run and command:
            status = {
                "status": "skip",
                "exit_code": None,
                "duration_ms": 0,
                "artifacts": [],
                "failure_excerpt": "",
                "skip_reason": "Dry run requested.",
            }
        else:
            status = scenario_status(scenario, command_results)
        normalized.append({**scenario, **status, "command": command})

    required_failures = [s for s in normalized if s.get("required") and s["status"] == "fail"]
    status = "fail" if required_failures else ("warn" if any(s["status"] in ("skip", "warn") for s in normalized) else "pass")
    report = {
        "schema_version": "1.0",
        "generated_at": dt.datetime.now(dt.timezone.utc).isoformat(timespec="seconds"),
        "catalog": args.catalog,
        "overall": {
            "status": status,
            "required_failed": len(required_failures),
            "skipped": sum(1 for s in normalized if s["status"] == "skip"),
        },
        "scenarios": normalized,
    }

    output_json = Path(args.output_json)
    output_md = Path(args.output_md)
    output_json.parent.mkdir(parents=True, exist_ok=True)
    output_json.write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
    write_markdown(report, output_md)
    print(f"[bpmn-matrix] Wrote {output_json}")
    print(f"[bpmn-matrix] Wrote {output_md}")
    return 0 if status != "fail" else 1


if __name__ == "__main__":
    raise SystemExit(main())
