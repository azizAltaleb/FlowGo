#!/usr/bin/env node
import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import { fileURLToPath } from 'node:url';

const scriptDir = path.dirname(fileURLToPath(import.meta.url));
const candidates = [
  process.cwd(),
  path.join(process.cwd(), 'clients', 'nodejs-sdk'),
  path.join(scriptDir, '..', 'clients', 'nodejs-sdk'),
];

const packageDir = candidates.find((candidate) => {
  const packagePath = path.join(candidate, 'package.json');
  if (!fs.existsSync(packagePath)) {
    return false;
  }
  const pkg = JSON.parse(fs.readFileSync(packagePath, 'utf8'));
  return pkg.name === '@goflow/nodejs-sdk';
});

if (!packageDir) {
  console.error('Could not locate clients/nodejs-sdk package.json');
  process.exit(1);
}

const packagePath = path.join(packageDir, 'package.json');
const pkg = JSON.parse(fs.readFileSync(packagePath, 'utf8'));
const failures = [];

function requireEqual(field, actual, expected) {
  if (actual !== expected) {
    failures.push(`${field} must be ${JSON.stringify(expected)}; got ${JSON.stringify(actual)}`);
  }
}

function requirePresent(field, value) {
  if (value === undefined || value === null || value === '') {
    failures.push(`${field} is required`);
  }
}

function requireArrayIncludes(field, values, expectedValues) {
  if (!Array.isArray(values)) {
    failures.push(`${field} must be an array`);
    return;
  }
  for (const expected of expectedValues) {
    if (!values.includes(expected)) {
      failures.push(`${field} must include ${expected}`);
    }
  }
}

requireEqual('name', pkg.name, '@goflow/nodejs-sdk');
requireEqual('license', pkg.license, 'MIT');
requireEqual('main', pkg.main, 'dist/index.js');
requireEqual('types', pkg.types, 'dist/index.d.ts');
requirePresent('description', pkg.description);
requirePresent('author', pkg.author);
requirePresent('homepage', pkg.homepage);
requirePresent('repository.url', pkg.repository?.url);
requireEqual('repository.directory', pkg.repository?.directory, 'clients/nodejs-sdk');
requirePresent('bugs.url', pkg.bugs?.url);
requireEqual('publishConfig.access', pkg.publishConfig?.access, 'public');

if (!/^\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?$/.test(pkg.version || '')) {
  failures.push('version must be a valid SemVer value');
}

if (!String(pkg.engines?.node || '').includes('>=20')) {
  failures.push('engines.node must require Node.js 20 or newer');
}

requireArrayIncludes('keywords', pkg.keywords, ['goflow', 'workflow', 'bpmn', 'worker', 'sdk', 'oidc']);
requireArrayIncludes('files', pkg.files, ['dist', 'examples', 'README.md', 'LICENSE']);

const requiredScripts = ['build', 'test', 'validate:package', 'prepack'];
for (const scriptName of requiredScripts) {
  requirePresent(`scripts.${scriptName}`, pkg.scripts?.[scriptName]);
}

const requiredFiles = ['README.md', 'LICENSE', 'examples/sdk-smoke-test.js', 'src/index.ts'];
for (const requiredFile of requiredFiles) {
  if (!fs.existsSync(path.join(packageDir, requiredFile))) {
    failures.push(`required package file is missing: ${requiredFile}`);
  }
}

if (failures.length > 0) {
  console.error('Node.js SDK package metadata validation failed:');
  for (const failure of failures) {
    console.error(`- ${failure}`);
  }
  process.exit(1);
}

console.log(`Node.js SDK package metadata validation passed for ${pkg.name}@${pkg.version}`);
