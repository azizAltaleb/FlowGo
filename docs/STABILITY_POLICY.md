# Stability Policy

Workflowsa is pre-1.0 software.

## Versioning

- Public releases use Semantic Versioning.
- Before `1.0.0`, minor versions may include breaking changes.
- Breaking changes should be documented in release notes and the changelog.

## Compatibility Surfaces

Compatibility-sensitive surfaces include:

- Worker REST API.
- Node.js SDK public API.
- Docker image environment variables.
- Helm values.
- IAM role names and claim mapping behavior.
- BPMN parser/runtime semantics.

## Deprecation

Where possible, deprecations should include:

- Replacement behavior.
- Migration notes.
- At least one release cycle of overlap.
