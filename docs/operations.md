# Operations Guide

## Health Checks

- Command service: `/health`.
- Query service: `/health`.
- Sync worker: `/health`.
- Gateway/frontend: `/`.

## Metrics and Internal Signals

Command service exposes compact health metrics and internal counters for:

- Outbox relay activity.
- Idempotency records.
- Publish lag.
- Retry exhaustion.

Sync worker exposes freshness and projection health through its health endpoint.

## Backups

Production deployments should back up:

- Postgres.
- Search indexes or source-of-truth data required to rebuild them.
- IAM configuration and secrets.
- Helm values and release manifests.

## Recovery

- Rebuild search projections from source events or CDC topics when possible.
- Use `make init-connector` only as a manual recovery path for local/development profiles.
- Keep Debezium connector configuration versioned and reviewed.

## Production Checklist

- TLS enabled.
- Default credentials replaced.
- Strict OIDC audience validation enabled.
- Secrets stored outside source control.
- Resource requests and limits configured.
- Backup and restore tested.
- Monitoring and alerts configured.
