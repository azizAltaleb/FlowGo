# Changelog

All notable changes to FlowGo will be documented in this file.

The format is based on Keep a Changelog, and this project follows Semantic Versioning while it remains pre-1.0.

## [Unreleased]

### Added

- Public launch readiness documentation and governance files.
- GitHub CI, security, Docker image release, and npm package release guidance.
- Helm chart documentation for external IAM and bundled ZITADEL deployment modes.
- Compatibility matrix, Docker image reference, community launch checklist, and release dry-run validation.

### Security

- Repository ignore rules hardened for local secrets, generated tokens, binaries, reports, and dependency folders.
- Release workflows now default manual dispatches to validation/dry-run behavior unless maintainers explicitly enable publishing.

## [0.1.1] - Planned

### Changed

- Renamed project branding, package scope, Docker namespace, Helm chart, and repository metadata to FlowGo.

### Added

- First FlowGo-branded release target for source, Docker images, Helm chart, and Node.js SDK.

## [0.1.0] - 2026-05-12

### Added

- Initial public release target for source, Docker images, Helm chart, and Node.js SDK.
