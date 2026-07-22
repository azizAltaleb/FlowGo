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

## UI Consistency Contract

- Command API success means a deploy, start, or delete operation was accepted by the write side.
- Workflow and instance lists read from CQRS query projections and are eventually consistent with command writes.
- Single workflow and instance detail reads use command routes and are not subject to list projection lag.
- After command writes that affect query-backed lists, the frontend should show a syncing state and poll the relevant query list until the expected row appears or disappears.
- If frontend polling times out, treat the command as successful but the projection as still syncing; show Retry or Refresh instead of implying failure.
- Dashboard and aggregate views should describe data as a query projection, not as real-time data.
- Frontend polling is user-facing feedback only. Use `make cqrs-parity-check` and `make cqrs-e2e-smoke` for operational validation.

## Common Actions

- Confirm `KAFKA_TOPICS` matches the selected `SYNC_PROJECTION_CONTRACT`.
- Confirm Debezium connector registration when using CDC topics.
- Confirm search backend address and credentials.
- Re-run connector initialization only as a manual recovery path when automatic bootstrap is unavailable.
