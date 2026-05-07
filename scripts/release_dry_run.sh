#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RUN_NPM_CHECKS="${RUN_NPM_CHECKS:-true}"
RUN_DOCKER_BUILDS="${RUN_DOCKER_BUILDS:-true}"
VERSION="${VERSION:-0.1.0-dry-run}"
REVISION="${REVISION:-$(git -C "${ROOT_DIR}" rev-parse --short HEAD 2>/dev/null || echo unknown)}"

run() {
  echo "+ $*"
  "$@"
}

run_quiet() {
  echo "+ $* > /dev/null"
  "$@" > /dev/null
}

run_in() {
  local dir="$1"
  shift
  echo "+ (cd ${dir} && $*)"
  (cd "${dir}" && "$@")
}

cd "${ROOT_DIR}"

run bash -n scripts/init_connector.sh
run bash -n scripts/cqrs_parity_check.sh
run bash -n scripts/cqrs_e2e_smoke.sh
run bash -n scripts/worker_conformance_smoke.sh
run bash -n scripts/validate_helm.sh
run_quiet docker compose -f docker-compose.external-iam.yml config
run_quiet docker compose -f docker-compose.zitadel.yml config
run_quiet docker compose -f docker-compose.external-iam.yml -f docker-compose.release.yml config
run_quiet docker compose -f docker-compose.zitadel.yml -f docker-compose.release.yml config
run bash scripts/validate_helm.sh
run node scripts/validate_nodejs_sdk_package.mjs

if [[ "${RUN_NPM_CHECKS}" == "true" ]]; then
  run npm --prefix frontend ci
  run npm --prefix frontend run lint
  run npm --prefix frontend test
  run npm --prefix frontend run build
  run npm --prefix clients/nodejs-sdk ci
  run npm --prefix clients/nodejs-sdk test
  run npm --prefix clients/nodejs-sdk run validate:package
  run_in clients/nodejs-sdk npm pack --dry-run
  echo "+ (cd clients/nodejs-sdk && npm sbom --sbom-format cyclonedx --omit dev > /tmp/goflow-nodejs-sdk-sbom.cdx.json)"
  (cd clients/nodejs-sdk && npm sbom --sbom-format cyclonedx --omit dev > /tmp/goflow-nodejs-sdk-sbom.cdx.json)
fi

if [[ "${RUN_DOCKER_BUILDS}" == "true" ]]; then
  run docker build -f backend/Dockerfile --build-arg "VERSION=${VERSION}" --build-arg "REVISION=${REVISION}" -t goflow/workflow-command:dry-run .
  run docker build -f backend/Dockerfile.runtime --build-arg "VERSION=${VERSION}" --build-arg "REVISION=${REVISION}" -t goflow/workflow-runtime:dry-run .
  run docker build -f backend/Dockerfile.workflow-query --build-arg "VERSION=${VERSION}" --build-arg "REVISION=${REVISION}" -t goflow/workflow-query:dry-run .
  run docker build -f backend/Dockerfile.sync-worker --build-arg "VERSION=${VERSION}" --build-arg "REVISION=${REVISION}" -t goflow/sync-worker:dry-run .
  run docker build -f frontend/Dockerfile --build-arg "VERSION=${VERSION}" --build-arg "REVISION=${REVISION}" -t goflow/frontend:dry-run frontend
fi

echo "Release dry run passed"
