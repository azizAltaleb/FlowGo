#!/usr/bin/env bash
# test-security.sh — govulncheck + gosec + trivy image scan
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPORTS="${ROOT}/reports"
mkdir -p "${REPORTS}"

OVERALL=0

{
  echo "# Security Test Results"
  echo ""
  echo "Generated: $(date '+%Y-%m-%d %H:%M:%S')"
  echo ""
} > "${REPORTS}/security.md"

# ── 1. govulncheck ─────────────────────────────────────────────────────────

echo "[security] Running govulncheck..."
if ! command -v govulncheck &>/dev/null; then
  echo "  Installing govulncheck..."
  go install golang.org/x/vuln/cmd/govulncheck@latest 2>/dev/null
fi

{
  echo "## govulncheck (CVE scan — Go deps)"
  echo '```'
} >> "${REPORTS}/security.md"

set +e
govulncheck ./... 2>&1 | tee "${REPORTS}/govulncheck.txt"
VULN_EXIT=$?
set -e

cat "${REPORTS}/govulncheck.txt" >> "${REPORTS}/security.md"
{
  echo '```'
  if [[ $VULN_EXIT -eq 0 ]]; then
    echo "**Status: ✅ No vulnerabilities found**"
  else
    echo "**Status: ❌ Vulnerabilities found — review above**"
    OVERALL=1
  fi
  echo ""
} >> "${REPORTS}/security.md"

# ── 2. gosec ───────────────────────────────────────────────────────────────

echo "[security] Running gosec (SAST)..."
if ! command -v gosec &>/dev/null; then
  echo "  Installing gosec..."
  go install github.com/securego/gosec/v2/cmd/gosec@latest 2>/dev/null
fi

{
  echo "## gosec (Go SAST)"
  echo '```'
} >> "${REPORTS}/security.md"

set +e
gosec -fmt text -out "${REPORTS}/gosec.txt" \
  -exclude-dir=vendor \
  -exclude-generated \
  ./backend/... 2>&1 || true
GOSEC_EXIT=$?
set -e

cat "${REPORTS}/gosec.txt" >> "${REPORTS}/security.md" 2>/dev/null || true
HIGH_ISSUES=$(grep -c "Severity: HIGH" "${REPORTS}/gosec.txt" 2>/dev/null || echo 0)
MED_ISSUES=$(grep -c "Severity: MEDIUM" "${REPORTS}/gosec.txt" 2>/dev/null || echo 0)

{
  echo '```'
  echo "**Findings: HIGH=${HIGH_ISSUES}, MEDIUM=${MED_ISSUES}**"
  if [[ "${HIGH_ISSUES}" -gt 0 ]]; then
    echo "**Status: ❌ High severity issues found**"
    OVERALL=1
  elif [[ "${MED_ISSUES}" -gt 0 ]]; then
    echo "**Status: ⚠️  Medium severity issues — review recommended**"
  else
    echo "**Status: ✅ No high/medium issues**"
  fi
  echo ""
} >> "${REPORTS}/security.md"

# ── 3. trivy — container image scan ───────────────────────────────────────

echo "[security] Running trivy container scan..."

{
  echo "## trivy (Container Image Scan)"
  echo ""
} >> "${REPORTS}/security.md"

IMAGES=("workflowsa-app" "workflowsa-workflow-query" "workflowsa-workflow-runtime" "workflowsa-sync-worker")
TRIVY_CMD=""

if command -v trivy &>/dev/null; then
  TRIVY_CMD="trivy"
elif docker image ls grafana/trivy >/dev/null 2>&1 || true; then
  TRIVY_CMD="docker run --rm -v /var/run/docker.sock:/var/run/docker.sock aquasec/trivy:latest"
fi

if [[ -z "${TRIVY_CMD}" ]]; then
  echo "  trivy not found — skipping image scan (install with: brew install trivy)"
  echo "**trivy: ⏭ Skipped (not installed)**" >> "${REPORTS}/security.md"
else
  for img in "${IMAGES[@]}"; do
    echo "  Scanning ${img}..."
    {
      echo "### ${img}"
      echo '```'
    } >> "${REPORTS}/security.md"
    set +e
    ${TRIVY_CMD} image --exit-code 0 --severity HIGH,CRITICAL \
      --format table "${img}" 2>&1 | \
      tee -a "${REPORTS}/trivy-${img}.txt" >> "${REPORTS}/security.md" || true
    set -e
    {
      echo '```'
      echo ""
    } >> "${REPORTS}/security.md"
  done
fi

# ── final status ───────────────────────────────────────────────────────────

{
  echo "---"
  echo "_Security scan complete._"
} >> "${REPORTS}/security.md"

echo "[security] Done. Report: ${REPORTS}/security.md"
exit $OVERALL
