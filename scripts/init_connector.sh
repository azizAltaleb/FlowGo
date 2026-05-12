#!/usr/bin/env bash
set -euo pipefail

CONNECT_URL="${CONNECT_URL:-http://localhost:8083}"
CONNECTOR_NAME="${CONNECTOR_NAME:-flowgo-postgres-connector}"
CONNECTOR_FILE="${CONNECTOR_FILE:-debezium/connector-register.json}"
CONNECT_WAIT_TIMEOUT_SEC="${CONNECT_WAIT_TIMEOUT_SEC:-180}"

echo "Waiting for Kafka Connect to be ready (timeout=${CONNECT_WAIT_TIMEOUT_SEC}s)..."
started="$(date +%s)"
until [ "$(curl -s -o /dev/null -w "%{http_code}" "${CONNECT_URL}/connectors")" -eq 200 ]; do
  now="$(date +%s)"
  if (( now - started > CONNECT_WAIT_TIMEOUT_SEC )); then
    echo "Timed out waiting for Kafka Connect at ${CONNECT_URL}/connectors"
    exit 1
  fi
  echo "Waiting for connect..."
  sleep 5
done

if curl -s "${CONNECT_URL}/connectors" | grep -q "\"${CONNECTOR_NAME}\""; then
  echo "Connector ${CONNECTOR_NAME} already exists. Skipping registration."
  exit 0
fi

echo "Registering connector ${CONNECTOR_NAME}..."
response="$(curl -sS -w $'\n%{http_code}' -X POST -H "Content-Type: application/json" --data @"${CONNECTOR_FILE}" "${CONNECT_URL}/connectors")"
status="${response##*$'\n'}"
body="${response%$'\n'*}"

if [[ "${status}" == "200" || "${status}" == "201" ]]; then
  echo
  echo "Connector registered."
  exit 0
fi

if [[ "${status}" == "409" ]]; then
  echo
  echo "Connector ${CONNECTOR_NAME} already exists (race/duplicate registration). Skipping."
  exit 0
fi

echo "Connector registration failed with status=${status}"
echo "Response body: ${body}"
exit 1
