# Compatibility Matrix

ArtificialFlow publishes compatibility guarantees for the surfaces below. Pre-`1.0.0` releases remain conservative; the `1.0` contract policy in [STABILITY_POLICY.md](STABILITY_POLICY.md) applies once `1.0.0` is tagged.

## Versioning Policy

| Surface | Current line | Compatibility policy |
| :--- | :--- | :--- |
| Platform release | `v0.4.x` (toward `1.0.0`) | Patch releases should be backward compatible unless a security fix requires otherwise. From `1.0.0`, breaking changes require a major bump. |
| Command API | `v1` routes and generated protobuf API `artificialflow.api.v1` | Additive changes preferred. Breaking changes require release notes and, after `1.0.0`, deprecation. |
| Query API | `v1` HTTP responses | Response fields may be added. Existing field meaning should remain stable. |
| Worker API | Protocol headers and `/jobs/*` endpoints | Wire compatibility protected by `make worker-conformance`. |
| Inbox API | `/inbox*` endpoints | Acting-user header contract is part of the public surface for SDK and browser inbox. |
| Node.js SDK | `@artificialflow/nodejs-sdk@0.2.x` | Patch releases should preserve public method signatures where practical. |
| Go worker library | `backend/libs/worker` | Public `Worker` / handler APIs are compatibility-sensitive toward `1.0.0`. |
| Docker images | `artificialflow/*:v0.4.x` | Image environment variables and exposed ports should stay stable within a minor line. |
| Helm chart | `charts/artificialflow` `0.4.x` | Values may be added. Renames/removals require migration notes. |
| BPMN semantics | See [BPMN_SUPPORT_MATRIX.md](BPMN_SUPPORT_MATRIX.md) | Supported/not-supported/not-evaluated statuses are part of the contract. |

## Camunda-Facing Supported Contracts

Teams evaluating ArtificialFlow against Camunda 8 self-managed should treat these as the bake-off contracts:

| Contract | ArtificialFlow surface | Notes |
| :--- | :--- | :--- |
| Process deploy / start | Command API + modeler | BPMN 2.0 XML with ArtificialFlow extensions; see Camunda import guide. |
| External workers | `/jobs/*` + Node/Go SDKs | Job activate, complete, fail, extend-lock, capabilities, idempotency. |
| Human tasks | `/inbox*` + instance task APIs | Claim/complete with acting-user identity; browser inbox in the frontend. |
| Ops history / search | Query API + Elasticsearch/OpenSearch | Eventually consistent CQRS projection. |
| Identity | External OIDC or bundled ZITADEL | Roles: `artificialflow admin`, `artificialflow modeler`, `artificialflow client`. |
| Kubernetes install | Helm chart | External Postgres, Kafka/NATS, Elasticsearch/OpenSearch, Debezium for Kafka projection. |

See also [CAMUNDA_BPMN_IMPORT.md](CAMUNDA_BPMN_IMPORT.md) and [HA_REFERENCE.md](HA_REFERENCE.md).

## Supported Deployment Combinations

| Deployment | IAM mode | Runtime dependencies | Status |
| :--- | :--- | :--- | :--- |
| Docker Compose external IAM | Existing OIDC provider | Postgres, Kafka, Debezium Connect, Elasticsearch | Development/evaluation supported. |
| Docker Compose bundled ZITADEL | Bundled ZITADEL | Postgres, Kafka, Debezium Connect, Elasticsearch | Development/evaluation supported. |
| Docker Compose release override | External IAM or bundled ZITADEL | Published `artificialflow/*` images | Release smoke validation path. |
| Helm external IAM | Existing OIDC provider | Managed or chart-provided dependencies | Production-oriented path. |
| Helm bundled ZITADEL | Bundled ZITADEL | Managed or chart-provided dependencies | Production-oriented path for solution-managed IAM. |

## Worker Protocol

| Item | Compatibility note |
| :--- | :--- |
| Request version header | Workers may send `X-Workflow-Worker-Protocol-Version`. |
| Response version header | The API returns `X-Workflow-Engine-Protocol-Version`. |
| Idempotency | Mutation retries should use `Idempotency-Key`. |
| Capabilities | `GET /jobs/capabilities` exposes server-supported worker features. |
| Conformance | Run `make worker-conformance` before changing worker API behavior. |

## SDK Compatibility

| SDK package | Compatible platform target | Notes |
| :--- | :--- | :--- |
| `@artificialflow/nodejs-sdk@0.2.x` | ArtificialFlow `v0.2.x` / `v0.3.x` | Use Node.js 20 or newer. Includes inbox helpers. |
| `@artificialflow/nodejs-sdk@0.1.x` | ArtificialFlow `v0.1.x` | Historical package and product line. |

The SDK can target local Compose through `ARTIFICIALFLOW_BASE_URL=http://localhost:9100/api`. New bundled-ZITADEL clients use a service-account JSON profile and automatic JWT Profile exchange. The raw `token` option remains compatible with external IAM and staged legacy PAT migration.

Key revocation prevents new exchanges but does not retroactively invalidate already-minted short-lived access tokens. Existing bundled PATs remain accepted and revocable during the migration window; new PAT creation and rotation are disabled by default.

## Release Validation

Before publishing a public release, run:

```bash
make release-dry-run
make smoke-release-profiles
```

For worker API changes, also run:

```bash
make worker-conformance
```
