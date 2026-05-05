#!/usr/bin/env bash
# test-report.sh — Generate per-layer detail reports from raw test output.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPORTS="${ROOT}/reports"
mkdir -p "${REPORTS}"

# ── unit.md ────────────────────────────────────────────────────────────────

if [[ -f "${REPORTS}/coverage.txt" ]]; then
  {
    echo "# Unit Test Coverage Report"
    echo ""
    echo "Generated: $(date '+%Y-%m-%d %H:%M:%S')"
    echo ""
    echo "| Package | Coverage |"
    echo "|---|---|"
    grep -v "^total" "${REPORTS}/coverage.txt" | \
      awk '{printf "| %s | %s |\n", $1, $NF}' || true
    echo ""
    TOTAL=$(grep "^total" "${REPORTS}/coverage.txt" | awk '{print $NF}' || echo "n/a")
    echo "**Total: ${TOTAL}**"
  } > "${REPORTS}/unit.md"
fi

# ── integration.md ─────────────────────────────────────────────────────────

if [[ -f "${REPORTS}/integration-raw.json" ]]; then
  {
    echo "# Integration Test Results"
    echo ""
    echo "Generated: $(date '+%Y-%m-%d %H:%M:%S')"
    echo ""
    echo "| Test | Status | Duration |"
    echo "|---|---|---|"
    python3 - "${REPORTS}/integration-raw.json" <<'PY'
import json, sys
tests = {}
with open(sys.argv[1]) as f:
    for line in f:
        line = line.strip()
        if not line:
            continue
        try:
            d = json.loads(line)
        except Exception:
            continue
        name = d.get("Test")
        action = d.get("Action")
        elapsed = d.get("Elapsed", 0)
        if not name:
            continue
        if action in ("pass", "fail"):
            icon = "✅ PASS" if action == "pass" else "❌ FAIL"
            print(f"| {name} | {icon} | {elapsed:.2f}s |")
PY
  } > "${REPORTS}/integration.md"
fi

# ── performance.md ─────────────────────────────────────────────────────────

if [[ -f "${REPORTS}/perf-raw.txt" ]]; then
  {
    echo "# Performance Test Results (k6)"
    echo ""
    echo "Generated: $(date '+%Y-%m-%d %H:%M:%S')"
    echo ""
    echo '```'
    grep -E "(✓|✗|http_req_duration|http_reqs|checks|vus)" "${REPORTS}/perf-raw.txt" 2>/dev/null || \
      tail -50 "${REPORTS}/perf-raw.txt"
    echo '```'
  } > "${REPORTS}/performance.md"
fi

# ── frontend.md ────────────────────────────────────────────────────────────

{
  echo "# Frontend Test Results"
  echo ""
  echo "Generated: $(date '+%Y-%m-%d %H:%M:%S')"
  echo ""
} > "${REPORTS}/frontend.md"

if [[ -f "${REPORTS}/frontend-vitest.json" ]]; then
  python3 - "${REPORTS}/frontend-vitest.json" >> "${REPORTS}/frontend.md" <<'PY'
import json, sys
d = json.load(open(sys.argv[1]))
print("## Vitest (Unit)")
print("")
print(f"- Passed: {d.get('numPassedTests',0)}")
print(f"- Failed: {d.get('numFailedTests',0)}")
print(f"- Total:  {d.get('numTotalTests',0)}")
print("")
for suite in d.get("testResults", []):
    status = "✅" if suite.get("status") == "passed" else "❌"
    name = suite.get("testFilePath","").split("/")[-1]
    print(f"### {status} {name}")
    for t in suite.get("testResults", []):
        icon = "✅" if t.get("status") == "passed" else "❌"
        print(f"- {icon} {t.get('fullName','')}")
    print("")
PY
fi

if [[ -f "${REPORTS}/playwright-raw.txt" ]]; then
  {
    echo "## Playwright (Browser E2E)"
    echo ""
    grep -E "(passed|failed|PASS|FAIL|✓|✘|×)" "${REPORTS}/playwright-raw.txt" 2>/dev/null | head -40 || \
      tail -20 "${REPORTS}/playwright-raw.txt"
  } >> "${REPORTS}/frontend.md"
fi

# ── append links to summary.md ─────────────────────────────────────────────

if [[ -f "${REPORTS}/summary.md" ]]; then
  {
    echo ""
    echo "## Detailed Reports"
    echo ""
    echo "- [Unit Coverage](unit.md)"
    echo "- [Integration](integration.md)"
    echo "- [E2E Backend](e2e.md)"
    echo "- [Frontend](frontend.md)"
    echo "- [Performance](performance.md)"
    echo "- [Security](security.md)"
  } >> "${REPORTS}/summary.md"
fi

echo "[report] Reports written to ${REPORTS}/"
