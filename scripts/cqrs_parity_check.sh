#!/usr/bin/env bash
set -euo pipefail

COMPOSE_CMD="${COMPOSE_CMD:-docker-compose}"
POSTGRES_USER="${POSTGRES_USER:-user}"
POSTGRES_DB="${POSTGRES_DB:-workflow_db}"
ES_ADDR="${ES_ADDR:-http://localhost:9200}"
ES_INDEX_PREFIX="${ES_INDEX_PREFIX:-goflow}"
INCLUDE_KEY_DIFF="${INCLUDE_KEY_DIFF:-false}"

TABLES=(
  process
  process_instance
  element_instance
  variable
  job
  incident
  timer
  message_subscription
)

KEY_DIFF_TABLES=(
  process
  process_instance
  element_instance
  job
  incident
  timer
  message_subscription
)

pg_count() {
  local table="$1"
  ${COMPOSE_CMD} exec -T postgres psql -U "${POSTGRES_USER}" -d "${POSTGRES_DB}" -tAc "SELECT COUNT(*) FROM ${table};" | tr -d '[:space:]'
}

es_count() {
  local index="$1"
  local response
  response="$(curl -fsS "${ES_ADDR}/${index}/_count" 2>/dev/null || true)"
  if [[ -z "${response}" ]]; then
    echo "0"
    return
  fi
  python3 - <<'PY' "${response}"
import json,sys
try:
    body=json.loads(sys.argv[1])
except Exception:
    print(0)
    raise SystemExit(0)
print(body.get("count",0))
PY
}

key_diff() {
  local table="$1"
  local index="$2"

  local tmp_pg tmp_es
  tmp_pg="$(mktemp)"
  tmp_es="$(mktemp)"

  ${COMPOSE_CMD} exec -T postgres psql -U "${POSTGRES_USER}" -d "${POSTGRES_DB}" -tAc "SELECT key FROM ${table} ORDER BY key;" | sed '/^$/d' > "${tmp_pg}"
  {
    curl -fsS "${ES_ADDR}/${index}/_search?size=10000&_source=key&sort=key:asc" 2>/dev/null || true
  } | python3 - <<'PY' > "${tmp_es}"
import json,sys
try:
    payload=json.load(sys.stdin)
except Exception:
    raise SystemExit(0)
for hit in payload.get("hits",{}).get("hits",[]):
    src=hit.get("_source",{})
    if "key" in src:
        print(src["key"])
PY

  sort -u "${tmp_pg}" -o "${tmp_pg}"
  sort -u "${tmp_es}" -o "${tmp_es}"

  local missing_in_es missing_in_pg
  missing_in_es="$(comm -23 "${tmp_pg}" "${tmp_es}" | wc -l | tr -d '[:space:]')"
  missing_in_pg="$(comm -13 "${tmp_pg}" "${tmp_es}" | wc -l | tr -d '[:space:]')"

  if [[ "${missing_in_es}" != "0" || "${missing_in_pg}" != "0" ]]; then
    echo "   key-diff: missing_in_es=${missing_in_es}, missing_in_pg=${missing_in_pg}"
    echo "   sample missing in ES:"
    comm -23 "${tmp_pg}" "${tmp_es}" | head -n 5 | sed 's/^/     - /'
    echo "   sample missing in PG:"
    comm -13 "${tmp_pg}" "${tmp_es}" | head -n 5 | sed 's/^/     - /'
    rm -f "${tmp_pg}" "${tmp_es}"
    return 1
  fi

  rm -f "${tmp_pg}" "${tmp_es}"
  return 0
}

contains_table() {
  local value="$1"
  shift
  for item in "$@"; do
    if [[ "${item}" == "${value}" ]]; then
      return 0
    fi
  done
  return 1
}

echo "Running CQRS parity check"
echo "- Compose command: ${COMPOSE_CMD}"
echo "- Postgres DB: ${POSTGRES_DB}"
echo "- ES address: ${ES_ADDR}"
echo "- Index prefix: ${ES_INDEX_PREFIX}"

overall_status=0
for table in "${TABLES[@]}"; do
  index="${ES_INDEX_PREFIX}-${table}"
  pg_val="$(pg_count "${table}")"
  es_val="$(es_count "${index}")"

  status="OK"
  if [[ "${pg_val}" != "${es_val}" ]]; then
    status="MISMATCH"
    overall_status=1
  fi

  echo "- ${table}: pg=${pg_val} es=${es_val} [${status}]"

  if [[ "${INCLUDE_KEY_DIFF}" == "true" ]] && contains_table "${table}" "${KEY_DIFF_TABLES[@]}"; then
    if ! key_diff "${table}" "${index}"; then
      overall_status=1
    fi
  fi
done

if [[ "${overall_status}" -ne 0 ]]; then
  echo "Parity check failed"
  exit 1
fi

echo "Parity check passed"
