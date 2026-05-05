.PHONY: up up-external-iam up-zitadel up-external-iam-release up-zitadel-release down down-external-iam down-zitadel restart logs logs-external-iam logs-zitadel ps ps-external-iam ps-zitadel clean clean-external-iam clean-zitadel demo build-backend build-frontend up-core up-full up-full-cqrs smoke-core smoke-full smoke-release-core smoke-release-full smoke-release-profiles smoke-profiles validate-helm cqrs-parity-check cqrs-e2e-smoke worker-conformance test-bpmn-matrix test-unit test-integration test-e2e test-frontend test-perf test-security test-report test-all


# Docker Compose Commands
up:
	@$(MAKE) up-external-iam

up-external-iam:
	docker compose -f docker-compose.external-iam.yml up -d --build

up-zitadel:
	docker compose -f docker-compose.zitadel.yml up -d --build

up-external-iam-release:
	docker compose -f docker-compose.external-iam.yml -f docker-compose.release.yml up -d

up-zitadel-release:
	docker compose -f docker-compose.zitadel.yml -f docker-compose.release.yml up -d

up-core:
	@$(MAKE) up-external-iam

up-full:
	@$(MAKE) up-external-iam

up-full-cqrs:
	@$(MAKE) up-external-iam

smoke-core:
	docker compose -f docker-compose.external-iam.yml config > /dev/null

smoke-full:
	docker compose -f docker-compose.zitadel.yml config > /dev/null

smoke-profiles: smoke-core smoke-full

smoke-release-core:
	docker compose -f docker-compose.external-iam.yml -f docker-compose.release.yml config > /dev/null

smoke-release-full:
	docker compose -f docker-compose.zitadel.yml -f docker-compose.release.yml config > /dev/null

smoke-release-profiles: smoke-release-core smoke-release-full

validate-helm:
	bash scripts/validate_helm.sh

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
	@$(MAKE) logs-external-iam

logs-external-iam:
	docker compose -f docker-compose.external-iam.yml logs -f

logs-zitadel:
	docker compose -f docker-compose.zitadel.yml logs -f

ps:
	@$(MAKE) ps-external-iam

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
	go test ./backend/services/workflow-command/internal/domain/bpmn -run 'TestParse_(ElementTypeMatrix|MapsExtendedElementsAndProperties|MapsPlainAttributeAliasesForCompatibility|MergesExtensionPropertiesWithoutOverridingMappedKeys|CanonicalizesExtensionPropertyAliases|BoundaryCancelActivityAttributeTakesPrecedenceOverExtensionAlias|PopulatesIncomingForGatewayJoin|FailsForUnsupportedElementReferences|FailsForUnsupportedSendTaskReferences)' -count=1
	go test ./backend/services/workflow-command/tests -run 'TestDeployWorkflowFromBPMN_(CallActivityBusinessRuleAndManualTask|EventBasedGatewayReceiveAndTimer|BoundaryTimerInterruptsTask|ThrowSignalTriggersCatch|ThrowMessageUsesCorrelationKey|ThrowMessageUsesPlainCorrelationKeyAlias|BoundaryTimerNonInterruptingKeepsTaskActive|BoundaryCancelActivityExtensionAliasKeepsTaskActive|BoundaryCancelActivityAttributeTakesPrecedenceOverExtensionAlias|ServiceTaskPlainTaskTypeAliasMapsImplementation|ServiceTaskExtensionPropertyMapsImplementation|UserTaskAssignmentFromExtensionProperties|FailsForUnsupportedElementReferences|FailsForUnsupportedSendTaskReferences)' -count=1

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
	cd frontend && npm install --silent && npx vitest run --reporter=json --outputFile=../reports/frontend-vitest.json 2>&1 | tee ../reports/frontend-vitest.txt
	cd tests/e2e/playwright && npm install --silent && npx playwright install --with-deps chromium 2>/dev/null && npx playwright test --reporter=json 2>&1 | tee ../../../reports/playwright-raw.json || true

test-perf:
	@mkdir -p reports
	docker run --rm --network host \
	  -v "$(PWD)/tests/performance/k6:/scripts" \
	  -v "$(PWD)/reports:/reports" \
	  grafana/k6:latest run --out json=/reports/k6.json /scripts/run-all.js 2>&1 | tee reports/perf-raw.txt; \
	echo "Performance tests complete."

test-security:
	@mkdir -p reports
	bash scripts/test-security.sh

test-report:
	@mkdir -p reports
	bash scripts/test-report.sh

test-all:
	@mkdir -p reports
	bash scripts/test-all.sh

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
