# Changelog

All notable changes to ArtificialFlow will be documented in this file.

The format is based on Keep a Changelog, and this project follows Semantic Versioning.

## [Unreleased]

## [1.1.0] - 2026-08-05

### Added

- Expanded executable BPMN coverage for timer schedules, escalation, conditional,
  message/signal start and event sub-process behavior, compensation, transaction
  cancellation, connector input/output mappings, and manual-task waiting.
- Added capability-driven BPMN validation, exhaustive matrix scenarios, and
  completeness regression tests.
- Added richer modeler support for BPMN events, gateways, task properties,
  visual artifacts, diagram history, and connector descriptors.
- Added the language-agnostic HTTP API guide covering command, query, worker,
  inbox, identity, and direct non-SDK integration.
- Added instance variable editing for nested objects and lists, including add,
  edit, and delete controls.
- Added the process-definition version to the instance details header.

### Changed

- Hardened HTTP/webhook connector validation, redirect handling, descriptors,
  and runtime input resolution.
- Updated the Node.js SDK and public API types for the additive `1.1` surfaces.
- Hid the Incidents console page while preserving the admin incident HTTP API.
- Updated bundled-IAM quickstart and Compose documentation for direct API and
  SDK consumers.

### Fixed

- Fixed empty object/list values being impossible to edit from instance details.
- Fixed intermittent browser-session loss after transient OIDC renewal or
  session-monitor errors.
- Fixed frontend route/access handling for hidden Decisions and Incidents pages.
- Fixed BPMN parser/runtime edge cases covered by the `1.1` capability matrix.

### Security

- Revalidate connector redirect destinations against the configured allowlist.
- Updated the frontend nginx runtime base to remove fix-available high-severity
  Alpine package vulnerabilities from the release image.
- Preserve release secret scanning, dependency audits, SAST, DAST, image scans,
  SBOM/provenance, and signing as publication-blocking gates.

## [1.0.0] - 2026-07-30

1.0.0 GA — freeze tip after partner/self-run bake-off and RC Phase T.

## [1.0.0-rc.1] - 2026-07-29

1.0 freeze candidate (same product tip as 0.4.0 plus RC version pins).

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
