# High Availability Reference

This document describes the supported ArtificialFlow HA topology for production Kubernetes deployments.

## Supported replica topology (1.0)

| Component | Recommended replicas | Notes |
| :--- | :--- | :--- |
| Gateway | ≥ 2 | Stateless NGINX; safe to scale. |
| Frontend | ≥ 2 | Stateless; safe to scale. |
| Command API | ≥ 2 | Stateless HTTP/gRPC; Postgres is the consistency boundary. |
| Query API | ≥ 2 | Stateless reads from Elasticsearch/OpenSearch. |
| Sync worker | ≥ 1 (Kafka consumer group) | Scale with partition count; use a stable consumer group id. |
| Runtime | ≥ 1 (single-active scheduler) | Timer/SLA/outbox cycles use a Postgres lease so only one active scheduler runs work. Helm default is `replicaCount.runtime: 1`; you may run standby replicas. |

External dependencies (Postgres, Kafka or NATS, Elasticsearch/OpenSearch, Debezium Connect for Kafka CDC) must themselves be HA according to your platform standards.

## Runtime single-active lease

The runtime process acquires a row lease in `runtime_scheduler_lease` before each tick (`TryAcquireRuntimeSchedulerLease`).

- Holder identity: `RUNTIME_LEASE_HOLDER` (default `hostname-pid`).
- TTL: `RUNTIME_LEASE_TTL` (default `15s`, at least `2 × RUNTIME_TICK_INTERVAL`).
- A replica renews its own lease; another replica steals only after expiry.

Outbox publish remains claim-based per message (`ClaimOutboxMessage`). The scheduler lease prevents duplicate timer/SLA side effects across runtime pods.

### Failover verification

Unit coverage: `TestRuntimeSchedulerLeaseFailover` in the persistence package.

Compose drill (no Kubernetes required):

```bash
# Stack must already be up (make up-zitadel or make up-zitadel-release)
./scripts/runtime-lease-failover-compose.sh
# Optional: USE_RELEASE=1 RUNTIME_SERVICE=runtime ./scripts/runtime-lease-failover-compose.sh
```

The script restarts the runtime container and checks it resumes within `RUNTIME_LEASE_TTL + RUNTIME_TICK_INTERVAL` (+ buffer). Record measured RTO in [CAPACITY.md](CAPACITY.md).

Manual Kubernetes check:

1. Scale runtime to 2 standby-capable pods (`replicaCount.runtime=2`).
2. Confirm only one pod logs successful scheduler ticks / holds the lease.
3. Delete the active runtime pod.
4. Within `RUNTIME_LEASE_TTL + RUNTIME_TICK_INTERVAL`, the standby acquires the lease and resumes timer/SLA/outbox cycles.

Helm ships PodDisruptionBudgets for command/query/frontend/gateway when replicas ≥ 2. Do not PDB-protect a single runtime replica as if it were multi-active.

## Failure modes

| Failure | Expected behavior |
| :--- | :--- |
| Command pod loss | In-flight requests fail; clients retry with `Idempotency-Key`. |
| Runtime pod loss | Standby acquires lease after TTL; timers/SLAs/outbox resume. |
| Sync worker loss | Projection lag grows until consumers catch up; command writes remain durable. |
| Postgres unavailable | Writes and runtime ticks fail; restore Postgres first. |
| Elasticsearch unavailable | Query/UI degraded; command path continues. |
| Kafka/Debezium unavailable | Projection lag / outbox backlog; follow [RUNBOOK_CQRS_SYNC.md](RUNBOOK_CQRS_SYNC.md) and [RUNBOOK_OUTBOX_IDEMPOTENCY.md](RUNBOOK_OUTBOX_IDEMPOTENCY.md). |

## Capacity evidence

Performance scenarios live under `tests/performance/k6/`:

- `workflow_throughput` — deploy/start load
- `worker_activate_jobs` — job activation
- `query_read_load` — query API reads
- `cqrs_lag` — write-to-query visibility lag samples

Run:

```bash
make test-perf
```

Publish p50/p99 from the k6 JSON report with each release that claims capacity numbers. See [CAPACITY.md](CAPACITY.md) for the latest published snapshot format.

## Helm guidance

- Keep `replicaCount.runtime` at `1` unless you intentionally run standby runtime pods that share the lease.
- Prefer PDB on command, query, gateway, and frontend.
- Externalize Postgres, Kafka/NATS, and search; do not run production data planes inside the ArtificialFlow chart.
- Configure ingress TLS and secrets per [deployment.md](deployment.md).
