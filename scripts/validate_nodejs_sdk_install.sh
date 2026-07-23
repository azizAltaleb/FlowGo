#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

canonical_tgz="$(
  cd "${ROOT}/clients/nodejs-sdk"
  npm pack --silent --pack-destination "${TMP_DIR}"
)"

cd "${TMP_DIR}"
npm init --yes >/dev/null
npm install --ignore-scripts --no-audit --no-fund "./${canonical_tgz}" >/dev/null

node <<'NODE'
const assert = require("node:assert/strict");
const sdk = require("@artificialflow/nodejs-sdk");
assert.equal(typeof sdk.ArtificialFlowClient, "function");
assert.equal(typeof sdk.ArtificialFlowApiError, "function");
NODE

echo "Canonical SDK pack/install validation passed"
