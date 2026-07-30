# Changelog

All notable changes to ArtificialFlow will be documented in this file.

The format is based on Keep a Changelog, and this project follows Semantic Versioning while it remains pre-1.0.

## [1.0.0] - 2026-07-30

1.0.0 GA — freeze tip after partner/self-run bake-off and RC Phase T.

## [1.0.0-rc.1] - 2026-07-29

1.0 freeze candidate (same product tip as 0.4.0 plus RC version pins).

## [Unreleased]

### Added

- BPMN Tier-2 before 1.0 GA: Escalation, Conditional, Cancel/transaction, Event sub-process, Message/Signal start fan-out, Compensation XML fixtures, Manual Task wait semantics.
- Modeler Visual palette (Pool/Lane/Data/Annotation), event markers (link/terminate/escalation/conditional), deploy lint against sequence flows on visual-only shapes.

## [0.4.0] - 2026-07-28

### Added

- Runtime Postgres scheduler lease, HA reference, capacity snapshot, CQRS lag k6 scenario, golden-demo bake-off.
- Browser Task Inbox, instance job ops/retry, CQRS status banner, Go/Python worker samples, external IAM recipes (Keycloak/Entra/Auth0).
- DMN-lite JSON decision tables (API/runtime; Decisions UI behind feature flag), official connectors (HTTP/Kafka/email/webhook/S3), process start versioning.
- Modeler: editable element IDs, labeled palette (no horizontal scroll), inline deploy/BPMN validation errors, Tier-3 visual artifacts.
- Send Task support via external jobs (default `io.artificialflow.connector.send`).
- Terminate end event and link catch/throw; clear parse errors for escalation/conditional/cancel/event sub-process.
- Release test suite (`make test-release-suite`), DAST (`make test-dast`), mega BPMN UAT fixture/spec.
- Helm managed-deps example values; Node SDK worker processing example; gateway BPMN examples.
- Partner bake-off scorecard for design-partner self-hosted evaluation.

### Changed

- BPMN support matrix tiered (Tier-1/2/3) for honest Supported / Not supported claims.
- Compatibility matrix and release pins target `v0.4.x` toward `1.0.0`.

## [0.3.0] - 2026-07-22

### Changed

- Moved the coordinated ArtificialFlow transition release to an unpublished version across npm, container, Compose, and Helm artifacts.
- Hardened release collision checks, ZITADEL migration pagination, and persistent-state migration guidance.

## [0.2.0] - 2026-07-22

### Added

- Production SDK authentication for bundled ZITADEL and external IAM, including JWT Profile exchange, introspection, role validation, credential rotation, and HTTP/gRPC propagation.
- Workflow history, identity-management UI, service-account helpers, and a reusable Playwright UAT harness.
- Exhaustive BPMN, deployment-matrix, CQRS, Compose, Helm, and release validation gates.

### Changed

- Hardened transactional workflow event handling, variable persistence, job idempotency, CQRS projection behavior, and operational feedback.
- Upgraded Go, gRPC, React Router, and Node.js SDK dependencies.
- Required the full CQRS end-to-end smoke test for pull requests.

### Security

- Repository ignore rules hardened for local secrets, generated tokens, binaries, reports, and dependency folders.
- Addressed high-severity Go and npm dependency findings and expanded secret, static-analysis, vulnerability, image, and CodeQL gates.

## [0.1.1] - 2026-05-17

### Changed

- Renamed project branding, package scope, Docker namespace, Helm chart, and repository metadata to FlowGo.

### Added

- First FlowGo-branded release target for source, Docker images, Helm chart, and Node.js SDK.

## [0.1.0] - 2026-05-12

### Added

- Initial public release target for source, Docker images, Helm chart, and Node.js SDK.
