#!/usr/bin/env bash
set -euo pipefail

MODE="dry-run"
CONFIRM=""
CONNECT_URL="${CONNECT_URL:-http://localhost:8083}"
KAFKA_BOOTSTRAP="${KAFKA_BOOTSTRAP:-localhost:9092}"
PG_DSN="${PG_DSN:-host=localhost user=artificialflow password=password dbname=workflow_db sslmode=disable}"
OLD_CONNECTOR="${OLD_CONNECTOR:-legacy-postgres-connector}"
NEW_CONNECTOR="${NEW_CONNECTOR:-artificialflow-postgres-connector}"
OLD_GROUP="${OLD_GROUP:-legacy-sync-worker-v8}"
NEW_GROUP="${NEW_GROUP:-artificialflow-sync-worker-v8}"
OLD_SLOT="${OLD_SLOT:-legacy_slot}"
NEW_SLOT="${NEW_SLOT:-artificialflow_slot}"
NEW_CONNECTOR_FILE="${NEW_CONNECTOR_FILE:-debezium/connector-register.json}"
STATE_DIR="${STATE_DIR:-streaming-cutover-state}"
QUIESCED="${QUIESCED:-false}"
WORKER_DRAINED="${WORKER_DRAINED:-false}"
CONNECT_INTERNALS_MIGRATED="${CONNECT_INTERNALS_MIGRATED:-false}"

usage() {
  cat <<'EOF'
Usage:
  scripts/migrate_streaming_identifiers.sh
  scripts/migrate_streaming_identifiers.sh --capture
  scripts/migrate_streaming_identifiers.sh --apply \
    --confirm migrate-artificialflow-streaming

The default is an offline dry-run. --capture performs read-only preflight
queries. --apply requires previously captured state plus explicit quiescence,
drain, and Kafka Connect internal-topic migration acknowledgements.
EOF
}

while (($#)); do
  case "$1" in
    --capture) MODE="capture" ;;
    --apply) MODE="apply" ;;
    --confirm)
      shift
      CONFIRM="${1:-}"
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
  shift
done

if [[ "${MODE}" == "dry-run" ]]; then
  cat <<EOF
DRY RUN: no service or data requests were made.

Cutover checkpoints:
1. Capture ${OLD_CONNECTOR} config/status, ${OLD_GROUP} offsets/lag, and
   replication slot ${OLD_SLOT} confirmed_flush_lsn/restart_lsn.
2. Quiesce command/runtime writes; drain and stop every sync-worker replica.
3. Pause the old connector and verify its tasks are not RUNNING.
4. Stop Kafka Connect; restart it with group/internal topics prefixed
   artificialflow (set CONNECT_INTERNALS_MIGRATED=true only after verification).
5. Copy unchanged-topic offsets from ${OLD_GROUP} to inactive ${NEW_GROUP}.
6. Register ${NEW_CONNECTOR} with topic prefix artificialflow and slot
   ${NEW_SLOT}; keep the new sync worker stopped until connector health is checked.
7. Never run old and new connectors or consumer deployments simultaneously.

Read-only capture:
  $0 --capture

Explicit cutover after the runbook checkpoints:
  QUIESCED=true WORKER_DRAINED=true CONNECT_INTERNALS_MIGRATED=true \\
    $0 --apply --confirm migrate-artificialflow-streaming
EOF
  exit 0
fi

command -v curl >/dev/null
command -v jq >/dev/null
command -v kafka-consumer-groups >/dev/null
command -v psql >/dev/null

normalize_consumer_group() {
  local input="$1" output="$2" expected_group="$3"
  awk -v expected_group="${expected_group}" '
    BEGIN {
      OFS = "\t"
      header_found = 0
      row_count = 0
      failed = 0
    }

    function fail(message) {
      print "Invalid kafka-consumer-groups output: " message > "/dev/stderr"
      failed = 1
      exit 1
    }

    /^[[:space:]]*$/ { next }

    !header_found {
      inactive_message = "Consumer group \047" expected_group "\047 has no active members."
      if ($0 == inactive_message) {
        next
      }

      delete column
      delete seen_header
      for (i = 1; i <= NF; i++) {
        if (++seen_header[$i] > 1) {
          fail("duplicate header " $i)
        }
        column[$i] = i
      }
      required[1] = "GROUP"
      required[2] = "TOPIC"
      required[3] = "PARTITION"
      required[4] = "CURRENT-OFFSET"
      required[5] = "LOG-END-OFFSET"
      required[6] = "LAG"
      required_max = 0
      for (i = 1; i <= 6; i++) {
        if (!(required[i] in column)) {
          fail("missing required header " required[i])
        }
        if (column[required[i]] > required_max) {
          required_max = column[required[i]]
        }
      }

      print "GROUP", "TOPIC", "PARTITION", "CURRENT-OFFSET", "LOG-END-OFFSET", "LAG"
      header_found = 1
      next
    }

    {
      row_count++
      if (NF < required_max) {
        fail("row " row_count " has fewer fields than the header")
      }

      group = $(column["GROUP"])
      topic = $(column["TOPIC"])
      partition = $(column["PARTITION"])
      current_offset = $(column["CURRENT-OFFSET"])
      log_end_offset = $(column["LOG-END-OFFSET"])
      lag = $(column["LAG"])

      if (group != expected_group) {
        fail("row " row_count " belongs to unexpected group " group)
      }
      if (topic == "") {
        fail("row " row_count " has an empty topic")
      }
      if (partition !~ /^[0-9]+$/) {
        fail("row " row_count " has non-numeric partition " partition)
      }
      if (current_offset !~ /^[0-9]+$/) {
        fail("row " row_count " has unknown or non-numeric current offset " current_offset)
      }
      if (log_end_offset !~ /^[0-9]+$/) {
        fail("row " row_count " has unknown or non-numeric log-end offset " log_end_offset)
      }
      if (lag !~ /^[0-9]+$/) {
        fail("row " row_count " has unknown or non-numeric lag " lag)
      }

      key = topic SUBSEP partition
      if (seen_partition[key]++) {
        fail("duplicate topic/partition " topic "/" partition)
      }
      print group, topic, partition, current_offset, log_end_offset, lag
    }

    END {
      if (failed) {
        exit 1
      }
      if (!header_found) {
        print "Invalid kafka-consumer-groups output: required header was not found" > "/dev/stderr"
        exit 1
      }
      if (row_count == 0) {
        print "Invalid kafka-consumer-groups output: no partition rows were returned" > "/dev/stderr"
        exit 1
      }
    }
  ' "${input}" >"${output}"
}

if [[ "${MODE}" == "capture" ]]; then
  mkdir -p "${STATE_DIR}"
  curl --fail --silent --show-error \
    "${CONNECT_URL}/connectors/${OLD_CONNECTOR}/config" |
    jq . >"${STATE_DIR}/old-connector-config.json"
  curl --fail --silent --show-error \
    "${CONNECT_URL}/connectors/${OLD_CONNECTOR}/status" |
    jq . >"${STATE_DIR}/old-connector-status.json"
  # Invalidate any earlier offset capture before starting a new one so a failed
  # normalization cannot leave stale state eligible for a later apply.
  rm -f \
    "${STATE_DIR}/old-consumer-group.txt" \
    "${STATE_DIR}/old-consumer-group.tsv"
  consumer_group_raw="$(mktemp "${STATE_DIR}/.old-consumer-group.raw.XXXXXX")"
  consumer_group_normalized="$(mktemp "${STATE_DIR}/.old-consumer-group.normalized.XXXXXX")"
  trap 'rm -f "${consumer_group_raw:-}" "${consumer_group_normalized:-}"' EXIT
  kafka-consumer-groups --bootstrap-server "${KAFKA_BOOTSTRAP}" \
    --group "${OLD_GROUP}" --describe >"${consumer_group_raw}"
  normalize_consumer_group \
    "${consumer_group_raw}" \
    "${consumer_group_normalized}" \
    "${OLD_GROUP}"
  mv "${consumer_group_raw}" "${STATE_DIR}/old-consumer-group.txt"
  mv "${consumer_group_normalized}" "${STATE_DIR}/old-consumer-group.tsv"
  psql "${PG_DSN}" --no-psqlrc --csv -c \
    "SELECT slot_name, active, restart_lsn, confirmed_flush_lsn FROM pg_replication_slots WHERE slot_name IN ('${OLD_SLOT}','${NEW_SLOT}') ORDER BY slot_name;" \
    >"${STATE_DIR}/replication-slots.csv"
  echo "Read-only preflight captured in ${STATE_DIR}"
  exit 0
fi

if [[ "${CONFIRM}" != "migrate-artificialflow-streaming" ]]; then
  echo "Refusing apply: pass --confirm migrate-artificialflow-streaming" >&2
  exit 2
fi
for acknowledgement in QUIESCED WORKER_DRAINED CONNECT_INTERNALS_MIGRATED; do
  if [[ "${!acknowledgement}" != "true" ]]; then
    echo "Refusing apply: ${acknowledgement}=true is required" >&2
    exit 2
  fi
done
for captured in old-connector-config.json old-connector-status.json old-consumer-group.txt old-consumer-group.tsv replication-slots.csv; do
  if [[ ! -s "${STATE_DIR}/${captured}" ]]; then
    echo "Refusing apply: missing read-only capture ${STATE_DIR}/${captured}" >&2
    exit 2
  fi
done

apply_tmp="$(mktemp -d "${STATE_DIR}/.streaming-apply.XXXXXX")"
trap 'rm -rf "${apply_tmp:-}"' EXIT
normalize_consumer_group \
  "${STATE_DIR}/old-consumer-group.txt" \
  "${apply_tmp}/old-consumer-group.tsv" \
  "${OLD_GROUP}"
if ! cmp -s "${STATE_DIR}/old-consumer-group.tsv" "${apply_tmp}/old-consumer-group.tsv"; then
  echo "Refusing apply: normalized consumer-group capture does not match its raw source" >&2
  exit 1
fi

if ! awk -F '	' '
  NR == 1 { next }
  $6 != "0" { exit 1 }
' "${apply_tmp}/old-consumer-group.tsv"; then
  echo "Refusing apply: every captured old consumer lag must be numeric zero" >&2
  exit 1
fi

connector_may_be_active() {
  local name="$1" status body
  status="$(curl --silent --output /tmp/artificialflow-connector-status.json \
    --write-out '%{http_code}' "${CONNECT_URL}/connectors/${name}/status")"
  if [[ "${status}" == "404" ]]; then
    return 1
  fi
  if [[ "${status}" != "200" ]]; then
    echo "Unable to inspect connector ${name}; HTTP ${status}" >&2
    exit 1
  fi
  body="$(< /tmp/artificialflow-connector-status.json)"
  jq -e '
    def may_be_active:
      ((. // "") | tostring | ascii_upcase) as $state
      | ($state != "FAILED" and $state != "PAUSED" and $state != "STOPPED");
    (.connector.state | may_be_active)
      or any(.tasks[]?; .state | may_be_active)
  ' \
    <<<"${body}" >/dev/null
}

if connector_may_be_active "${OLD_CONNECTOR}"; then
  echo "Refusing apply: old connector ${OLD_CONNECTOR} or one of its tasks may still be active" >&2
  exit 1
fi
if connector_may_be_active "${NEW_CONNECTOR}"; then
  echo "New connector may already be active; leaving state unchanged (idempotent exit)."
  exit 0
fi

# Copy offsets only for unchanged topics (for example workflow.events.v1).
# Renamed Debezium topics intentionally begin at the new connector's cutover.
awk -F '	' '
  BEGIN { OFS = "\t"; print "TOPIC", "PARTITION", "CURRENT-OFFSET" }
  NR > 1 && $2 !~ /^legacy\./ { print $2, $3, $4 }
' "${apply_tmp}/old-consumer-group.tsv" >"${apply_tmp}/unchanged-offsets.tsv"

while IFS=$'\t' read -r topic partition offset; do
  if [[ "${topic}" == "TOPIC" ]]; then
    continue
  fi
  kafka-consumer-groups --bootstrap-server "${KAFKA_BOOTSTRAP}" \
    --group "${NEW_GROUP}" --topic "${topic}:${partition}" \
    --reset-offsets --to-offset "${offset}" --execute >/dev/null
done <"${apply_tmp}/unchanged-offsets.tsv"

unchanged_count="$(awk 'END { print NR - 1 }' "${apply_tmp}/unchanged-offsets.tsv")"
if ((unchanged_count > 0)); then
  kafka-consumer-groups --bootstrap-server "${KAFKA_BOOTSTRAP}" \
    --group "${NEW_GROUP}" --describe >"${apply_tmp}/new-consumer-group.txt"
  normalize_consumer_group \
    "${apply_tmp}/new-consumer-group.txt" \
    "${apply_tmp}/new-consumer-group.tsv" \
    "${NEW_GROUP}"

  if ! awk -F '	' '
    FNR == NR {
      if (FNR > 1) {
        source[$1 SUBSEP $2] = $3
      }
      next
    }
    FNR > 1 {
      destination[$2 SUBSEP $3] = $4
    }
    END {
      failed = 0
      for (key in source) {
        split(key, parts, SUBSEP)
        if (!(key in destination)) {
          print "Destination group is missing " parts[1] "/" parts[2] > "/dev/stderr"
          failed = 1
        } else if (destination[key] != source[key]) {
          print "Destination offset mismatch for " parts[1] "/" parts[2] \
            ": expected " source[key] ", got " destination[key] > "/dev/stderr"
          failed = 1
        }
      }
      exit failed
    }
  ' "${apply_tmp}/unchanged-offsets.tsv" "${apply_tmp}/new-consumer-group.tsv"; then
    echo "Refusing apply: destination consumer offsets did not match captured source offsets" >&2
    exit 1
  fi
fi

payload="$(jq \
  --arg name "${NEW_CONNECTOR}" \
  --arg slot "${NEW_SLOT}" \
  '.name=$name |
   .config["topic.prefix"]="artificialflow" |
   .config["slot.name"]=$slot |
   .config["snapshot.mode"]="no_data"' \
  "${NEW_CONNECTOR_FILE}")"
curl --fail --silent --show-error \
  -X POST "${CONNECT_URL}/connectors" \
  -H 'Content-Type: application/json' \
  -d "${payload}" |
  jq -e --arg name "${NEW_CONNECTOR}" '.name == $name' >/dev/null

echo "Canonical connector registered. Keep sync-worker stopped until connector"
echo "status, slot LSN, canonical topics, and ${NEW_GROUP} offsets are verified."
