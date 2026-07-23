#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

canonical_tgz="$(
  cd "${ROOT}/clients/nodejs-sdk"
  npm pack --silent --pack-destination "${TMP_DIR}"
)"
wrapper_tgz="$(
  cd "${ROOT}/clients/nodejs-sdk-legacy"
  npm pack --silent --pack-destination "${TMP_DIR}"
)"

cd "${TMP_DIR}"
npm init --yes >/dev/null
npm install --ignore-scripts --no-audit --no-fund "./${canonical_tgz}" >/dev/null
npm install --ignore-scripts --no-audit --no-fund "./${wrapper_tgz}" >/dev/null

node <<'NODE'
const assert = require("node:assert/strict");
const canonical = require("@artificialflow/nodejs-sdk");
const wrapperName = ["@", "flow", "go/nodejs-sdk"].join("");
const wrapper = require(wrapperName);

assert.equal(wrapper.ArtificialFlowClient, canonical.ArtificialFlowClient);
assert.equal(wrapper.ArtificialFlowApiError, canonical.ArtificialFlowApiError);
NODE

echo "Canonical SDK and compatibility wrapper pack/install validation passed"
