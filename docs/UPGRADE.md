# Upgrade Guide (0.x → 1.0)

Operator checklist for upgrading ArtificialFlow with durable state.

## Before you start

1. Read [COMPATIBILITY_MATRIX.md](COMPATIBILITY_MATRIX.md) and release notes for the target version.
2. Confirm external Postgres, Kafka/NATS, Elasticsearch/OpenSearch, and Debezium are healthy.
3. Schedule a maintenance window if you expect projection rebuilds.

## SLO checks (pre- and post-upgrade)

| Signal | How to check | Target |
| :--- | :--- | :--- |
| Outbox pending | Metrics `artificialflow_outbox_pending` / [RUNBOOK_OUTBOX_IDEMPOTENCY.md](RUNBOOK_OUTBOX_IDEMPOTENCY.md) | Drain or stable near zero before cutover |
| CQRS lag | Deploy/start then query visibility; sync-worker logs | p99 under nominal load &lt; 15s |
| Runtime lease | Runtime logs show `lease_holder`; only one active tick worker | Failover &lt; lease TTL + tick |

## Upgrade procedure

1. **Backup Postgres** (command database is source of truth).
2. **Pin image tags/digests** in Helm values or Compose release override.
3. **Review Helm value diffs** for the chart minor/major; apply migration notes (including [PERSISTENT_DATA_RENAME_MIGRATION.md](PERSISTENT_DATA_RENAME_MIGRATION.md) when renaming releases).
4. **Drain outbox** — wait until pending/processing counts are acceptable ([RUNBOOK_OUTBOX_IDEMPOTENCY.md](RUNBOOK_OUTBOX_IDEMPOTENCY.md)).
5. **Helm upgrade** (or Compose pull + recreate application services). Prefer rolling command/query/frontend; runtime may briefly pause ticks during pod replace (lease TTL bounds recovery).
6. **Verify**:
   - `/health` on command and query
   - OIDC login
   - Deploy a test process, start an instance, activate a job, complete a user task via inbox
   - Query projection shows the new instance ([RUNBOOK_CQRS_SYNC.md](RUNBOOK_CQRS_SYNC.md))
7. **If projection is stale or corrupt** — follow CQRS rebuild steps in the CQRS runbook, then re-check lag.

## Operator checklist

- [ ] Postgres backup completed and restore-tested
- [ ] Image tags/digests recorded
- [ ] Outbox drained / lag acceptable
- [ ] Helm/Compose upgrade applied
- [ ] Health endpoints green
- [ ] Worker smoke (activate/complete)
- [ ] User-task smoke (claim/complete)
- [ ] Query shows new instance within SLO
- [ ] Runtime lease holder present in logs

## Rollback

1. Redeploy previous image tags/digests.
2. Do not restore Postgres unless schema migrations are incompatible; if you must restore, restore search indexes or rebuild projections afterward.
3. Re-run the operator checklist.
