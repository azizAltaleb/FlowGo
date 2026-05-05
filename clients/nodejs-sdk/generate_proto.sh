#!/bin/bash
set -e

# Generate Typescript and JS code
# Ensure output directory exists
mkdir -p src/proto

# Define paths
PROTOC_GEN_TS="./node_modules/.bin/protoc-gen-ts"
PROTO_DIR="../../backend/api/proto"
OUT_DIR="./src/proto"

# Run protoc
npx grpc_tools_node_protoc \
    --plugin="protoc-gen-ts=${PROTOC_GEN_TS}" \
    --ts_out="grpc_js:${OUT_DIR}" \
    --js_out="import_style=commonjs,binary:${OUT_DIR}" \
    --grpc_out="grpc_js:${OUT_DIR}" \
    -I "${PROTO_DIR}" \
    "${PROTO_DIR}/job_worker_service.proto"

echo "Node.js SDK code generated successfully."
