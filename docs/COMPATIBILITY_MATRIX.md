# Compatibility Matrix

GoFlow is pre-1.0 software. Compatibility guarantees are conservative until the public API, worker protocol, SDK, image tags, and Helm values are stabilized.

## Versioning Policy

| Surface | Current line | Compatibility policy before 1.0 |
| :--- | :--- | :--- |
| Platform release | `v0.1.x` | Patch releases should be backward compatible unless a security fix requires otherwise. |
| Command API | `v1` routes and generated protobuf API | Additive changes are preferred. Breaking changes require release notes. |
| Query API | `v1` HTTP responses | Response fields may be added. Existing field meaning should remain stable. |
| Worker API | Protocol headers and `/jobs/*` endpoints | Wire compatibility is protected by conformance checks. |
| Node.js SDK | `@goflow/nodejs-sdk@0.1.x` | Patch releases should preserve public method signatures where practical. |
| Docker images | `goflow/*:v0.1.x` | Image environment variables and exposed ports should stay stable within a minor line. |
| Helm chart | `charts/goflow` `0.1.x` | Values may be added. Renames/removals require migration notes. |

## Supported Deployment Combinations

| Deployment | IAM mode | Runtime dependencies | Status |
| :--- | :--- | :--- | :--- |
| Docker Compose external IAM | Existing OIDC provider | Postgres, Kafka, Debezium Connect, Elasticsearch | Development/evaluation supported. |
| Docker Compose bundled ZITADEL | Bundled ZITADEL | Postgres, Kafka, Debezium Connect, Elasticsearch | Development/evaluation supported. |
| Docker Compose release override | External IAM or bundled ZITADEL | Published `goflow/*` images | Release smoke validation path. |
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
| `@goflow/nodejs-sdk@0.1.x` | GoFlow `v0.1.x` | Use Node.js 20 or newer. |

The SDK can target local Compose through `GOFLOW_BASE_URL=http://localhost:9100/api` and a token with the `goflow client` role.

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
