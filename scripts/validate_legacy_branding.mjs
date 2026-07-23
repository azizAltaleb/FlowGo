#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";
import process from "node:process";
import { execFileSync } from "node:child_process";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const legacyPattern = new RegExp(["Flow", "Go|flow", "go|FLOW", "GO"].join(""), "g");

// Compatibility tests and fixtures are allowed to spell old identifiers so they
// can prove the one-release read path. Product code does not get this broad rule.
const compatibilityTestFiles = new Set([
  "backend/libs/auth/claims_mapper_test.go",
  "backend/libs/auth/grpc_test.go",
  "backend/libs/auth/introspection_verifier_test.go",
  "backend/libs/auth/roles_test.go",
  "backend/libs/iam/manager_test.go",
  "backend/libs/iam/zitadel_management_client_test.go",
  "backend/libs/metrics/outbox_test.go",
  "backend/services/sync-worker/cmd/connector_bootstrap_test.go",
  "backend/services/sync-worker/cmd/main_test.go",
  "backend/services/sync-worker/internal/application/service_test.go",
  "backend/services/sync-worker/internal/infrastructure/messaging/kafka_consumer_test.go",
  "backend/services/workflow-command/internal/domain/bpmn/parser_test.go",
  "backend/services/workflow-command/internal/infrastructure/persistence/gorm_repository_test.go",
  "backend/services/workflow-command/internal/interfaces/http/handler_test.go",
  "backend/services/workflow-command/internal/interfaces/http/identity_management_key_test.go",
  "backend/services/workflow-command/tests/engine_bpmn_xml_elements_test.go",
  "backend/services/workflow-command/tests/engine_bpmn_xml_regression_test.go",
  "backend/services/workflow-query/internal/infrastructure/persistence/es_repository_test.go",
  "backend/services/workflow-query/internal/interfaces/http/handler_test.go",
  "clients/nodejs-sdk/test/client.test.js",
  "frontend/src/lib/bpmn-parser.test.ts",
  "frontend/src/lib/cqrsSync.test.ts",
  "frontend/src/lib/roles.test.ts",
  "scripts/bootstrap_zitadel.test.mjs",
  "scripts/test_persistent_migrations.sh",
  "scripts/qa/run_uat_video_suite.sh",
  "tests/bpmn/matrix/scenarios.yml",
  "tests/e2e/playwright/fixtures/uat/bpmn.ts",
  "tests/e2e/playwright/specs/uat-functional-video.spec.ts",
  "tests/performance/k6/scenarios/worker_activate_jobs.js",
  "tests/performance/k6/scenarios/workflow_throughput.js",
]);

// These documents are operational migration instructions, not current branding.
const migrationDocuments = new Set([
  "docs/ARTIFICIALFLOW_MIGRATION.md",
  "docs/COMPATIBILITY_MATRIX.md",
  "docs/PERSISTENT_DATA_RENAME_MIGRATION.md",
  "docs/transition-compatibility.json",
]);

// Each production exception is an exact path plus the line shapes it may use.
// Additions should describe an actual transition reader or compatibility writer.
const productionRules = new Map([
  [
    "backend/api/v1/go/job_worker_service_legacy.go",
    [/LegacyJobWorker/, /legacy service name/, /pre-migration service name/, /api\.v1\.JobWorkerService/],
  ],
  [
    "backend/libs/auth/roles.go",
    [
      /\bLegacyRoleFlowGo(?:Admin|Modeler|Client)\b/,
      /^\s*RoleFlowGo(?:Admin|Modeler|Client)\s*=\s*RoleArtificialFlow/,
      /flowgo (?:admin|modeler|client)/i,
    ],
  ],
  [
    "backend/libs/iam/zitadel_management.go",
    [/\bFLOWGO_[A-Z0-9_]+\b/],
  ],
  [
    "backend/libs/iam/zitadel_management_client.go",
    [/flowgo-bootstrap/, /flowgo-client:/],
  ],
  [
    "backend/libs/metrics/outbox.go",
    [/legacy/i, /flowgo_outbox_/],
  ],
  [
    "backend/services/sync-worker/cmd/main.go",
    [/legacy/i, /flowgo(?:\b|[-_.])/],
  ],
  [
    "backend/services/workflow-command/cmd/server/main.go",
    [/X-FlowGo-Acting-/, /RegisterLegacyJobWorker/],
  ],
  [
    "backend/services/workflow-command/internal/application/jobs.go",
    [/LegacyUserTaskJobType/, /flowgo:userTask/],
  ],
  [
    "backend/services/workflow-command/internal/domain/bpmn/parser.go",
    [/LegacyFlowGo/, /flowgo\.com/],
  ],
  [
    "backend/services/workflow-command/internal/infrastructure/persistence/gorm_repository.go",
    [/legacyUserTaskJobType/, /flowgo:userTask/],
  ],
  [
    "backend/services/workflow-command/internal/interfaces/http/handler.go",
    [/legacyActing/, /X-FlowGo-Acting-/],
  ],
  [
    "backend/services/workflow-command/internal/interfaces/http/identity_management_handler.go",
    [/legacy flowgo roles/i, /flowgo viewer/i],
  ],
  [
    "backend/services/workflow-query/cmd/server/main.go",
    [/X-FlowGo-Acting-/],
  ],
  [
    "backend/services/workflow-query/internal/infrastructure/persistence/es_repository.go",
    [/legacy/i, /flowgo-/],
  ],
  [
    "clients/nodejs-sdk/src/Client.ts",
    [/FlowGo(?:ApiError|Client)/, /createFlowGo/, /listFlowGo/],
  ],
  [
    "clients/nodejs-sdk/src/Environment.ts",
    [/\bFLOWGO_[A-Z0-9_]+\b/],
  ],
  [
    "clients/nodejs-sdk/src/types.ts",
    [/FlowGo(?:AuthOptions|ClientBaseOptions|ClientOptions)/],
  ],
  [
    "frontend/entrypoint.sh",
    [/\bFLOWGO_[A-Z0-9_]+\b/, /__FLOWGO_RUNTIME_CONFIG__/],
  ],
  [
    "frontend/src/lib/bpmn-namespaces.ts",
    [/LEGACY_FLOWGO/, /flowgo\.com/, /flowgo[:_"]/],
  ],
  [
    "frontend/src/lib/bpmn-parser.ts",
    [/@_flowgo/],
  ],
  [
    "frontend/src/lib/cqrsSync.ts",
    [/legacy/i, /flowgo\.pending/],
  ],
  [
    "frontend/src/lib/roles.ts",
    [
      /\bLEGACY_FLOWGO_(?:ADMIN|MODELER|CLIENT)_ROLE\b/,
      /^export const FLOWGO_(?:ADMIN|MODELER|CLIENT)_ROLE = ARTIFICIALFLOW_/,
      /^export const STATIC_FLOWGO_ROLES = STATIC_ARTIFICIALFLOW_ROLES/,
      /flowgo (?:admin|modeler|client)/,
    ],
  ],
  [
    "frontend/src/lib/runtimeConfig.ts",
    [/__FLOWGO_RUNTIME_CONFIG__/],
  ],
  [
    "frontend/src/pages/IdentityManagement.tsx",
    [/FLOWGO_VIEWER_ROLE/, /flowgo (?:viewer|roles)/i],
  ],
  [
    "charts/artificialflow/files/bootstrap_zitadel.mjs",
    [/\bFLOWGO_[A-Z0-9_]+\b/, /LEGACY_.*NAMES/, /FlowGo (?:Frontend|API)/, /"FlowGo"/, /flowgo (?:admin|modeler|client)/],
  ],
  [
    "charts/artificialflow/templates/_helpers.tpl",
    [/compatibility/, /flowgo/i],
  ],
  [
    "charts/artificialflow/values-legacy-persistent-identifiers.yaml",
    [/legacy/i, /flowgo/i],
  ],
  [
    "charts/artificialflow/values.yaml",
    [/legacy/i, /former flowgo/i],
  ],
  [
    "charts/artificialflow/README.md",
    [
      /Legacy `flowgo \.\.\.` roles remain accepted/,
      /upgrade a `flowgo` release/,
      /^`flowgo`, render/,
      /helm template flowgo/,
      /tmp\/flowgo-upgrade/,
      /compatibility file preserves `flowgo` fullnames/,
      /https:\/\/flowgo\.example\.com/,
      /\/flowgo\/bootstrap\/flowgo-frontend-client-id/,
    ],
  ],
  [
    "clients/nodejs-sdk/README.md",
    [/@flowgo\/nodejs-sdk/, /FlowGo(?:Client|ApiError|\*Options)/, /\bFLOWGO_[A-Z0-9_]+\b/],
  ],
  [
    "docs/RELEASE_CHECKLIST.md",
    [/@flowgo\/nodejs-sdk/],
  ],
  [
    "docs/deployment.md",
    [
      /^export ARTIFICIALFLOW_.*flowgo/,
      /standard former `flowgo` project/,
      /Changing from legacy `flowgo` resource names/,
    ],
  ],
  [
    "docs/sdk-nodejs.md",
    [/@flowgo\/nodejs-sdk/, /FlowGo(?:Client|ApiError|\*Options)/, /\bFLOWGO_[A-Z0-9_]+\b/],
  ],
  [
    "docs/troubleshooting.md",
    [/legacy internal mount paths remain stable/],
  ],
  [
    "README.md",
    [/Existing FlowGo installations must set the exact old/],
  ],
  [
    "docker-compose.release.yml",
    [/\bFLOWGO_IMAGE_(?:REGISTRY|TAG)\b/],
  ],
  [
    "docker-compose.zitadel.yml",
    [/\bFLOWGO_[A-Z0-9_]+\b/],
  ],
  [
    ".github/workflows/release-npm.yml",
    [/@flowgo\/nodejs-sdk/],
  ],
  [
    "scripts/assert_helm_render_compatibility.rb",
    [/legacy/i, /flowgo/i],
  ],
  [
    "scripts/bootstrap_zitadel.mjs",
    [/\bFLOWGO_[A-Z0-9_]+\b/, /LEGACY_.*NAMES/, /FlowGo (?:Frontend|API)/, /"FlowGo"/, /flowgo (?:admin|modeler|client)/],
  ],
  [
    "scripts/migrate_database_and_state.sh",
    [/OLD_DB_ROLE/, /flowgo/],
  ],
  [
    "scripts/migrate_es_prefix.sh",
    [/OLD_PREFIX/, /flowgo/],
  ],
  [
    "scripts/migrate_streaming_identifiers.sh",
    [/OLD_(?:CONNECTOR|GROUP|SLOT)/, /flowgo/],
  ],
  [
    "scripts/validate_migration_release.sh",
    [/TestIdentityManagementProtectsFlowGoPlatformRoles/],
  ],
  [
    "scripts/test-all-functionality.sh",
    [/\bFLOWGO_[A-Z0-9_]+\b/],
  ],
  [
    "scripts/validate_compose_identities.sh",
    [/legacy/i, /flowgo/i],
  ],
  [
    "scripts/validate_helm.sh",
    [/legacy/i, /flowgo/i],
  ],
  [
    "scripts/validate_nodejs_sdk_package.mjs",
    [/@flowgo\/nodejs-sdk/],
  ],
  [
    "scripts/validate_release_version.mjs",
    [/FLOWGO_IMAGE_TAG/],
  ],
]);

function repositoryFiles() {
  const output = execFileSync(
    "git",
    ["ls-files", "--cached", "--others", "--exclude-standard", "-z"],
    { cwd: root },
  ).toString("utf8");
  return [...new Set(output.split("\0").filter(Boolean))].sort();
}

function allowedByHistoricalChangelog(file, lineNumber, content) {
  if (file !== "CHANGELOG.md") return false;
  const lines = content.split(/\r?\n/);
  const historicalStart = lines.findIndex((line) => /^## \[0\.1\.1\]/.test(line));
  return historicalStart >= 0 && lineNumber > historicalStart;
}

function allowed(file, line, lineNumber, content) {
  // The validator necessarily contains the narrow patterns it enforces.
  if (file === "scripts/validate_legacy_branding.mjs") return true;
  if (compatibilityTestFiles.has(file)) return true;
  if (migrationDocuments.has(file)) return true;
  if (file.startsWith("clients/nodejs-sdk-legacy/")) return true;
  if (allowedByHistoricalChangelog(file, lineNumber, content)) return true;

  const patterns = productionRules.get(file);
  return Boolean(patterns?.some((pattern) => pattern.test(line)));
}

const failures = [];
for (const file of repositoryFiles()) {
  const absolute = path.join(root, file);
  if (!fs.existsSync(absolute) || !fs.statSync(absolute).isFile()) continue;

  const buffer = fs.readFileSync(absolute);
  if (buffer.includes(0)) continue;
  const content = buffer.toString("utf8");
  const lines = content.split(/\r?\n/);

  lines.forEach((line, index) => {
    legacyPattern.lastIndex = 0;
    if (!legacyPattern.test(line)) return;
    if (!allowed(file, line, index + 1, content)) {
      failures.push(`${file}:${index + 1}: ${line.trim()}`);
    }
  });
}

if (failures.length > 0) {
  console.error("Unexpected pre-rename product references found:");
  failures.forEach((failure) => console.error(`- ${failure}`));
  console.error("Add a narrow, commented compatibility rule only for an intentional transition reader.");
  process.exit(1);
}

console.log("Legacy branding allowlist validation passed");
