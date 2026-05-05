#!/usr/bin/env bash
# test-e2e.sh — Run all backend E2E scripts and capture results.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPORTS="${ROOT}/reports"
mkdir -p "${REPORTS}"

PASS=0
FAIL=0
DETAILS=()

run_script() {
  local name="$1"
  local script="$2"
  shift 2
  local start; start="$(date +%s)"
  local out_file="${REPORTS}/e2e-${name}.txt"

  echo "[e2e] Running ${name}..."
  if bash "${script}" "$@" >"${out_file}" 2>&1; then
    local dur=$(( $(date +%s) - start ))
    echo "  ✅ ${name} passed (${dur}s)"
    DETAILS+=("| ${name} | ✅ PASS | ${dur}s |")
    PASS=$(( PASS + 1 ))
  else
    local dur=$(( $(date +%s) - start ))
    echo "  ❌ ${name} FAILED (${dur}s) — see ${out_file}"
    DETAILS+=("| ${name} | ❌ FAIL | ${dur}s |")
    FAIL=$(( FAIL + 1 ))
  fi
}

# ── Run each E2E script ────────────────────────────────────────────────────

run_script "cqrs-e2e-smoke"       "${ROOT}/scripts/cqrs_e2e_smoke.sh"
run_script "worker-conformance"   "${ROOT}/scripts/worker_conformance_smoke.sh"
run_script "cqrs-parity-check"    "${ROOT}/scripts/cqrs_parity_check.sh"

# ── Write e2e.md ───────────────────────────────────────────────────────────

{
  echo "# E2E Backend Test Results"
  echo ""
  echo "| Script | Status | Duration |"
  echo "|---|---|---|"
  for row in "${DETAILS[@]}"; do
    echo "${row}"
  done
  echo ""
  echo "_Passed: ${PASS} | Failed: ${FAIL}_"
} > "${REPORTS}/e2e.md"

echo "[e2e] Done — ${PASS} passed, ${FAIL} failed"

if [[ $FAIL -gt 0 ]]; then
  exit 1
fi
