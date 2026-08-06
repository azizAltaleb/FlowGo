#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

cat > "${TMP_DIR}/compose" <<'SH'
#!/usr/bin/env bash
printf '1\n'
SH

cat > "${TMP_DIR}/curl" <<'SH'
#!/usr/bin/env bash
count_file="${CQRS_PARITY_TEST_COUNT_FILE:?}"
count=0
[[ ! -f "${count_file}" ]] || count="$(<"${count_file}")"
count=$((count + 1))
printf '%s' "${count}" > "${count_file}"
if [[ "${count}" -eq 1 ]]; then
  printf '{"count":0}\n'
else
  printf '{"count":1}\n'
fi
SH

chmod +x "${TMP_DIR}/compose" "${TMP_DIR}/curl"

output="$(
  PATH="${TMP_DIR}:${PATH}" \
  COMPOSE_CMD="${TMP_DIR}/compose" \
  CQRS_PARITY_TEST_COUNT_FILE="${TMP_DIR}/curl-count" \
  PARITY_WAIT_TIMEOUT_SEC=2 \
  PARITY_POLL_INTERVAL_SEC=0 \
  bash "${ROOT}/scripts/cqrs_parity_check.sh"
)"

[[ "${output}" == *"Parity has not converged"* ]]
[[ "${output}" == *"Parity check passed"* ]]

echo "CQRS parity retry regression test passed"
