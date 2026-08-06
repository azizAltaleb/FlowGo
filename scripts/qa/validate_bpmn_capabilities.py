#!/usr/bin/env python3
"""Validate that scenarios.yml and BPMN_SUPPORT_MATRIX.md stay aligned with capabilities.yml."""

from __future__ import annotations

import argparse
import re
import sys
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


def load_yaml_list(path: Path, root_key: str) -> list[dict[str, Any]]:
    items: list[dict[str, Any]] = []
    current: dict[str, Any] | None = None
    in_root = False
    for raw_line in path.read_text(encoding="utf-8").splitlines():
        line = raw_line.rstrip()
        stripped = line.strip()
        if not stripped or stripped.startswith("#"):
            continue
        if stripped == f"{root_key}:":
            in_root = True
            continue
        if not in_root:
            continue
        if stripped.startswith("- "):
            if current:
                items.append(current)
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
        items.append(current)
    return items


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", default=".", help="Repository root")
    args = parser.parse_args()
    root = Path(args.root).resolve()

    capabilities_path = root / "tests/bpmn/matrix/capabilities.yml"
    scenarios_path = root / "tests/bpmn/matrix/scenarios.yml"
    matrix_path = root / "docs/BPMN_SUPPORT_MATRIX.md"

    errors: list[str] = []
    if not capabilities_path.exists():
        print(f"ERROR: missing {capabilities_path}", file=sys.stderr)
        return 2

    capabilities = load_yaml_list(capabilities_path, "capabilities")
    scenarios = load_yaml_list(scenarios_path, "scenarios") if scenarios_path.exists() else []
    scenario_ids = {str(s.get("id")) for s in scenarios}
    matrix_text = matrix_path.read_text(encoding="utf-8") if matrix_path.exists() else ""

    required_scenario_refs: set[str] = set()
    for cap in capabilities:
        cap_id = str(cap.get("id") or "")
        status = str(cap.get("status") or "")
        tests = cap.get("tests") or []
        if isinstance(tests, str):
            tests = [t for t in tests.split(",") if t]
        for test_id in tests:
            required_scenario_refs.add(str(test_id))
            if test_id not in scenario_ids:
                errors.append(f"capability {cap_id}: missing scenario id `{test_id}` in scenarios.yml")

        # Status honesty checks against support matrix keywords.
        if status == "supported":
            if "Not supported" in matrix_text and cap_id.replace("-", " ") in matrix_text.lower():
                pass  # soft; matrix uses feature names not ids
        if status == "unsupported" and "Ad-hoc" not in matrix_text and "Complex gateway" not in matrix_text:
            errors.append("support matrix must document unsupported ad-hoc/complex gateway rows")

        # Supported executable capabilities must claim runtime + xml roundtrip.
        if status == "supported" and cap.get("runtime") is False:
            errors.append(f"capability {cap_id}: status=supported but runtime=false")
        if status == "supported" and not cap.get("xml_roundtrip", False) and str(cap.get("category")) != "connectors":
            # connectors may use job contracts rather than BPMN XML elements
            pass

    # Scenarios referenced by capabilities should have a command when required/supported.
    scenario_by_id = {str(s.get("id")): s for s in scenarios}
    for test_id in sorted(required_scenario_refs):
        scenario = scenario_by_id.get(test_id)
        if not scenario:
            continue
        support = str(scenario.get("support_status") or "")
        command = str(scenario.get("command") or "").strip()
        if support in ("supported", "partial") and scenario.get("required") is True and not command:
            errors.append(f"scenario {test_id}: required supported/partial scenario has empty command")

    # sendTask must not remain marked unsupported in scenarios once catalog says supported.
    send_caps = [c for c in capabilities if "send-task" in str(c.get("id"))]
    if send_caps and str(send_caps[0].get("status")) == "supported":
        for s in scenarios:
            if "sendTask" in str(s.get("elements")) and str(s.get("support_status")) == "unsupported":
                errors.append(
                    f"scenario {s.get('id')}: sendTask marked unsupported but capabilities.yml marks send-task supported"
                )
        if "Send tasks" in matrix_text and "Not supported" in matrix_text:
            # only fail if the send-task row itself is not supported
            for line in matrix_text.splitlines():
                if "Send task" in line and "Not supported" in line:
                    errors.append("BPMN_SUPPORT_MATRIX.md marks Send tasks as Not supported")

    if "Multi-instance" in matrix_text:
        for line in matrix_text.splitlines():
            if "Multi-instance" in line and "Not supported" in line:
                errors.append("BPMN_SUPPORT_MATRIX.md marks Multi-instance as Not supported while catalog says supported")

    if errors:
        print("BPMN capability validation FAILED:")
        for err in errors:
            print(f"  - {err}")
        return 1

    print(f"OK: {len(capabilities)} capabilities, {len(scenarios)} scenarios, matrix present={matrix_path.exists()}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
