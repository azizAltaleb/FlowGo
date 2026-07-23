#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT}"

run() {
  echo "+ $*"
  "$@"
}

for command in go node npm docker helm actionlint; do
  command -v "${command}" >/dev/null 2>&1 || {
    echo "${command} is required for the migration release gate" >&2
    exit 1
  }
done

run node scripts/validate_legacy_branding.mjs
run node scripts/validate_transition_compatibility.mjs
run node scripts/validate_release_version.mjs
run node scripts/validate_release_workflows.mjs
run node --test scripts/validate_transition_compatibility.test.mjs

while IFS= read -r script; do
  run bash -n "${script}"
done < <(git ls-files 'scripts/*.sh' 'scripts/**/*.sh')

run actionlint -color=false
run node --test scripts/bootstrap_zitadel.test.mjs
run bash scripts/test_persistent_migrations.sh

run go test ./backend/api/v1/go -run 'TestCanonicalAndLegacyJobWorkerServicePaths|TestProtobufNamespaceAndLegacyWireCompatibility' -count=1
run go test ./backend/services/workflow-command/internal/domain/bpmn -run 'TestParse_ArtificialFlowNamespaceCompatibilityAndPrecedence|TestParse_CanonicalizesExtensionPropertyAliases' -count=1
run go test ./backend/services/workflow-command/tests -run 'TestDeployAndExecuteCanonicalArtificialFlowBPMNAttributes|TestDeployWorkflowFromBPMN_ServiceTaskExtensionPropertyMapsImplementation|TestDeployWorkflowFromBPMN_UserTaskAssignmentFromExtensionProperties' -count=1
run go test ./backend/libs/auth -run 'TestCanonicalizeRolesMigratesLegacyRolesAndPreservesCustomRoles|TestPrincipalFromClaims_CanonicalizesMixedStandardRoleClaims|TestUnaryServerInterceptor' -count=1
run go test ./backend/libs/iam -run 'TestResolveZITADELManagementConfigPrefersCanonicalEnvironment|TestClientDescriptionUsesCanonicalPrefixAndReadsBothPrefixes|TestNormalizeRoleKeysCanonicalizesLegacyPlatformRoles' -count=1
run go test ./backend/services/workflow-command/internal/interfaces/http -run 'TestActingPrincipalFromRequestPrefersCanonicalHeadersAndCanonicalizesRoles|TestActingPrincipalFromRequestAcceptsLegacyHeaders|TestIdentityManagementProtectsFlowGoPlatformRoles' -count=1
run go test ./backend/services/workflow-command/internal/infrastructure/persistence -run 'TestUserTaskQueriesIncludeCanonicalAndLegacyTypes' -count=1
run go test ./backend/services/workflow-query/internal/infrastructure/persistence -run 'TestNewESRepositoryDefaultsToCanonicalPrefix|TestESRepository' -count=1
run go test ./backend/libs/metrics -run 'TestOutboxCollectorExposesCanonicalAndLegacyMirrors' -count=1
run go test ./backend/services/sync-worker/cmd -run 'TestEnsureConnectorBootstrap|TestBuildConnectorCreateRequest|TestConflictingConnectorNames' -count=1

if [[ ! -d frontend/node_modules || ! -d clients/nodejs-sdk/node_modules ]]; then
  echo "run npm ci for frontend and clients/nodejs-sdk before the migration release gate" >&2
  exit 1
fi

run npm --prefix frontend test -- \
  src/lib/bpmn-parser.test.ts \
  src/lib/roles.test.ts \
  src/lib/runtimeConfig.test.ts \
  src/lib/cqrsSync.test.ts
run npm --prefix clients/nodejs-sdk test
run bash scripts/validate_nodejs_sdk_install.sh
run bash scripts/validate_generated_proto.sh

run docker compose -f docker-compose.yml config --quiet
run docker compose -f docker-compose.external-iam.yml config --quiet
run docker compose -f docker-compose.zitadel.yml config --quiet
run docker compose -f docker-compose.yml -f docker-compose.release.yml config --quiet
run docker compose -f docker-compose.external-iam.yml -f docker-compose.release.yml config --quiet
run docker compose -f docker-compose.zitadel.yml -f docker-compose.release.yml config --quiet
run bash scripts/validate_compose_kafka_wiring.sh -f docker-compose.yml
run bash scripts/validate_compose_kafka_wiring.sh -f docker-compose.external-iam.yml
run bash scripts/validate_compose_kafka_wiring.sh -f docker-compose.zitadel.yml
run bash scripts/validate_compose_identities.sh
run bash scripts/validate_helm.sh

echo "ArtificialFlow migration release gate passed"
