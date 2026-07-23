#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";
import process from "node:process";
import { fileURLToPath } from "node:url";

const manifestPath = "docs/transition-compatibility.json";
const requiredRemovalVersion = "0.4.0";

function parseSemver(value) {
  const match = String(value || "").match(
    /^v?(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z.-]+))?$/,
  );
  if (!match) return null;
  return {
    numbers: match.slice(1, 4).map(Number),
    prerelease: match[4] || "",
  };
}

function compareSemver(left, right) {
  for (let index = 0; index < 3; index += 1) {
    if (left.numbers[index] !== right.numbers[index]) {
      return left.numbers[index] - right.numbers[index];
    }
  }
  if (left.prerelease === right.prerelease) return 0;
  if (!left.prerelease) return 1;
  if (!right.prerelease) return -1;
  return left.prerelease.localeCompare(right.prerelease);
}

export function validateTransitionCompatibility(root, releaseVersion) {
  const failures = [];
  const release = parseSemver(releaseVersion);
  if (!release) {
    return [`transition release version is not SemVer: ${JSON.stringify(releaseVersion)}`];
  }

  let manifest;
  try {
    manifest = JSON.parse(fs.readFileSync(path.join(root, manifestPath), "utf8"));
  } catch (error) {
    return [`${manifestPath} is not valid readable JSON: ${error.message}`];
  }

  if (manifest.schemaVersion !== 1) {
    failures.push(`${manifestPath} schemaVersion must be 1`);
  }
  if (!Array.isArray(manifest.readers)) {
    failures.push(`${manifestPath} readers must be an array`);
    return failures;
  }

  const identities = new Set();
  manifest.readers.forEach((reader, index) => {
    const label = `${manifestPath} readers[${index}]`;
    for (const field of ["owner", "surface", "legacyIdentifier", "removalVersion"]) {
      if (typeof reader?.[field] !== "string" || reader[field].trim() === "") {
        failures.push(`${label}.${field} must be a non-empty string`);
      }
    }
    if (reader?.removalVersion !== requiredRemovalVersion) {
      failures.push(`${label}.removalVersion must be ${requiredRemovalVersion}`);
    }

    const identity = `${reader?.owner}\u0000${reader?.surface}\u0000${reader?.legacyIdentifier}`;
    if (identities.has(identity)) {
      failures.push(`${label} duplicates another compatibility reader`);
    }
    identities.add(identity);
  });

  const removal = parseSemver(requiredRemovalVersion);
  if (
    failures.length === 0
    && manifest.readers.length > 0
    && compareSemver(release, removal) >= 0
  ) {
    failures.push(
      `release ${releaseVersion} cannot retain ${manifest.readers.length} transition readers scheduled for removal in ${requiredRemovalVersion}`,
    );
  }
  return failures;
}

function argumentValue(name) {
  const index = process.argv.indexOf(name);
  if (index >= 0) return process.argv[index + 1] || "";
  const inline = process.argv.find((argument) => argument.startsWith(`${name}=`));
  return inline ? inline.slice(name.length + 1) : "";
}

const currentFile = fileURLToPath(import.meta.url);
if (process.argv[1] && path.resolve(process.argv[1]) === currentFile) {
  const root = path.resolve(path.dirname(currentFile), "..");
  const releaseVersion = argumentValue("--version")
    || JSON.parse(fs.readFileSync(path.join(root, "clients/nodejs-sdk/package.json"), "utf8")).version;
  const failures = validateTransitionCompatibility(root, releaseVersion);
  if (failures.length > 0) {
    console.error("Transition compatibility validation failed:");
    failures.forEach((failure) => console.error(`- ${failure}`));
    process.exit(1);
  }
  console.log(`Transition compatibility validation passed for ${releaseVersion}`);
}
