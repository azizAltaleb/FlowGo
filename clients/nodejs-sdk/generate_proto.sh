#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${SCRIPT_DIR}"

# Generate Typescript and JS code
# Ensure output directory exists
mkdir -p src/proto

# Define paths
PROTOC="./node_modules/.bin/grpc_tools_node_protoc"
PROTOC_GEN_TS="./node_modules/.bin/protoc-gen-ts"
PROTO_DIR="../../backend/api/proto"
OUT_DIR="./src/proto"

# Run protoc
"${PROTOC}" \
    --plugin="protoc-gen-ts=${PROTOC_GEN_TS}" \
    --ts_out="grpc_js:${OUT_DIR}" \
    --js_out="import_style=commonjs,binary:${OUT_DIR}" \
    --grpc_out="grpc_js:${OUT_DIR}" \
    -I "${PROTO_DIR}" \
    "${PROTO_DIR}/job_worker_service.proto"

echo "Node.js SDK code generated successfully."
