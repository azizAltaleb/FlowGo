#!/usr/bin/env python3
"""Build the ArtificialFlow all-functionality QA report.

The orchestrator writes one JSON object per line to an events file. This
script normalizes those events into a durable JSON report and a concise
Markdown summary that humans and agents can review.
"""

from __future__ import annotations

import argparse
import datetime as dt
import json
import os
import platform
import subprocess
from pathlib import Path
from typing import Any


STATUSES = ("pass", "fail", "warn", "skip", "flaky")


def utc_now() -> str:
    return dt.datetime.now(dt.timezone.utc).isoformat(timespec="seconds")


def read_json(path: Path) -> Any:
    if not path.exists():
        return None
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except Exception:
        return None


def read_jsonl(path: Path) -> list[dict[str, Any]]:
    if not path.exists():
        return []
    events: list[dict[str, Any]] = []
    for line in path.read_text(encoding="utf-8", errors="replace").splitlines():
        if not line.strip():
            continue
        try:
            event = json.loads(line)
        except json.JSONDecodeError:
            event = {
                "id": "malformed-event",
                "name": "Malformed event",
                "surface": "reporting",
                "status": "warn",
                "required": False,
                "failure_excerpt": line[:500],
            }
        events.append(event)
    return events


def shell_value(command: list[str]) -> str:
    try:
        return subprocess.check_output(command, text=True, stderr=subprocess.DEVNULL).strip()
    except Exception:
        return ""


def git_metadata(repo: Path) -> dict[str, Any]:
    return {
        "sha": shell_value(["git", "-C", str(repo), "rev-parse", "HEAD"]),
        "branch": shell_value(["git", "-C", str(repo), "branch", "--show-current"]),
        "dirty": bool(shell_value(["git", "-C", str(repo), "status", "--porcelain"])),
    }


def tool_versions() -> dict[str, str]:
    return {
        "go": shell_value(["go", "version"]),
        "node": shell_value(["node", "--version"]),
        "npm": shell_value(["npm", "--version"]),
        "docker": shell_value(["docker", "--version"]),
        "helm": shell_value(["helm", "version", "--short"]),
        "kubectl": shell_value(["kubectl", "version", "--client=true", "--short"]),
    }


def normalize_event(raw: dict[str, Any]) -> dict[str, Any]:
    status = str(raw.get("status") or "warn").lower()
    if status not in STATUSES:
        status = "warn"
    return {
        "id": raw.get("id", "unknown"),
        "name": raw.get("name", raw.get("id", "unknown")),
        "surface": raw.get("surface", "unknown"),
        "deployment_model": raw.get("deployment_model"),
        "status": status,
        "required": bool(raw.get("required", False)),
        "command": raw.get("command", ""),
        "exit_code": raw.get("exit_code"),
        "duration_ms": int(raw.get("duration_ms") or 0),
        "artifacts": raw.get("artifacts", []),
        "summary": raw.get("summary", {}),
        "failure_excerpt": raw.get("failure_excerpt", ""),
        "skip_reason": raw.get("skip_reason", ""),
        "started_at": raw.get("started_at", ""),
        "ended_at": raw.get("ended_at", ""),
    }


def status_for_events(events: list[dict[str, Any]]) -> str:
    required_failed = [e for e in events if e["required"] and e["status"] == "fail"]
    if required_failed:
        return "fail"
    if any(e["status"] == "fail" for e in events):
        return "warn"
    if any(e["status"] in ("warn", "skip", "flaky") for e in events):
        return "warn"
    return "pass"


def build_functionality_matrix(events: list[dict[str, Any]]) -> list[dict[str, Any]]:
    grouped: dict[str, list[dict[str, Any]]] = {}
    for event in events:
        grouped.setdefault(event["surface"], []).append(event)

    matrix: list[dict[str, Any]] = []
    for surface, surface_events in sorted(grouped.items()):
        matrix.append(
            {
                "surface": surface,
                "status": status_for_events(surface_events),
                "evidence": [
                    {
                        "id": event["id"],
                        "deployment_model": event.get("deployment_model"),
                        "status": event["status"],
                        "required": event["required"],
                        "artifacts": event["artifacts"],
                    }
                    for event in surface_events
                ],
                "gaps": [
                    event.get("skip_reason") or event.get("failure_excerpt")
                    for event in surface_events
                    if event["status"] in ("skip", "warn", "fail")
                    and (event.get("skip_reason") or event.get("failure_excerpt"))
                ],
            }
        )
    return matrix


def artifact_manifest(reports_dir: Path) -> list[str]:
    if not reports_dir.exists():
        return []
    return sorted(
        str(path.relative_to(reports_dir.parent))
        for path in reports_dir.rglob("*")
        if path.is_file()
    )


def build_report(args: argparse.Namespace) -> dict[str, Any]:
    repo = Path(args.repo).resolve()
    reports_dir = Path(args.reports_dir).resolve()
    raw_events = read_jsonl(Path(args.events))
    events = [normalize_event(event) for event in raw_events]
    bpmn_report = read_json(Path(args.bpmn_report)) if args.bpmn_report else None
    if bpmn_report:
        for scenario in bpmn_report.get("scenarios", []):
            events.append(
                normalize_event(
                    {
                        "id": f"bpmn:{scenario.get('id')}",
                        "name": scenario.get("name"),
                        "surface": "bpmn",
                        "status": scenario.get("status", "warn"),
                        "required": scenario.get("required", False),
                        "command": scenario.get("command", ""),
                        "duration_ms": scenario.get("duration_ms", 0),
                        "artifacts": scenario.get("artifacts", []),
                        "failure_excerpt": scenario.get("failure_excerpt", ""),
                        "skip_reason": scenario.get("skip_reason", ""),
                    }
                )
            )

    status_counts = {status: sum(1 for e in events if e["status"] == status) for status in STATUSES}
    required_failed = sum(1 for e in events if e["required"] and e["status"] == "fail")
    overall_status = status_for_events(events)

    return {
        "schema_version": "1.0",
        "run": {
            "id": os.environ.get("GITHUB_RUN_ID") or args.run_id or dt.datetime.now().strftime("%Y%m%d%H%M%S"),
            "started_at": args.started_at,
            "ended_at": utc_now(),
            "repo": str(repo),
            "git": git_metadata(repo),
            "environment": {
                "ci": bool(os.environ.get("CI")),
                "os": platform.platform(),
                "tools": tool_versions(),
            },
            "flags": json.loads(args.flags_json or "{}"),
        },
        "overall": {
            "status": overall_status,
            "required_failed": required_failed,
            "status_counts": status_counts,
        },
        "layers": events,
        "functionality_matrix": build_functionality_matrix(events),
        "bpmn": bpmn_report,
        "artifacts": artifact_manifest(reports_dir),
        "residual_risks": [
            e.get("skip_reason") or e.get("failure_excerpt")
            for e in events
            if e["status"] in ("skip", "warn", "fail")
            and (e.get("skip_reason") or e.get("failure_excerpt"))
        ],
        "recommendations": [
            "Review failed required layers before release or merge.",
            "Review skipped live deployment/IAM/UI/performance layers before claiming exhaustive coverage.",
        ],
    }


def md_status(status: str) -> str:
    return {
        "pass": "PASS",
        "fail": "FAIL",
        "warn": "WARN",
        "skip": "SKIP",
        "flaky": "FLAKY",
    }.get(status, status.upper())


def event_label(event: dict[str, Any]) -> str:
    label = f"`{event['id']}`"
    if event.get("deployment_model"):
        label += f" [{event['deployment_model']}]"
    return label


def write_markdown(report: dict[str, Any], path: Path) -> None:
    lines: list[str] = ["# All Functionality Test Report", ""]
    overall = report["overall"]
    lines.extend(
        [
            f"- Overall status: **{md_status(overall['status'])}**",
            f"- Required failures: {overall['required_failed']}",
            f"- Git branch: `{report['run']['git']['branch']}`",
            f"- Git SHA: `{report['run']['git']['sha']}`",
            f"- Dirty working tree: `{report['run']['git']['dirty']}`",
            "",
            "## Status Counts",
            "",
        ]
    )
    for status in STATUSES:
        lines.append(f"- {md_status(status)}: {overall['status_counts'].get(status, 0)}")

    lines.extend(["", "## Functionality Matrix", "", "| Surface | Status | Evidence |", "| :--- | :--- | :--- |"])
    for row in report["functionality_matrix"]:
        evidence = ", ".join(f"{event_label(item)} ({item['status']})" for item in row["evidence"][:8])
        if len(row["evidence"]) > 8:
            evidence += f", +{len(row['evidence']) - 8} more"
        lines.append(f"| {row['surface']} | {md_status(row['status'])} | {evidence or 'none'} |")

    failures = [e for e in report["layers"] if e["status"] == "fail"]
    skips = [e for e in report["layers"] if e["status"] == "skip"]
    warnings = [e for e in report["layers"] if e["status"] == "warn"]

    lines.extend(["", "## Failed Tests And Diagnostics", ""])
    if failures:
        for event in failures:
            lines.append(f"### {event['id']} - {event['name']}")
            lines.append("")
            lines.append(f"- Surface: `{event['surface']}`")
            if event.get("deployment_model"):
                lines.append(f"- Deployment model: `{event['deployment_model']}`")
            lines.append(f"- Required: `{event['required']}`")
            lines.append(f"- Command: `{event['command']}`")
            if event.get("failure_excerpt"):
                lines.append("")
                lines.append("```text")
                lines.append(str(event["failure_excerpt"])[:4000])
                lines.append("```")
            lines.append("")
    else:
        lines.append("- No failed layers recorded.")

    lines.extend(["", "## Skipped Tests With Reasons", ""])
    if skips:
        for event in skips:
            lines.append(f"- {event_label(event)}: {event.get('skip_reason') or 'No reason recorded.'}")
    else:
        lines.append("- No skipped layers recorded.")

    lines.extend(["", "## Warnings", ""])
    if warnings:
        for event in warnings:
            lines.append(f"- {event_label(event)}: {event.get('failure_excerpt') or event.get('skip_reason') or 'Review recommended.'}")
    else:
        lines.append("- No warnings recorded.")

    lines.extend(["", "## Artifacts", ""])
    if report["artifacts"]:
        for artifact in report["artifacts"]:
            lines.append(f"- `{artifact}`")
    else:
        lines.append("- No artifacts found.")

    path.write_text("\n".join(lines) + "\n", encoding="utf-8")


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repo", default=".")
    parser.add_argument("--reports-dir", default="reports")
    parser.add_argument("--events", default="reports/all-functionality/events.jsonl")
    parser.add_argument("--bpmn-report", default="reports/bpmn-matrix-report.json")
    parser.add_argument("--output-json", default="reports/all-functionality-report.json")
    parser.add_argument("--output-md", default="reports/all-functionality-report.md")
    parser.add_argument("--run-id", default="")
    parser.add_argument("--started-at", default=utc_now())
    parser.add_argument("--flags-json", default="{}")
    args = parser.parse_args()

    report = build_report(args)
    output_json = Path(args.output_json)
    output_md = Path(args.output_md)
    output_json.parent.mkdir(parents=True, exist_ok=True)
    output_json.write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
    write_markdown(report, output_md)
    print(f"[report] Wrote {output_json}")
    print(f"[report] Wrote {output_md}")
    return 0 if report["overall"]["status"] != "fail" else 1


if __name__ == "__main__":
    raise SystemExit(main())
