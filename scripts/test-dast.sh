#!/usr/bin/env bash
# DAST against a running ArtificialFlow gateway (OWASP ZAP baseline).
# Requires: docker, a healthy stack at TARGET_URL (default http://localhost:9100).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
REPORTS="${ROOT}/reports/dast"
mkdir -p "${REPORTS}"

TARGET_URL="${TARGET_URL:-http://localhost:9100}"
ALLOWLIST="${ROOT}/tests/security/dast-allowlist.txt"
ZAP_IMAGE="${ZAP_IMAGE:-ghcr.io/zaproxy/zaproxy:stable}"

if ! curl -fsS "${TARGET_URL}/api/health" >/dev/null 2>&1; then
  echo "ERROR: ${TARGET_URL}/api/health is not healthy. Start Compose first." >&2
  exit 1
fi

echo "[dast] Running ZAP baseline against ${TARGET_URL}"
# Baseline scan; do not fail the container on warnings — we parse the report.
docker run --rm --network host \
  -v "${REPORTS}:/zap/wrk:rw" \
  "${ZAP_IMAGE}" \
  zap-baseline.py -t "${TARGET_URL}" -r zap-baseline.html -J zap-baseline.json -I \
  || true

if [[ ! -f "${REPORTS}/zap-baseline.json" ]]; then
  echo "ERROR: ZAP did not produce zap-baseline.json" >&2
  exit 1
fi

python3 - <<'PY' "${REPORTS}/zap-baseline.json" "${ALLOWLIST}"
import json, sys
from pathlib import Path
report_path, allow_path = Path(sys.argv[1]), Path(sys.argv[2])
data = json.loads(report_path.read_text())
allowed = set()
if allow_path.exists():
    for line in allow_path.read_text().splitlines():
        line = line.strip()
        if not line or line.startswith("#"):
            continue
        allowed.add(line.split()[0])

high = []
for site in data.get("site", []):
    for alert in site.get("alerts", []):
        risk = str(alert.get("riskcode", alert.get("risk", "")))
        name = alert.get("alert") or alert.get("name") or ""
        plugin = str(alert.get("pluginid") or alert.get("pluginId") or "")
        if risk in ("3", "High") and plugin not in allowed and name not in allowed:
            high.append({"name": name, "pluginid": plugin, "count": alert.get("count")})

if high:
    print("DAST FAIL: high-risk alerts not allowlisted:")
    for h in high:
        print(" -", h)
    sys.exit(1)
print("DAST OK: no unallowlisted high-risk alerts")
print("Report:", report_path.with_suffix(".html"))
PY
