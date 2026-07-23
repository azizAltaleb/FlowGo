#!/usr/bin/env bash
set -euo pipefail

MODE="dry-run"
CONFIRM=""
OLD_PREFIX="${OLD_PREFIX:-flowgo}"
NEW_PREFIX="${NEW_PREFIX:-artificialflow}"
INDEX_VERSION="${INDEX_VERSION:-v1}"
ES_URL="${ES_URL:-http://localhost:9200}"
SNAPSHOT_REPOSITORY="${SNAPSHOT_REPOSITORY:-}"
SNAPSHOT_NAME="${SNAPSHOT_NAME:-artificialflow-prefix-cutover-${INDEX_VERSION}}"
WORKER_DRAINED="${WORKER_DRAINED:-false}"
TABLES=(process process_instance element_instance variable job incident timer message_subscription)

usage() {
  cat <<'EOF'
Usage: scripts/migrate_es_prefix.sh [--apply --confirm migrate-artificialflow-es]

Defaults to dry-run and performs no network requests. Apply mode additionally
requires WORKER_DRAINED=true and SNAPSHOT_REPOSITORY=<configured repository>.
Legacy indices are never deleted.
EOF
}

while (($#)); do
  case "$1" in
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
DRY RUN: no Elasticsearch requests were made.

Planned migration:
1. Stop producers that change projection tables; stop and drain sync-worker.
2. Set WORKER_DRAINED=true only after Kafka lag is zero and no worker is running.
3. Snapshot ${OLD_PREFIX}-* to SNAPSHOT_REPOSITORY as ${SNAPSHOT_NAME}.
4. For each known table, create ${NEW_PREFIX}-<table>-${INDEX_VERSION}, reindex
   ${OLD_PREFIX}-<table>, compare document counts and complete ID digests.
5. Atomically point ${NEW_PREFIX}-<table> aliases at the versioned indices.
6. Retain every ${OLD_PREFIX}-* index for rollback.

Apply explicitly:
  WORKER_DRAINED=true SNAPSHOT_REPOSITORY=<repo> \\
    $0 --apply --confirm migrate-artificialflow-es
EOF
  exit 0
fi

if [[ "${CONFIRM}" != "migrate-artificialflow-es" ]]; then
  echo "Refusing apply: pass --confirm migrate-artificialflow-es" >&2
  exit 2
fi
if [[ "${WORKER_DRAINED}" != "true" ]]; then
  echo "Refusing apply: WORKER_DRAINED=true is required" >&2
  exit 2
fi
if [[ -z "${SNAPSHOT_REPOSITORY}" ]]; then
  echo "Refusing apply: SNAPSHOT_REPOSITORY is required" >&2
  exit 2
fi
command -v curl >/dev/null
command -v jq >/dev/null

es_status() {
  curl --silent --output /dev/null --write-out '%{http_code}' "$1"
}

existing_indices=()
for table in "${TABLES[@]}"; do
  old_index="${OLD_PREFIX}-${table}"
  if [[ "$(es_status "${ES_URL}/${old_index}")" == "200" ]]; then
    existing_indices+=("${old_index}")
  fi
done
if ((${#existing_indices[@]} == 0)); then
  echo "No ${OLD_PREFIX}-* known indices exist; nothing to migrate."
  exit 0
fi

indices_csv="$(IFS=,; echo "${existing_indices[*]}")"
snapshot_status="$(es_status "${ES_URL}/_snapshot/${SNAPSHOT_REPOSITORY}/${SNAPSHOT_NAME}")"
if [[ "${snapshot_status}" == "404" ]]; then
  curl --fail --silent --show-error \
    -X PUT "${ES_URL}/_snapshot/${SNAPSHOT_REPOSITORY}/${SNAPSHOT_NAME}?wait_for_completion=true" \
    -H 'Content-Type: application/json' \
    -d "{\"indices\":\"${indices_csv}\",\"include_global_state\":false}" |
    jq -e '.snapshot.state == "SUCCESS"' >/dev/null
elif [[ "${snapshot_status}" == "200" ]]; then
  curl --fail --silent --show-error \
    "${ES_URL}/_snapshot/${SNAPSHOT_REPOSITORY}/${SNAPSHOT_NAME}" |
    jq -e '.snapshots | length > 0 and all(.state == "SUCCESS")' >/dev/null
else
  echo "Unable to inspect snapshot; HTTP ${snapshot_status}" >&2
  exit 1
fi
echo "Snapshot checkpoint verified: ${SNAPSHOT_REPOSITORY}/${SNAPSHOT_NAME}"

id_digest() {
  local index="$1"
  local output scroll_id response hits
  output="$(mktemp)"
  response="$(curl --fail --silent --show-error \
    -X POST "${ES_URL}/${index}/_search?scroll=1m" \
    -H 'Content-Type: application/json' \
    -d '{"size":1000,"sort":["_doc"],"_source":false}')"
  scroll_id="$(jq -r '._scroll_id' <<<"${response}")"
  while :; do
    hits="$(jq '.hits.hits | length' <<<"${response}")"
    [[ "${hits}" == "0" ]] && break
    jq -r '.hits.hits[]._id' <<<"${response}" >>"${output}"
    response="$(curl --fail --silent --show-error \
      -X POST "${ES_URL}/_search/scroll" \
      -H 'Content-Type: application/json' \
      -d "$(jq -nc --arg id "${scroll_id}" '{scroll:"1m",scroll_id:$id}')" )"
    scroll_id="$(jq -r '._scroll_id' <<<"${response}")"
  done
  curl --silent --output /dev/null -X DELETE "${ES_URL}/_search/scroll" \
    -H 'Content-Type: application/json' \
    -d "$(jq -nc --arg id "${scroll_id}" '{scroll_id:[$id]}')" || true
  LC_ALL=C sort "${output}" | shasum -a 256 | awk '{print $1}'
  rm -f "${output}"
}

for table in "${TABLES[@]}"; do
  old_index="${OLD_PREFIX}-${table}"
  destination="${NEW_PREFIX}-${table}-${INDEX_VERSION}"
  alias_name="${NEW_PREFIX}-${table}"

  if [[ "$(es_status "${ES_URL}/${old_index}")" != "200" ]]; then
    echo "Skipping absent legacy index ${old_index}"
    continue
  fi
  if [[ "$(es_status "${ES_URL}/${alias_name}")" == "200" ]] &&
     [[ "$(es_status "${ES_URL}/_alias/${alias_name}")" == "404" ]]; then
    echo "Refusing: ${alias_name} is a concrete index, not an alias" >&2
    exit 1
  fi

  if [[ "$(es_status "${ES_URL}/${destination}")" == "404" ]]; then
    mapping="$(curl --fail --silent --show-error "${ES_URL}/${old_index}/_mapping" |
      jq -c --arg index "${old_index}" '.[$index] | {mappings:.mappings}')"
    curl --fail --silent --show-error \
      -X PUT "${ES_URL}/${destination}" \
      -H 'Content-Type: application/json' \
      -d "${mapping}" >/dev/null
  fi

  curl --fail --silent --show-error \
    -X POST "${ES_URL}/_reindex?wait_for_completion=true&refresh=true" \
    -H 'Content-Type: application/json' \
    -d "$(jq -nc --arg source "${old_index}" --arg dest "${destination}" \
      '{source:{index:$source},dest:{index:$dest,op_type:"index"}}')" |
    jq -e '.failures | length == 0' >/dev/null

  old_count="$(curl --fail --silent --show-error "${ES_URL}/${old_index}/_count" | jq -r '.count')"
  new_count="$(curl --fail --silent --show-error "${ES_URL}/${destination}/_count" | jq -r '.count')"
  if [[ "${old_count}" != "${new_count}" ]]; then
    echo "Count mismatch for ${table}: ${old_count} != ${new_count}" >&2
    exit 1
  fi
  if [[ "$(id_digest "${old_index}")" != "$(id_digest "${destination}")" ]]; then
    echo "ID mismatch for ${table}" >&2
    exit 1
  fi

  alias_actions="$(jq -nc --arg alias "${alias_name}" --arg destination "${destination}" \
    '{actions:[{remove:{index:"*",alias:$alias,must_exist:false}},{add:{index:$destination,alias:$alias,is_write_index:true}}]}')"
  curl --fail --silent --show-error \
    -X POST "${ES_URL}/_aliases" \
    -H 'Content-Type: application/json' \
    -d "${alias_actions}" >/dev/null
  echo "Verified and switched ${alias_name} (${new_count} documents)"
done

echo "Migration complete. Legacy indices remain intact for rollback."
