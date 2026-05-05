# CQRS Sync Runbook

## Signals

Review:

- Sync worker `/health`.
- Query service `/health`.
- Kafka consumer group assignments.
- Debezium connector status.
- Elasticsearch/OpenSearch index document counts.

## Checks

```bash
make cqrs-parity-check
make cqrs-e2e-smoke
```

## Common Actions

- Confirm `KAFKA_TOPICS` matches the selected `SYNC_PROJECTION_CONTRACT`.
- Confirm Debezium connector registration when using CDC topics.
- Confirm search backend address and credentials.
- Re-run connector initialization only as a manual recovery path when automatic bootstrap is unavailable.
