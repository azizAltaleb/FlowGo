#!/usr/bin/env bash
set -euo pipefail

WORKER_API_BASE_URL="${WORKER_API_BASE_URL:-http://localhost:8080}"
WORKER_PROTOCOL_VERSION="${WORKER_PROTOCOL_VERSION:-v1}"
WORKER_AUTH_BEARER_TOKEN="${WORKER_AUTH_BEARER_TOKEN:-}"

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

curl_with_context() {
  local method="$1"
  local path="$2"
  local body_file="$3"
  local response_body_file="$4"
  local response_header_file="$5"
  shift 5

  local url="${WORKER_API_BASE_URL}${path}"
  local auth_args=()
  if [[ -n "${WORKER_AUTH_BEARER_TOKEN}" ]]; then
    auth_args=(-H "Authorization: Bearer ${WORKER_AUTH_BEARER_TOKEN}")
  fi

  local curl_args=(
    -sS
    -X "${method}"
    -o "${response_body_file}"
    -D "${response_header_file}"
    -w "%{http_code}"
    "${auth_args[@]}"
    "$@"
    "${url}"
  )

  if [[ -n "${body_file}" ]]; then
    curl_args+=(--data-binary "@${body_file}")
  fi

  curl "${curl_args[@]}"
}

require_status() {
  local actual="$1"
  local expected="$2"
  local context="$3"
  local body_file="$4"

  if [[ "${actual}" != "${expected}" ]]; then
    echo "[FAIL] ${context}: expected HTTP ${expected}, got ${actual}" >&2
    echo "--- response body ---" >&2
    cat "${body_file}" >&2 || true
    echo >&2
    exit 1
  fi
}

require_body_contains() {
  local body_file="$1"
  local needle="$2"
  local context="$3"

  if ! grep -Fq "${needle}" "${body_file}"; then
    echo "[FAIL] ${context}: expected body to contain '${needle}'" >&2
    echo "--- response body ---" >&2
    cat "${body_file}" >&2 || true
    echo >&2
    exit 1
  fi
}

require_header_equals() {
  local header_file="$1"
  local header_name="$2"
  local expected="$3"
  local context="$4"

  local value
  value="$(awk -F': *' -v key="${header_name}" 'BEGIN{IGNORECASE=1} $1==key {gsub(/\r/, "", $2); print $2}' "${header_file}" | tail -n1)"

  if [[ "${value}" != "${expected}" ]]; then
    echo "[FAIL] ${context}: expected header ${header_name}=${expected}, got '${value}'" >&2
    echo "--- response headers ---" >&2
    cat "${header_file}" >&2 || true
    echo >&2
    exit 1
  fi
}

check_capabilities_json() {
  local body_file="$1"
  local expected_protocol="$2"

  python3 - "${body_file}" "${expected_protocol}" <<'PY'
import json
import sys

path = sys.argv[1]
expected = sys.argv[2]

with open(path, "r", encoding="utf-8") as f:
    payload = json.load(f)

if payload.get("protocolVersion") != expected:
    raise SystemExit(f"protocolVersion mismatch: expected {expected}, got {payload.get('protocolVersion')}")

capabilities = payload.get("capabilities")
if not isinstance(capabilities, list):
    raise SystemExit("capabilities must be a JSON array")

required = {"activate", "complete", "fail", "extend-lock"}
missing = sorted(required.difference(set(str(c) for c in capabilities)))
if missing:
    raise SystemExit("missing capabilities: " + ", ".join(missing))
PY
}

check_activate_json_shape() {
  local body_file="$1"

  python3 - "${body_file}" <<'PY'
import json
import sys

path = sys.argv[1]

with open(path, "r", encoding="utf-8") as f:
    payload = json.load(f)

jobs = payload.get("jobs")
if not isinstance(jobs, list):
    raise SystemExit("jobs must be a JSON array")
PY
}

echo "Running worker API conformance smoke against ${WORKER_API_BASE_URL}"

cap_body="${TMP_DIR}/capabilities.body"
cap_headers="${TMP_DIR}/capabilities.headers"
cap_status="$(curl_with_context "GET" "/jobs/capabilities" "" "${cap_body}" "${cap_headers}" -H "X-Workflow-Worker-Protocol-Version: ${WORKER_PROTOCOL_VERSION}")"

if [[ "${cap_status}" == "401" || "${cap_status}" == "403" ]]; then
  echo "[FAIL] capabilities endpoint requires auth (status=${cap_status}); supply WORKER_AUTH_BEARER_TOKEN" >&2
  exit 1
fi
require_status "${cap_status}" "200" "GET /jobs/capabilities" "${cap_body}"
require_header_equals "${cap_headers}" "X-Workflow-Engine-Protocol-Version" "${WORKER_PROTOCOL_VERSION}" "GET /jobs/capabilities"
check_capabilities_json "${cap_body}" "${WORKER_PROTOCOL_VERSION}"
echo "[OK] capabilities contract"

activate_invalid_req="${TMP_DIR}/activate-invalid.request.json"
activate_invalid_body="${TMP_DIR}/activate-invalid.body"
activate_invalid_headers="${TMP_DIR}/activate-invalid.headers"
cat >"${activate_invalid_req}" <<'JSON'
{"type":"conformance-no-jobs","worker":"conformance-worker","maxJobs":1,"timeoutMs":10,"lockDurationMs":1000}
JSON

activate_invalid_status="$(curl_with_context "POST" "/jobs/activate" "${activate_invalid_req}" "${activate_invalid_body}" "${activate_invalid_headers}" -H "Content-Type: application/json" -H "X-Workflow-Worker-Protocol-Version: v999")"
require_status "${activate_invalid_status}" "400" "POST /jobs/activate (unsupported protocol)" "${activate_invalid_body}"
require_body_contains "${activate_invalid_body}" "unsupported worker protocol version" "POST /jobs/activate (unsupported protocol)"
echo "[OK] protocol version validation"

activate_req="${TMP_DIR}/activate.request.json"
activate_body="${TMP_DIR}/activate.body"
activate_headers="${TMP_DIR}/activate.headers"
cat >"${activate_req}" <<'JSON'
{"type":"conformance-no-jobs","worker":"conformance-worker","maxJobs":1,"timeoutMs":10,"lockDurationMs":1000}
JSON

activate_status="$(curl_with_context "POST" "/jobs/activate" "${activate_req}" "${activate_body}" "${activate_headers}" -H "Content-Type: application/json" -H "X-Workflow-Worker-Protocol-Version: ${WORKER_PROTOCOL_VERSION}")"
require_status "${activate_status}" "200" "POST /jobs/activate" "${activate_body}"
require_header_equals "${activate_headers}" "X-Workflow-Engine-Protocol-Version" "${WORKER_PROTOCOL_VERSION}" "POST /jobs/activate"
check_activate_json_shape "${activate_body}"
echo "[OK] activate response shape"

idempotency_oversized="$(python3 - <<'PY'
print('x' * 129)
PY
)"

complete_req="${TMP_DIR}/complete.request.json"
complete_body="${TMP_DIR}/complete.body"
complete_headers="${TMP_DIR}/complete.headers"
cat >"${complete_req}" <<'JSON'
{"worker":"conformance-worker","variables":{}}
JSON

complete_status="$(curl_with_context "POST" "/jobs/1/complete" "${complete_req}" "${complete_body}" "${complete_headers}" -H "Content-Type: application/json" -H "X-Workflow-Worker-Protocol-Version: ${WORKER_PROTOCOL_VERSION}" -H "Idempotency-Key: ${idempotency_oversized}")"
require_status "${complete_status}" "400" "POST /jobs/{key}/complete oversized Idempotency-Key" "${complete_body}"
require_body_contains "${complete_body}" "Idempotency-Key exceeds" "POST /jobs/{key}/complete oversized Idempotency-Key"
echo "[OK] idempotency key guardrail"

echo "Worker API conformance smoke passed"
