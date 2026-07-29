# Stability Policy

ArtificialFlow is approaching a `1.0.0` compatibility freeze for the surfaces listed below.

## Versioning

- Public releases use Semantic Versioning.
- Before `1.0.0`, minor versions may still include breaking changes when documented in release notes.
- From `1.0.0` onward, breaking changes on compatibility surfaces require a deprecation cycle and a major version bump.

## Compatibility Surfaces

Compatibility-sensitive surfaces include:

- Worker REST API (`/jobs/*` protocol headers, capabilities, activate/complete/fail/extend-lock).
- Inbox REST API (`/inbox*`) used by SDK and browser inbox clients.
- Command and Query HTTP API `v1` response field meaning.
- Protobuf package `artificialflow.api.v1`.
- Node.js SDK public API (`@artificialflow/nodejs-sdk`), including inbox helpers.
- Go worker library public API (`backend/libs/worker` / published module path).
- Docker image environment variables and exposed ports.
- Helm chart values under `charts/artificialflow`.
- IAM role names (`artificialflow admin`, `artificialflow modeler`, `artificialflow client`) and claim mapping behavior.
- BPMN parser/runtime semantics documented in `docs/BPMN_SUPPORT_MATRIX.md`.

## 1.0 Contract Policy

Once `1.0.0` is tagged:

- Additive changes are preferred on all compatibility surfaces.
- Removals or meaning changes require: replacement behavior, migration notes, and at least one minor release of deprecation overlap.
- Security fixes may break compatibility when no safe alternative exists; such breaks must be called out in release notes.

## Deprecation

Where possible, deprecations should include:

- Replacement behavior.
- Migration notes.
- At least one release cycle of overlap.
