#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

compare_generated() {
  local expected="$1"
  local generated="$2"
  if ! cmp -s "${expected}" "${generated}"; then
    echo "generated protobuf drift: ${expected#${ROOT}/}" >&2
    diff -u "${expected}" "${generated}" || true
    return 1
  fi
}

command -v docker >/dev/null 2>&1 || {
  echo "docker is required for deterministic Go protobuf drift validation" >&2
  exit 1
}

mkdir -p "${TMP_DIR}/go"
docker run --rm \
  -v "${ROOT}:/workspace:ro" \
  -v "${TMP_DIR}/go:/out" \
  -w /workspace \
  rvolosatovs/protoc:4.0.0 \
  --proto_path=. \
  --go_out=/out --go_opt=module=github.com/artificialflow/artificialflow \
  --go-grpc_out=/out --go-grpc_opt=module=github.com/artificialflow/artificialflow \
  backend/api/proto/job_worker_service.proto \
  backend/api/proto/events.proto

for file in \
  events.pb.go \
  job_worker_service.pb.go \
  job_worker_service_grpc.pb.go; do
  compare_generated \
    "${ROOT}/backend/api/v1/go/${file}" \
    "${TMP_DIR}/go/backend/api/v1/go/${file}"
done

SDK_DIR="${ROOT}/clients/nodejs-sdk"
PROTOC="${SDK_DIR}/node_modules/.bin/grpc_tools_node_protoc"
PROTOC_GEN_TS="${SDK_DIR}/node_modules/.bin/protoc-gen-ts"
if [[ ! -x "${PROTOC}" || ! -x "${PROTOC_GEN_TS}" ]]; then
  echo "run npm --prefix clients/nodejs-sdk ci before protobuf drift validation" >&2
  exit 1
fi

mkdir -p "${TMP_DIR}/node"
"${PROTOC}" \
  --plugin="protoc-gen-ts=${PROTOC_GEN_TS}" \
  --ts_out="grpc_js:${TMP_DIR}/node" \
  --js_out="import_style=commonjs,binary:${TMP_DIR}/node" \
  --grpc_out="grpc_js:${TMP_DIR}/node" \
  -I "${ROOT}/backend/api/proto" \
  "${ROOT}/backend/api/proto/job_worker_service.proto"

for file in \
  job_worker_service_grpc_pb.d.ts \
  job_worker_service_grpc_pb.js \
  job_worker_service_pb.d.ts \
  job_worker_service_pb.js; do
  compare_generated \
    "${SDK_DIR}/src/proto/${file}" \
    "${TMP_DIR}/node/${file}"
done

echo "Generated Go and Node.js protobuf artifacts match their sources"
