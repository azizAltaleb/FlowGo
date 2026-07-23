import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

import { validateTransitionCompatibility } from "./validate_transition_compatibility.mjs";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");

test("empty transition readers pass for current and later releases", () => {
  for (const version of ["0.3.0", "0.3.99", "0.4.0", "0.4.0-rc.1", "1.0.0"]) {
    assert.deepEqual(validateTransitionCompatibility(root, version), []);
  }
});

test("retained transition readers block their removal release and later releases", () => {
  const tempRoot = fs.mkdtempSync(path.join(os.tmpdir(), "transition-compat-"));
  fs.mkdirSync(path.join(tempRoot, "docs"), { recursive: true });
  fs.writeFileSync(
    path.join(tempRoot, "docs/transition-compatibility.json"),
    JSON.stringify({
      schemaVersion: 1,
      readers: [
        {
          owner: "test",
          surface: "roles",
          legacyIdentifier: "example-legacy-role",
          removalVersion: "0.4.0",
        },
      ],
    }),
  );

  assert.deepEqual(validateTransitionCompatibility(tempRoot, "0.3.99"), []);
  for (const version of ["0.4.0", "0.4.1", "1.0.0"]) {
    assert.match(
      validateTransitionCompatibility(tempRoot, version).join("\n"),
      /cannot retain \d+ transition readers scheduled for removal in 0\.4\.0/,
    );
  }
});
