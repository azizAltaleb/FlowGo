#!/bin/bash
set -e

# Use a docker image that has protoc and go plugins
# rvolosatovs/protoc is a good option, or we can use a standard golang image and install tools.
# Let's use a specialized image for convenience: rvolosatovs/protoc

echo "Generating Protobuf code..."

# Ensure output directory exists
mkdir -p backend/api/v1/go

# Run protoc via Docker
docker run --rm \
  -v $(pwd):/workspace \
  -w /workspace \
  rvolosatovs/protoc:4.0.0 \
  --proto_path=. \
  --go_out=. --go_opt=paths=source_relative \
  --go-grpc_out=. --go-grpc_opt=paths=source_relative \
  backend/api/proto/job_worker_service.proto \
  backend/api/proto/events.proto

echo "Done."
