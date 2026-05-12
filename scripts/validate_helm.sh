#!/usr/bin/env bash
set -euo pipefail

CHART_DIR="${CHART_DIR:-charts/flowgo}"

ruby -e 'require "yaml"; ARGV.each { |f| YAML.load_file(f); puts "ok #{f}" }' \
  "${CHART_DIR}/Chart.yaml" \
  "${CHART_DIR}/values.yaml" \
  "${CHART_DIR}/values-external-iam.yaml" \
  "${CHART_DIR}/values-internal-iam.yaml"

if ! command -v helm >/dev/null 2>&1; then
  echo "helm not installed; skipped helm lint/template"
  exit 0
fi

helm lint "${CHART_DIR}"
helm template flowgo "${CHART_DIR}" -f "${CHART_DIR}/values-external-iam.yaml" >/tmp/flowgo-external.yaml
helm template flowgo "${CHART_DIR}" -f "${CHART_DIR}/values-internal-iam.yaml" >/tmp/flowgo-internal.yaml

echo "helm validation passed"
