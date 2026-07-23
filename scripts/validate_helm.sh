#!/usr/bin/env bash
set -euo pipefail

CHART_DIR="${CHART_DIR:-charts/artificialflow}"

ruby -e 'require "yaml"; ARGV.each { |f| YAML.load_file(f); puts "ok #{f}" }' \
  "${CHART_DIR}/Chart.yaml" \
  "${CHART_DIR}/values.yaml" \
  "${CHART_DIR}/values-external-iam.yaml" \
  "${CHART_DIR}/values-internal-iam.yaml" \
  "${CHART_DIR}/values-legacy-persistent-identifiers.yaml"

ruby -e '
  require "yaml"
  external = YAML.load_file(ARGV[0])
  internal = YAML.load_file(ARGV[1])
  abort "values-external-iam.yaml must set iam.mode=external" unless external.dig("iam", "mode") == "external"
  abort "values-external-iam.yaml must keep zitadel.enabled=false" unless external.dig("zitadel", "enabled") == false
  abort "values-internal-iam.yaml must set iam.mode=zitadel" unless internal.dig("iam", "mode") == "zitadel"
  abort "values-internal-iam.yaml must set zitadel.enabled=true" unless internal.dig("zitadel", "enabled") == true
  puts "ok helm IAM mode values"
' "${CHART_DIR}/values-external-iam.yaml" "${CHART_DIR}/values-internal-iam.yaml"

if rg -n 'include "flowgo\.|define "flowgo\.' "${CHART_DIR}"; then
  echo "legacy flowgo Helm helper names remain in ${CHART_DIR}" >&2
  exit 1
fi

if ! command -v helm >/dev/null 2>&1; then
  echo "helm not installed; skipped helm lint/template"
  exit 0
fi

helm lint "${CHART_DIR}"
helm template artificialflow "${CHART_DIR}" -f "${CHART_DIR}/values-external-iam.yaml" >/tmp/artificialflow-external.yaml
helm template artificialflow "${CHART_DIR}" -f "${CHART_DIR}/values-internal-iam.yaml" >/tmp/artificialflow-internal.yaml
helm template flowgo "${CHART_DIR}" \
  -f "${CHART_DIR}/values-external-iam.yaml" \
  -f "${CHART_DIR}/values-legacy-persistent-identifiers.yaml" \
  >/tmp/artificialflow-legacy-external.yaml
helm template flowgo "${CHART_DIR}" \
  -f "${CHART_DIR}/values-internal-iam.yaml" \
  -f "${CHART_DIR}/values-legacy-persistent-identifiers.yaml" \
  >/tmp/artificialflow-legacy-internal.yaml
ruby scripts/assert_helm_render_compatibility.rb \
  /tmp/artificialflow-external.yaml \
  /tmp/artificialflow-internal.yaml \
  /tmp/artificialflow-legacy-external.yaml \
  /tmp/artificialflow-legacy-internal.yaml

echo "helm validation passed"
