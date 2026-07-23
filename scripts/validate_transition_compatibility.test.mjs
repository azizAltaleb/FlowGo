import assert from "node:assert/strict";
import path from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

import { validateTransitionCompatibility } from "./validate_transition_compatibility.mjs";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");

test("transition readers are allowed before their removal release", () => {
  assert.deepEqual(validateTransitionCompatibility(root, "0.3.99"), []);
  assert.deepEqual(validateTransitionCompatibility(root, "0.4.0-rc.1"), []);
});

test("transition readers block their removal release and later releases", () => {
  for (const version of ["0.4.0", "0.4.1", "1.0.0"]) {
    assert.match(
      validateTransitionCompatibility(root, version).join("\n"),
      /cannot retain \d+ transition readers scheduled for removal in 0\.4\.0/,
    );
  }
});
