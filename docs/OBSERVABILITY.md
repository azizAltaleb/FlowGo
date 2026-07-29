# Observability

## Metrics

Command/runtime expose Prometheus metrics (default runtime `:9091/metrics`) and JSON engine metrics for admins:

```bash
curl -H "Authorization: Bearer $TOKEN" http://localhost:9100/api/internal/metrics
```

Key series / fields:

| Signal | Source | Use |
| :--- | :--- | :--- |
| `artificialflow_outbox_pending` | Prometheus | Outbox backlog |
| `artificialflow_outbox_publish_success_total` | Prometheus | Publish success |
| `outboxPending` / `outboxPublishLagSec` | `/internal/metrics` | UI banner + ops |
| Sync-worker `/health` | HTTP | Projection consumer liveness |

## CQRS lag SLOs

See [CAPACITY.md](CAPACITY.md) and [UPGRADE.md](UPGRADE.md). Frontend samples projection freshness after query-backed list fetches and shows it in the header banner.

## Suggested Grafana panels

1. Outbox pending gauge
2. Outbox publish success/failure rate
3. Sync-worker consumer lag (Kafka)
4. Command/query p95 latency
5. Runtime lease renewals / tick errors (logs)
