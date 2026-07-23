# Persistent data rename migration

This transition changes only branded persistent identifiers. Database/table names,
service component names, and `workflow.events.v1` remain unchanged.

| Area | Canonical value | Transition compatibility |
| --- | --- | --- |
| User-task job type | `artificialflow:userTask` | reads/claims/completion/reporting also accept `flowgo:userTask` |
| Elasticsearch prefix | `artificialflow` | query merges canonical and `flowgo` reads, preferring canonical duplicate keys |
| Debezium connector | `artificialflow-postgres-connector` | explicit `CONNECTOR_NAME` override |
| Debezium topic prefix | `artificialflow` | explicit `KAFKA_TOPICS` and connector-config override |
| PostgreSQL slot | `artificialflow_slot` | explicit connector-config override |
| Connect group/topics | `artificialflow-debezium`, `artificialflow_connect_*` | Compose environment overrides |
| Sync consumer group | `artificialflow-sync-worker-v8` (Compose) | explicit `KAFKA_GROUP_ID` override |
| Elasticsearch writer | `artificialflow-*` | explicit `ES_INDEX_PREFIX=flowgo` override |
| Application DB/image user | `artificialflow` | explicit role/DSN and Docker build-arg overrides |
| Application state root | `/artificialflow` | explicit path, file-prefix, and existing-volume/PVC overrides |
| Outbox metrics | `artificialflow_*` | `flowgo_*` mirrors expose the same snapshot for one release |

All migration scripts default to dry-run. No legacy index, database role, state
directory, volume, PVC, connector capture, or snapshot is deleted by these
procedures.

## 1. Preflight and rollback checkpoint

1. Record deployed image digests, configuration, replica counts, and volume/PVC
   names.
2. Determine the current application UID/GID before copying state:

   ```bash
   docker run --rm --entrypoint sh CURRENT_COMMAND_IMAGE -c 'id -u; id -g'
   kubectl exec deploy/CURRENT_COMMAND -- id
   ```

3. Back up PostgreSQL and verify restore credentials. Record role attributes,
   memberships, ownership, grants, and replication privileges:

   ```sql
   SELECT rolname, oid, rolcanlogin, rolreplication FROM pg_roles
   WHERE rolname IN ('flowgo', 'user', 'artificialflow');
   \du+
   ```

4. Run the streaming read-only capture before stopping the old Connect cluster:

   ```bash
   STATE_DIR=streaming-cutover-state \
     scripts/migrate_streaming_identifiers.sh --capture
   ```

5. Run every migration script with no arguments and review its dry-run output.

Rollback checkpoint: keep the database backup, Elasticsearch snapshot name,
captured connector files, old group offsets, old slot LSN, and original
volume/PVC names together.

## 2. Quiesce and drain

Stop workflow writes and runtime progression. Scale the sync worker to zero only
after the old consumer group reports zero lag. Pause the old Debezium connector
and verify the connector and every task are not `RUNNING`.

Do not subscribe one worker to both `flowgo.public.*` and
`artificialflow.public.*`. The worker rejects that mixed configuration by
default because both topic families represent the same source tables. Do not
start the canonical connector while the legacy connector is running.

Rollback checkpoint: if lag cannot reach zero, resume the old deployment without
changing identifiers and investigate before retrying.

## 3. Elasticsearch prefix

Configure a snapshot repository first. Then run:

```bash
scripts/migrate_es_prefix.sh

WORKER_DRAINED=true \
SNAPSHOT_REPOSITORY=production-backups \
INDEX_VERSION=v1 \
  scripts/migrate_es_prefix.sh \
    --apply --confirm migrate-artificialflow-es
```

The script snapshots known `flowgo-*` indices, creates
`artificialflow-<table>-v1`, reindexes each known source, compares counts and
complete sorted ID digests, then atomically assigns the
`artificialflow-<table>` write alias. It refuses to replace a concrete canonical
index and retains all legacy indices.

During the transition, list queries merge and globally sort both index families
so creating a canonical index cannot hide old records. Exact de-duplication is
bounded to 10,000 documents per index (the Elasticsearch default result window),
and pagination is limited to the first 10,000 merged documents. Queries fail
instead of returning an incomplete page when either limit is exceeded.

Rollback: stop the canonical writer/query deployment and restore
`ES_INDEX_PREFIX=flowgo`. The old indices were not modified or deleted.

## 4. Kafka Connect, Debezium, and consumer offsets

After capture and quiescence, stop Kafka Connect. Restart it with:

```text
GROUP_ID=artificialflow-debezium
CONFIG_STORAGE_TOPIC=artificialflow_connect_configs
OFFSET_STORAGE_TOPIC=artificialflow_connect_offsets
STATUS_STORAGE_TOPIC=artificialflow_connect_statuses
```

Verify those internal topics belong to the new, stopped cutover and that the old
connector is not present/running in the restarted cluster. The apply step
copies offsets for unchanged topics such as `workflow.events.v1`; renamed
Debezium topics begin at the new connector cutover.

```bash
QUIESCED=true \
WORKER_DRAINED=true \
CONNECT_INTERNALS_MIGRATED=true \
STATE_DIR=streaming-cutover-state \
  scripts/migrate_streaming_identifiers.sh \
    --apply --confirm migrate-artificialflow-streaming
```

The registered migration connector uses `snapshot.mode=no_data`,
`artificialflow_slot`, and topic prefix `artificialflow`. Keep the sync worker at
zero until connector status, replication slot LSN, canonical topics, and the
new consumer-group offsets are verified. Then start exactly one canonical
sync-worker deployment.

Rollback: stop the canonical worker and connector first. Stop Connect, restore
the old Connect group/internal-topic settings, resume the captured old connector,
and start only the old consumer group. Never overlap old/new connector or worker
generations.

## 5. Database role and application state

The migration grants the old role to the new role, mirrors replication
capability, and retains old ownership for rollback. The standard former Compose
deployment used database role `user`; the former Helm chart used `flowgo`.
Inspect `pg_roles`, object ownership, grants, the application DSN, and the
Debezium connector on the live installation before choosing `OLD_DB_ROLE`—do
not infer or guess it from resource names.

Use the verified live role and numeric target UID/GID. This example is the
standard former Compose case:

```bash
scripts/migrate_database_and_state.sh

ADMIN_PG_DSN='postgresql://admin:...@db/workflow_db' \
OLD_DB_ROLE=user \
NEW_DB_PASSWORD='...' \
STATE_SOURCE=/mounted/legacy-state \
STATE_DESTINATION=/mounted/artificialflow-state \
STATE_UID=10001 \
STATE_GID=10001 \
  scripts/migrate_database_and_state.sh \
    --apply --confirm migrate-artificialflow-state
```

For a former Helm installation whose live inspection confirms the chart
default, run the same command with `OLD_DB_ROLE=flowgo`. If inspection reports a
different role, use that exact role instead.

Review explicit schema/table/sequence grants if `NOINHERIT` or security policy
prevents role membership. Verify Debezium retains `LOGIN`, `REPLICATION`, table
`SELECT`, and publication access before cutover.

For Docker compatibility with existing state, set:

```text
ARTIFICIALFLOW_COMPOSE_PROJECT_NAME=flowgo
ARTIFICIALFLOW_PGDATA_VOLUME=flowgo_pgdata
ARTIFICIALFLOW_ESDATA_VOLUME=flowgo_esdata
ARTIFICIALFLOW_ZITADEL_PGDATA_VOLUME=flowgo_zitadel-postgres-data
ARTIFICIALFLOW_ZITADEL_BOOTSTRAP_VOLUME=flowgo_zitadel-bootstrap
POSTGRES_USER=user
ES_INDEX_PREFIX=flowgo
CONNECT_GROUP_ID=flowgo-debezium
CONNECT_CONFIG_STORAGE_TOPIC=flowgo_connect_configs
CONNECT_OFFSET_STORAGE_TOPIC=flowgo_connect_offsets
CONNECT_STATUS_STORAGE_TOPIC=flowgo_connect_statuses
CONNECTOR_NAME=flowgo-postgres-connector
CONNECTOR_TOPIC_PREFIX=flowgo
CONNECTOR_SLOT_NAME=flowgo_slot
CONNECTOR_DATABASE_USER=user
KAFKA_GROUP_ID=flowgo-sync-worker-v8
KAFKA_TOPICS=workflow.events.v1,flowgo.public.process,...
ARTIFICIALFLOW_STATE_ROOT=/flowgo
ARTIFICIALFLOW_STATE_FILE_PREFIX=flowgo
ARTIFICIALFLOW_BOOTSTRAP_VOLUME=flowgo_flowgo-zitadel-bootstrap
ARTIFICIALFLOW_AUTH_VOLUME=flowgo_flowgo-zitadel-auth
ARTIFICIALFLOW_ZITADEL_NETWORK=zitadel
ARTIFICIALFLOW_API_CREDENTIAL_UID=<existing numeric UID>
```

These are examples for the standard old project. Use the exact names reported
for the installation; do not create guessed replacement volumes.

For Helm upgrades that still use legacy state, start with
`charts/artificialflow/values-legacy-persistent-identifiers.yaml`, correct the
three legacy resource-identity overrides, verify its numeric UID, and include it
explicitly with `-f`. Leave its existing-claim values empty for an in-place
upgrade so Helm renders and protects the same old PVC objects. Set exact existing
claims only when attaching storage managed outside that release.

That compatibility file also keeps the stock `flowgo.example.com` application
origin and TLS secret, `iam.flowgo.example.com` ZITADEL issuer/ingress and TLS
secret, CORS origin, bootstrap frontend URL, and
`/flowgo/bootstrap/flowgo-frontend-client-id`. Keep those values unchanged for
the compatibility release. Reusing the saved ZITADEL application does not by
itself authorize a new canonical browser URL. Before changing DNS, ingress,
issuer, frontend authority, or CORS, explicitly add and verify both old and new
redirect URIs, post-logout redirect URIs, and additional origins on the existing
ZITADEL application. If that application update is not part of the approved
migration, the old URL must remain canonical. External-IAM installations must
retain the issuer, authority, and client identity recorded from their live
provider rather than adopting the bundled-ZITADEL example values.

Fresh backend images use user/group `artificialflow` with UID/GID `10001`.
Compatibility rebuilds can pass `APP_USER`, `APP_UID`, and `APP_GID` build args;
UID/GID `100` is valid only when the deployed image was explicitly rebuilt with
`APP_UID=100` and `APP_GID=100`. Always inspect the deployed image and use the
numeric values it reports.

Rollback: restore the old DSN/role and old path/volume/PVC overrides. The old
database role and source state remain intact.

## 6. Final verification

Before removing maintenance mode:

- query known legacy and newly created user-task records; claim and complete one
  of each type;
- compare PostgreSQL and canonical Elasticsearch counts/IDs;
- confirm only canonical Debezium topics advance;
- confirm only one sync consumer generation is active and lag converges;
- compare `artificialflow_outbox_*` and `flowgo_outbox_*` metric values;
- test database writes, replication, state-file ownership, and restart recovery.

Retain all rollback checkpoints for at least the transition release. Removal of
legacy job reads, indices, metrics, roles, slots, topics, groups, paths, volumes,
or PVCs is a separate destructive change and requires a new approved procedure.
