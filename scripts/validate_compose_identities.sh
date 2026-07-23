#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

tmp_json="$(mktemp)"
trap 'rm -f "${tmp_json}"' EXIT

validate_config() {
  local identity="$1"
  local compose_file="$2"
  local release_overlay="$3"
  local expected_project="$4"
  local expected_registry="$5"
  local expected_volumes="$6"
  shift 6

  local -a compose_args=(-f "${compose_file}")
  if [[ "${release_overlay}" == "true" ]]; then
    compose_args+=(-f docker-compose.release.yml)
  fi
  if [[ "${compose_file}" == "docker-compose.yml" ]]; then
    compose_args+=(--profile full-cqrs)
  fi

  env "$@" docker compose "${compose_args[@]}" config --format json > "${tmp_json}"
  python3 - "${tmp_json}" "${identity}" "${expected_project}" "${expected_registry}" "${expected_volumes}" <<'PY'
import json
import sys

path, identity, expected_project, expected_registry, raw_volumes = sys.argv[1:]
with open(path, "r", encoding="utf-8") as handle:
    config = json.load(handle)

errors = []
if config.get("name") != expected_project:
    errors.append(f"project name {config.get('name')!r}, expected {expected_project!r}")

for service_name, service in config.get("services", {}).items():
    if service.get("container_name"):
        errors.append(f"{service_name} hard-codes container_name={service['container_name']!r}")

if expected_registry != "-":
    expected_images = {
        "app": "workflow-command",
        "workflow-runtime": "workflow-runtime",
        "workflow-query": "workflow-query",
        "sync-worker": "sync-worker",
        "frontend": "frontend",
    }
    for service_name, image_name in expected_images.items():
        actual = config.get("services", {}).get(service_name, {}).get("image")
        prefix = f"{expected_registry}/{image_name}:"
        if not str(actual).startswith(prefix):
            errors.append(f"{service_name} image {actual!r}, expected prefix {prefix!r}")

expected_volumes = {}
for item in filter(None, raw_volumes.split(",")):
    key, value = item.split("=", 1)
    expected_volumes[key] = value

actual_volumes = config.get("volumes", {})
for key, expected_name in expected_volumes.items():
    actual_name = actual_volumes.get(key, {}).get("name")
    if actual_name != expected_name:
        errors.append(f"volume {key} name {actual_name!r}, expected {expected_name!r}")

if errors:
    print(f"compose identity validation failed for {identity}:", file=sys.stderr)
    for error in errors:
        print(f"  - {error}", file=sys.stderr)
    sys.exit(1)
PY
}

canonical_env=(
  COMPOSE_PROJECT_NAME=artificialflow
  ARTIFICIALFLOW_COMPOSE_PROJECT_NAME=artificialflow
  ARTIFICIALFLOW_PGDATA_VOLUME=artificialflow-postgres-data
  ARTIFICIALFLOW_ESDATA_VOLUME=artificialflow-elasticsearch-data
  ARTIFICIALFLOW_ZITADEL_PGDATA_VOLUME=artificialflow-zitadel-postgres-data
  ARTIFICIALFLOW_ZITADEL_BOOTSTRAP_VOLUME=artificialflow-zitadel-system-bootstrap
  ARTIFICIALFLOW_BOOTSTRAP_VOLUME=artificialflow-zitadel-bootstrap
  ARTIFICIALFLOW_AUTH_VOLUME=artificialflow-zitadel-auth
  ARTIFICIALFLOW_ZITADEL_NETWORK=artificialflow-zitadel
  ARTIFICIALFLOW_IMAGE_REGISTRY=artificialflow
)


base_volumes="pgdata=artificialflow-postgres-data,esdata=artificialflow-elasticsearch-data"
zitadel_volumes="${base_volumes},zitadel-postgres-data=artificialflow-zitadel-postgres-data,zitadel-bootstrap=artificialflow-zitadel-system-bootstrap,application-bootstrap=artificialflow-zitadel-bootstrap,application-auth=artificialflow-zitadel-auth"

for release_overlay in false true; do
  canonical_registry="-"
  if [[ "${release_overlay}" == "true" ]]; then
    canonical_registry=artificialflow
  fi

  validate_config "canonical base release=${release_overlay}" docker-compose.yml "${release_overlay}" artificialflow "${canonical_registry}" "${base_volumes}" "${canonical_env[@]}"
  validate_config "canonical external release=${release_overlay}" docker-compose.external-iam.yml "${release_overlay}" artificialflow "${canonical_registry}" "${base_volumes}" "${canonical_env[@]}"
  validate_config "canonical zitadel release=${release_overlay}" docker-compose.zitadel.yml "${release_overlay}" artificialflow "${canonical_registry}" "${zitadel_volumes}" "${canonical_env[@]}"
done

echo "compose identity validation passed"
