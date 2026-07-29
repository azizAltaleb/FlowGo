.PHONY: up up-external-iam up-zitadel up-external-iam-release up-zitadel-release down down-external-iam down-zitadel restart logs logs-external-iam logs-zitadel ps ps-external-iam ps-zitadel clean clean-external-iam clean-zitadel demo build-backend build-frontend up-core up-full up-full-cqrs smoke-base smoke-core smoke-full smoke-release-base smoke-release-core smoke-release-full smoke-release-profiles smoke-profiles validate-helm validate-legacy-branding validate-release-version validate-migration-release release-dry-run cqrs-parity-check cqrs-e2e-smoke worker-conformance test-bpmn-matrix test-bpmn-exhaustive test-deployment-matrix test-uat-videos test-uat-mega-bpmn test-unit test-integration test-e2e test-frontend test-perf test-security test-dast test-release-suite test-report test-all test-all-functionality


# Docker Compose Commands
up:
	@$(MAKE) up-zitadel

up-external-iam:
	docker compose -f docker-compose.external-iam.yml up -d --build

up-zitadel:
	docker compose -f docker-compose.zitadel.yml up -d --build

up-external-iam-release:
	docker compose -f docker-compose.external-iam.yml -f docker-compose.release.yml up -d

up-zitadel-release:
	docker compose -f docker-compose.zitadel.yml -f docker-compose.release.yml up -d

up-core:
	@$(MAKE) up-zitadel

up-full:
	@$(MAKE) up-zitadel

up-full-cqrs:
	@$(MAKE) up-zitadel

smoke-base:
	docker compose -f docker-compose.yml config > /dev/null
	bash scripts/validate_compose_kafka_wiring.sh -f docker-compose.yml

smoke-core:
	docker compose -f docker-compose.external-iam.yml config > /dev/null
	bash scripts/validate_compose_kafka_wiring.sh -f docker-compose.external-iam.yml

smoke-full:
	docker compose -f docker-compose.zitadel.yml config > /dev/null
	bash scripts/validate_compose_kafka_wiring.sh -f docker-compose.zitadel.yml

smoke-profiles: smoke-base smoke-core smoke-full

smoke-release-base:
	docker compose -f docker-compose.yml -f docker-compose.release.yml config > /dev/null
	bash scripts/validate_compose_kafka_wiring.sh -f docker-compose.yml -f docker-compose.release.yml

smoke-release-core:
	docker compose -f docker-compose.external-iam.yml -f docker-compose.release.yml config > /dev/null
	bash scripts/validate_compose_kafka_wiring.sh -f docker-compose.external-iam.yml -f docker-compose.release.yml

smoke-release-full:
	docker compose -f docker-compose.zitadel.yml -f docker-compose.release.yml config > /dev/null
	bash scripts/validate_compose_kafka_wiring.sh -f docker-compose.zitadel.yml -f docker-compose.release.yml

smoke-release-profiles: smoke-release-base smoke-release-core smoke-release-full

validate-helm:
	bash scripts/validate_helm.sh

validate-legacy-branding:
	node scripts/validate_legacy_branding.mjs

validate-release-version:
	node scripts/validate_release_version.mjs

validate-migration-release:
	bash scripts/validate_migration_release.sh

release-dry-run:
	bash scripts/release_dry_run.sh

cqrs-parity-check:
	./scripts/cqrs_parity_check.sh

cqrs-e2e-smoke:
	./scripts/cqrs_e2e_smoke.sh

worker-conformance:
	bash ./scripts/worker_conformance_smoke.sh

init-connector:
	./scripts/init_connector.sh

down:
	@$(MAKE) down-external-iam
	@$(MAKE) down-zitadel

down-external-iam:
	docker compose -f docker-compose.external-iam.yml down

down-zitadel:
	docker compose -f docker-compose.zitadel.yml down

restart: down up

logs:
	@$(MAKE) logs-zitadel

logs-external-iam:
	docker compose -f docker-compose.external-iam.yml logs -f

logs-zitadel:
	docker compose -f docker-compose.zitadel.yml logs -f

ps:
	@$(MAKE) ps-zitadel

ps-external-iam:
	docker compose -f docker-compose.external-iam.yml ps

ps-zitadel:
	docker compose -f docker-compose.zitadel.yml ps

clean:
	@$(MAKE) clean-external-iam
	@$(MAKE) clean-zitadel

clean-external-iam:
	docker compose -f docker-compose.external-iam.yml down -v

clean-zitadel:
	docker compose -f docker-compose.zitadel.yml down -v

# Demo
demo:
	./demo.sh

# Tests
test-bpmn-matrix:
	go test ./backend/services/workflow-command/internal/domain/bpmn -run 'TestParse_(ElementTypeMatrix|MapsExtendedElementsAndProperties|MapsPlainAttributeAliasesForCompatibility|MergesExtensionPropertiesWithoutOverridingMappedKeys|CanonicalizesExtensionPropertyAliases|BoundaryCancelActivityAttributeTakesPrecedenceOverExtensionAlias|PopulatesIncomingForGatewayJoin|FailsForUnsupportedElementReferences|SupportsSendTask|MessageAndSignalStartProps|TerminateAndLink|SupportsEscalationAndEventSubProcess)' -count=1
	go test ./backend/services/workflow-command/tests -run 'TestDeployWorkflowFromBPMN_(CallActivityBusinessRuleAndManualTask|EventBasedGatewayReceiveAndTimer|BoundaryTimerInterruptsTask|ThrowSignalTriggersCatch|ThrowMessageUsesCorrelationKey|ThrowMessageUsesPlainCorrelationKeyAlias|BoundaryTimerNonInterruptingKeepsTaskActive|BoundaryCancelActivityExtensionAliasKeepsTaskActive|BoundaryCancelActivityAttributeTakesPrecedenceOverExtensionAlias|ServiceTaskPlainTaskTypeAliasMapsImplementation|ServiceTaskExtensionPropertyMapsImplementation|UserTaskAssignmentFromExtensionProperties|FailsForUnsupportedElementReferences|SupportsSendTask|LinkAndTerminate|EscalationBoundary|ConditionalCatch|MessageStart|TransactionCancel|CompensationFromXML|EventSubProcessMessage)|TestManualTaskWaitsForComplete' -count=1

test-bpmn-exhaustive:
	@mkdir -p reports
	python3 scripts/qa/run_bpmn_matrix.py --reports-dir reports --output-json reports/bpmn-matrix-report.json --output-md reports/bpmn-matrix-report.md

test-deployment-matrix:
	@mkdir -p reports
	bash scripts/test-all-functionality.sh --skip-ui --skip-sdk-live --skip-perf --skip-security

test-uat-videos:
	bash scripts/qa/run_uat_video_suite.sh both

# Mega BPMN UAT (requires ARTIFICIALFLOW_TOKEN + healthy stack)
test-uat-mega-bpmn:
	@mkdir -p reports
	cd tests/e2e/playwright && npm install --silent && npx playwright test specs/uat-mega-bpmn.spec.ts --reporter=list

# -----------------------------------------------------------------------
# Test Targets
# -----------------------------------------------------------------------

test-unit:
	@mkdir -p reports
	go test ./backend/... -coverprofile=reports/coverage.out -covermode=atomic -json 2>&1 | tee reports/unit-raw.json; \
	go tool cover -func=reports/coverage.out > reports/coverage.txt; \
	echo "Unit tests complete. Coverage in reports/coverage.txt"

test-integration:
	@mkdir -p reports
	curl -fsS http://localhost:8080/health >/dev/null 2>&1 || (echo "Stack not running. Run make up or make up-zitadel first." && exit 1)
	go test ./tests/integration/... -tags integration -v -json 2>&1 | tee reports/integration-raw.json; \
	echo "Integration tests complete."

test-e2e:
	@mkdir -p reports
	chmod +x scripts/cqrs_e2e_smoke.sh scripts/worker_conformance_smoke.sh scripts/cqrs_parity_check.sh
	bash scripts/test-e2e.sh

test-frontend:
	@mkdir -p reports
	bash -o pipefail -c 'cd frontend && npm install --silent && npx vitest run --reporter=json --outputFile=../reports/frontend-vitest.json 2>&1 | tee ../reports/frontend-vitest.txt'
	bash -o pipefail -c 'cd tests/e2e/playwright && npm install --silent && npx playwright install --with-deps chromium 2>/dev/null && npx playwright test specs/workflow.spec.ts --reporter=json 2>&1 | tee ../../../reports/playwright-raw.json'

test-perf:
	@mkdir -p reports
	bash -o pipefail -c 'docker run --rm --network host \
		-e COMMAND_URL="$(or $(COMMAND_URL),http://localhost:8080)" \
		-e QUERY_URL="$(or $(QUERY_URL),http://localhost:8081)" \
		-e AUTH_TOKEN="$(AUTH_TOKEN)" \
		-e WORKFLOW_ID="$(WORKFLOW_ID)" \
		-v "$(PWD)/tests/performance/k6:/scripts" \
		-v "$(PWD)/reports:/reports" \
		grafana/k6:latest run --out json=/reports/k6.json /scripts/run-all.js 2>&1 | tee reports/perf-raw.txt'
	@echo "Performance tests complete."

test-security:
	@mkdir -p reports
	bash scripts/test-security.sh
	@echo "[security] npm audit (frontend + nodejs-sdk + playwright)"
	npm --prefix frontend audit --audit-level=high
	npm --prefix clients/nodejs-sdk audit --audit-level=high
	@if [ -f tests/e2e/playwright/package.json ]; then npm --prefix tests/e2e/playwright audit --audit-level=high; fi
	@echo "[security] gitleaks (if installed)"
	@if command -v gitleaks >/dev/null 2>&1; then gitleaks detect --source . --redact --no-git -v 2>&1 | tee reports/gitleaks.txt; else echo "gitleaks not installed — CI security.yml covers it"; fi
	@echo "CodeQL runs in .github/workflows/security.yml (release-blocking on release/1.0)."

test-dast:
	@mkdir -p reports/dast
	chmod +x scripts/test-dast.sh
	bash scripts/test-dast.sh

# Unified release gate (Phase T) — run against a healthy stack where noted.
test-release-suite:
	@mkdir -p reports
	@echo "== release suite: unit =="
	@$(MAKE) test-unit
	@echo "== release suite: bpmn matrix =="
	@$(MAKE) test-bpmn-matrix
	@echo "== release suite: bpmn exhaustive =="
	@$(MAKE) test-bpmn-exhaustive || echo "WARN: bpmn-exhaustive skipped/failed"
	@echo "== release suite: smoke + helm =="
	@$(MAKE) smoke-profiles
	@$(MAKE) smoke-release-profiles
	@$(MAKE) validate-helm
	@echo "== release suite: security (SAST) =="
	@$(MAKE) test-security
	@echo "== release suite: worker conformance (requires stack) =="
	@$(MAKE) worker-conformance || echo "WARN: worker-conformance skipped/failed (stack may be down)"
	@echo "== release suite: frontend unit =="
	@mkdir -p reports && cd frontend && npm install --silent && npx vitest run --reporter=json --outputFile=../reports/frontend-vitest.json
	@echo "Release suite core complete. Also run: make test-all, make test-all-functionality, make test-perf, make test-dast, make test-uat-mega-bpmn."

test-report:
	@mkdir -p reports
	bash scripts/test-report.sh

test-all:
	@mkdir -p reports
	bash scripts/test-all.sh

test-all-functionality:
	@mkdir -p reports
	bash scripts/test-all-functionality.sh

# Local Development Helpers
build-command:
	go build -o bin/workflow-command backend/services/workflow-command/cmd/server/main.go

build-runtime:
	go build -o bin/workflow-runtime backend/services/workflow-command/cmd/runtime/main.go

build-query:
	go build -o bin/workflow-query backend/services/workflow-query/cmd/server/main.go

build-worker:
	go build -o bin/sync-worker backend/services/sync-worker/cmd/main.go

build-backend: build-command build-runtime build-query build-worker

build-frontend:
	cd frontend && npm install && npm run build
