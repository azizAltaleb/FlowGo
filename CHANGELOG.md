# Changelog

All notable changes to FlowGo will be documented in this file.

The format is based on Keep a Changelog, and this project follows Semantic Versioning while it remains pre-1.0.

## [Unreleased]

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
