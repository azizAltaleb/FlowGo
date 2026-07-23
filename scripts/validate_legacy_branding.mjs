#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";
import process from "node:process";
import { execFileSync } from "node:child_process";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const legacyPattern = new RegExp(["Flow", "Go|flow", "go|FLOW", "GO"].join(""), "g");

// Historical changelog entries may mention the previous brand name.
const allowedFiles = new Set(["CHANGELOG.md"]);

const ignoredPathPrefixes = [
  ".git/",
  "node_modules/",
  "frontend/node_modules/",
  "clients/nodejs-sdk/node_modules/",
  "clients/nodejs-sdk/dist/",
  "tests/e2e/playwright/node_modules/",
];

function listedFiles() {
  const output = execFileSync(
    "git",
    ["ls-files", "-z", "--cached", "--others", "--exclude-standard"],
    { cwd: root },
  ).toString("utf8");
  return output.split("\0").filter(Boolean);
}

function shouldIgnore(relativePath) {
  return ignoredPathPrefixes.some((prefix) => relativePath === prefix.slice(0, -1) || relativePath.startsWith(prefix));
}

const failures = [];
for (const relativePath of listedFiles()) {
  if (shouldIgnore(relativePath) || allowedFiles.has(relativePath)) {
    continue;
  }
  const absolutePath = path.join(root, relativePath);
  let content;
  try {
    content = fs.readFileSync(absolutePath, "utf8");
  } catch {
    continue;
  }
  const lines = content.split(/\r?\n/);
  lines.forEach((line, index) => {
    if (legacyPattern.test(line)) {
      failures.push(`${relativePath}:${index + 1}: ${line.trim()}`);
    }
    legacyPattern.lastIndex = 0;
  });
}

if (failures.length > 0) {
  console.error("Unexpected pre-rename product references found:");
  for (const failure of failures.slice(0, 50)) {
    console.error(`- ${failure}`);
  }
  if (failures.length > 50) {
    console.error(`... and ${failures.length - 50} more`);
  }
  process.exit(1);
}

console.log("Legacy branding allowlist validation passed");
