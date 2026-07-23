#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
scripts=(
  "${ROOT}/scripts/migrate_es_prefix.sh"
  "${ROOT}/scripts/migrate_streaming_identifiers.sh"
  "${ROOT}/scripts/migrate_database_and_state.sh"
)

for script in "${scripts[@]}"; do
  bash -n "${script}"
  output="$(bash "${script}")"
  if [[ "${output}" != *"DRY RUN"* ]]; then
    echo "${script} did not report dry-run mode" >&2
    exit 1
  fi
done

TEST_ROOT="$(mktemp -d)"
trap 'rm -rf "${TEST_ROOT}"' EXIT
MOCK_BIN="${TEST_ROOT}/bin"
mkdir -p "${MOCK_BIN}"

cat >"${MOCK_BIN}/curl" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

method="GET"
output=""
url=""
status_json="${MOCK_CONNECTOR_STATUS_JSON:-}"
if [[ -z "${status_json}" ]]; then
  status_json='{"connector":{"state":"PAUSED"},"tasks":[]}'
fi
while (($#)); do
  case "$1" in
    -X)
      shift
      method="${1:-}"
      ;;
    --output)
      shift
      output="${1:-}"
      ;;
    --write-out)
      shift
      ;;
    -H|-d)
      shift
      ;;
    --fail|--silent|--show-error)
      ;;
    http://*|https://*)
      url="$1"
      ;;
  esac
  shift
done

if [[ -n "${output}" ]]; then
  printf '%s\n' "${status_json}" >"${output}"
  printf '%s' "${MOCK_CONNECTOR_STATUS_HTTP:-404}"
elif [[ "${method}" == "POST" ]]; then
  printf '{"name":"artificialflow-postgres-connector"}\n'
elif [[ "${url}" == */config ]]; then
  printf '{"connector.class":"io.debezium.connector.postgresql.PostgresConnector"}\n'
elif [[ "${url}" == */status ]]; then
  printf '%s\n' "${status_json}"
else
  printf '{}\n'
fi
EOF

cat >"${MOCK_BIN}/kafka-consumer-groups" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

group=""
describe=false
reset=false
arguments="$*"
while (($#)); do
  case "$1" in
    --group)
      shift
      group="${1:-}"
      ;;
    --describe)
      describe=true
      ;;
    --reset-offsets)
      reset=true
      ;;
  esac
  shift
done

if [[ "${reset}" == "true" ]]; then
  printf '%s\n' "${arguments}" >>"${MOCK_KAFKA_LOG}"
  printf 'reset complete\n'
elif [[ "${describe}" == "true" && "${group}" == "${OLD_GROUP:-flowgo-sync-worker-v8}" ]]; then
  printf '%s\n' "${MOCK_OLD_DESCRIBE}"
elif [[ "${describe}" == "true" && "${group}" == "${NEW_GROUP:-artificialflow-sync-worker-v8}" ]]; then
  printf '%s\n' "${MOCK_NEW_DESCRIBE:-}"
else
  echo "unexpected kafka-consumer-groups invocation: ${arguments}" >&2
  exit 1
fi
EOF

cat >"${MOCK_BIN}/psql" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
printf 'slot_name,active,restart_lsn,confirmed_flush_lsn\n'
printf 'flowgo_slot,f,0/1,0/1\n'
EOF

chmod +x "${MOCK_BIN}/curl" "${MOCK_BIN}/kafka-consumer-groups" "${MOCK_BIN}/psql"

consumer_header="GROUP TOPIC PARTITION CURRENT-OFFSET LOG-END-OFFSET LAG CONSUMER-ID HOST CLIENT-ID"

run_capture() {
  local state_dir="$1" description="$2"
  (
    cd "${ROOT}"
    PATH="${MOCK_BIN}:${PATH}" \
      STATE_DIR="${state_dir}" \
      MOCK_KAFKA_LOG="${state_dir}/kafka.log" \
      MOCK_OLD_DESCRIBE="${description}" \
      bash scripts/migrate_streaming_identifiers.sh --capture
  )
}

run_apply() {
  local state_dir="$1" destination_description="$2"
  local connector_status="${3:-}"
  local connector_status_http="${4:-404}"
  if [[ -z "${connector_status}" ]]; then
    connector_status='{"connector":{"state":"PAUSED"},"tasks":[]}'
  fi
  (
    cd "${ROOT}"
    PATH="${MOCK_BIN}:${PATH}" \
      STATE_DIR="${state_dir}" \
      MOCK_KAFKA_LOG="${state_dir}/kafka.log" \
      MOCK_NEW_DESCRIBE="${destination_description}" \
      MOCK_CONNECTOR_STATUS_JSON="${connector_status}" \
      MOCK_CONNECTOR_STATUS_HTTP="${connector_status_http}" \
      QUIESCED=true \
      WORKER_DRAINED=true \
      CONNECT_INTERNALS_MIGRATED=true \
      bash scripts/migrate_streaming_identifiers.sh \
        --apply --confirm migrate-artificialflow-streaming
  )
}

expect_failure() {
  local label="$1"
  shift
  if "$@" >"${TEST_ROOT}/${label}.out" 2>&1; then
    echo "${label} unexpectedly succeeded" >&2
    printf '%s\n' "--- captured output ---" >&2
    while IFS= read -r line; do
      printf '%s\n' "${line}" >&2
    done <"${TEST_ROOT}/${label}.out"
    exit 1
  fi
}

success_state="${TEST_ROOT}/success"
reordered_header="TOPIC PARTITION GROUP LOG-END-OFFSET LAG CURRENT-OFFSET CONSUMER-ID HOST CLIENT-ID"
success_source="${reordered_header}
workflow.events.v1 0 flowgo-sync-worker-v8 10 0 10 consumer /host client
flowgo.public.process 0 flowgo-sync-worker-v8 33 0 33 consumer /host client"
success_destination="${consumer_header}
artificialflow-sync-worker-v8 workflow.events.v1 0 10 10 0 consumer /host client"
run_capture "${success_state}" "${success_source}"
run_apply "${success_state}" "${success_destination}"
if ! grep -F -- "--topic workflow.events.v1:0 --reset-offsets --to-offset 10" "${success_state}/kafka.log" >/dev/null; then
  echo "zero-lag migration did not reset the unchanged topic offset" >&2
  exit 1
fi
if grep -F -- "flowgo.public.process" "${success_state}/kafka.log" >/dev/null; then
  echo "Debezium flowgo.* offsets must not be copied" >&2
  exit 1
fi

unknown_lag_state="${TEST_ROOT}/unknown-lag"
unknown_lag="${consumer_header}
flowgo-sync-worker-v8 workflow.events.v1 0 10 10 - consumer /host client"
run_capture "${unknown_lag_state}" "${success_source}"
expect_failure "unknown-lag" run_capture "${unknown_lag_state}" "${unknown_lag}"
if [[ -e "${unknown_lag_state}/old-consumer-group.txt" || -e "${unknown_lag_state}/old-consumer-group.tsv" ]]; then
  echo "failed capture left stale consumer-group state eligible for apply" >&2
  exit 1
fi

malformed_state="${TEST_ROOT}/malformed"
malformed="${consumer_header}
flowgo-sync-worker-v8 workflow.events.v1 0 10"
expect_failure "malformed-output" run_capture "${malformed_state}" "${malformed}"

missing_partition_state="${TEST_ROOT}/missing-partition"
missing_partition_source="${consumer_header}
flowgo-sync-worker-v8 workflow.events.v1 0 10 10 0 consumer /host client
flowgo-sync-worker-v8 workflow.events.v1 1 20 20 0 consumer /host client"
missing_partition_destination="${consumer_header}
artificialflow-sync-worker-v8 workflow.events.v1 0 10 10 0 consumer /host client"
run_capture "${missing_partition_state}" "${missing_partition_source}"
expect_failure \
  "missing-partition" \
  run_apply "${missing_partition_state}" "${missing_partition_destination}"

mismatch_state="${TEST_ROOT}/destination-mismatch"
run_capture "${mismatch_state}" "${success_source}"
mismatch_destination="${consumer_header}
artificialflow-sync-worker-v8 workflow.events.v1 0 9 10 1 consumer /host client"
expect_failure \
  "destination-mismatch" \
  run_apply "${mismatch_state}" "${mismatch_destination}"

active_connector_state="${TEST_ROOT}/active-connector"
run_capture "${active_connector_state}" "${success_source}"
expect_failure \
  "active-connector" \
  run_apply \
    "${active_connector_state}" \
    "${success_destination}" \
    '{"connector":{"state":"UNASSIGNED"},"tasks":[]}' \
    "200"

echo "Persistent migration dry-run and mocked capture/apply checks passed."
