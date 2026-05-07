# Architecture

GoFlow is an open-source BPMN workflow engine with a browser modeler, command API, runtime worker, query API, CQRS sync worker, and Node.js SDK.

## Components

| Component | Responsibility |
| :--- | :--- |
| Frontend | BPMN modeler, dashboards, identity views, SDK client administration, and workflow operations. |
| Gateway | Single HTTP entry point that routes `/` to the frontend, `/api/` to command service, and `/api/query/` to query service. |
| Command service | Workflow deployment, instance mutation APIs, worker job APIs, IAM-backed identity endpoints, outbox, and runtime coordination. |
| Runtime | Background workflow execution loop for timers, jobs, and pending execution paths. |
| Query service | Read-optimized API backed by Elasticsearch or OpenSearch projections. |
| Sync worker | Projects command-side changes from Kafka/Debezium or event topics into search indexes. |
| IAM provider | External OIDC provider or bundled ZITADEL for local solution-managed IAM. |
| Node.js SDK | Programmatic workflow, worker, identity, and management API client. |

## Request Flow

1. Users open the frontend through the gateway.
2. The frontend authenticates with the configured OIDC provider.
3. Command APIs handle writes and emit events/outbox records.
4. Runtime and workers execute workflow jobs.
5. Sync worker projects state changes into search indexes.
6. Query APIs serve dashboard, process, and instance views.

## Deployment Modes

- **Docker Compose external IAM**: local/evaluation stack connected to an existing OIDC provider.
- **Docker Compose bundled ZITADEL**: local/evaluation stack with solution-managed ZITADEL.
- **Helm external IAM**: production Kubernetes deployment using externally managed infrastructure and OIDC.
- **Helm bundled ZITADEL**: Kubernetes deployment with bundled ZITADEL for environments that require solution-managed IAM.

## Production Boundaries

Production deployments should externalize Postgres, Kafka or NATS, Elasticsearch or OpenSearch, secrets, TLS, backups, and observability.
