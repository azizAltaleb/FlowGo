# Compatibility Matrix

ArtificialFlow publishes compatibility guarantees for the surfaces below. The
stable `1.x` contract policy in [STABILITY_POLICY.md](STABILITY_POLICY.md)
applies to `v1.1.0`.

## Versioning Policy

| Surface | Current line | Compatibility policy |
| :--- | :--- | :--- |
| Platform release | `v1.1.x` | Patch releases are backward compatible unless a security fix requires otherwise. Breaking compatibility changes require a major bump and deprecation policy. |
| Command API | `v1` routes and generated protobuf API `artificialflow.api.v1` | Additive changes preferred. Breaking changes require release notes and deprecation. |
| Query API | `v1` HTTP responses | Response fields may be added. Existing field meaning should remain stable. |
| Worker API | Protocol headers and `/jobs/*` endpoints | Wire compatibility protected by `make worker-conformance`. |
| Inbox API | `/inbox*` endpoints | Acting-user header contract is part of the public surface for SDK and browser inbox. |
| Node.js SDK | `@artificialflow/nodejs-sdk@1.1.x` | Patch releases preserve public method signatures; deprecations follow the 1.x policy. |
| Go worker library | `backend/libs/worker` | Public `Worker` / handler APIs are compatibility-sensitive under the 1.x policy. |
| Docker images | `artificialflow/*:v1.1.x` | Image environment variables and exposed ports stay stable within a minor line. |
| Helm chart | `charts/artificialflow` `1.1.x` | Values may be added. Renames/removals require migration notes. |
| BPMN semantics | See [BPMN_SUPPORT_MATRIX.md](BPMN_SUPPORT_MATRIX.md) | Supported/not-supported/not-evaluated statuses are part of the contract. |

## Developer HTTP API

Language-agnostic REST integration (command, query, worker, inbox, auth) is
documented in [API.md](API.md). That guide is the contract for consumers who do
not use `@artificialflow/nodejs-sdk`.

## Bake-off Supported Contracts

Teams evaluating ArtificialFlow for self-hosted BPMN ops should treat these as the bake-off contracts:

| Contract | ArtificialFlow surface | Notes |
| :--- | :--- | :--- |
| Process deploy / start | Command API + modeler | BPMN 2.0 XML with ArtificialFlow extensions; see external import guide. |
| External workers | `/jobs/*` + Node/Go SDKs | Job activate, complete, fail, extend-lock, capabilities, idempotency. |
| Event publish | `/messages`, `/signals`, `/escalations` + Node SDK | `publishMessage`, `publishSignal`, `publishEscalation` for catch/start/event-sub-process correlation. |
| Human tasks | `/inbox*` + instance task APIs | Claim/complete with acting-user identity; browser inbox in the frontend. Due-date breach publishes escalation `user-task.due-date.breached` and creates an SLA incident. |
| Incidents | `GET /incidents` + Incidents UI | Engine/SLA incidents (`SLA_DUE_DATE_BREACHED`, job failures, etc.). |
| Ops history / search | Query API + Elasticsearch/OpenSearch | Eventually consistent CQRS projection. |
| Identity | External OIDC or bundled ZITADEL | Roles: `artificialflow admin`, `artificialflow modeler`, `artificialflow client`. |
| Kubernetes install | Helm chart | External Postgres, Kafka/NATS, Elasticsearch/OpenSearch, Debezium for Kafka projection. |

See also [EXTERNAL_BPMN_IMPORT.md](EXTERNAL_BPMN_IMPORT.md) and [HA_REFERENCE.md](HA_REFERENCE.md).

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
