#!/usr/bin/env python3
"""Summarize ArtificialFlow test/report artifacts for agentic quality gate review."""

from __future__ import annotations

import argparse
import json
from pathlib import Path
from typing import Any


def read_text(path: Path, limit: int = 12000) -> str:
    if not path.exists():
        return ""
    text = path.read_text(encoding="utf-8", errors="replace")
    return text[:limit]


def read_json(path: Path) -> Any:
    if not path.exists():
        return None
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except Exception:
        return None


def markdown_status(reports: Path) -> str:
    summary = read_text(reports / "summary.md")
    quality = read_text(reports / "agentic" / "quality-evidence.md")
    security = read_text(reports / "security.md", limit=6000)
    frontend = read_json(reports / "frontend-vitest.json")

    lines: list[str] = ["# Agentic Test Report Summary", ""]

    if quality:
        lines.extend(["## Changed-Path Evidence", "", quality.strip(), ""])
    else:
        lines.extend(["## Changed-Path Evidence", "", "- No agentic quality evidence found.", ""])

    if summary:
        lines.extend(["## Test-All Summary", "", summary.strip(), ""])
    else:
        lines.extend(["## Test-All Summary", "", "- `reports/summary.md` not found.", ""])

    if frontend:
        lines.extend(
            [
                "## Frontend Vitest",
                "",
                f"- Passed: {frontend.get('numPassedTests', 0)}",
                f"- Failed: {frontend.get('numFailedTests', 0)}",
                f"- Total: {frontend.get('numTotalTests', 0)}",
                "",
            ]
        )

    if security:
        lines.extend(["## Security Report Excerpt", "", security.strip(), ""])
    else:
        lines.extend(["## Security Report Excerpt", "", "- `reports/security.md` not found.", ""])

    artifact_candidates = [
        "unit.md",
        "coverage.txt",
        "integration.md",
        "e2e.md",
        "frontend.md",
        "performance.md",
        "security.md",
        "agentic/quality-evidence.md",
        "agentic/quality-evidence.json",
    ]
    existing = [candidate for candidate in artifact_candidates if (reports / candidate).exists()]
    lines.extend(["## Available Artifacts", ""])
    if existing:
        lines.extend(f"- `{reports / candidate}`" for candidate in existing)
    else:
        lines.append("- No known report artifacts found.")
    lines.append("")

    return "\n".join(lines)


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--reports", default="reports", help="Reports directory")
    parser.add_argument(
        "--output",
        default="reports/agentic/test-report-summary.md",
        help="Markdown summary output path",
    )
    args = parser.parse_args()

    reports = Path(args.reports)
    output = Path(args.output)
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text(markdown_status(reports), encoding="utf-8")
    print(f"[agentic] Wrote {output}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
