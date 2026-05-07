# GoFlow Helm Chart

This chart deploys GoFlow for production Kubernetes environments.

Default GoFlow image repositories point to Docker Hub under `gofl0w/*` and use a pinned release tag. Pin image tags explicitly for production rollouts.

## Deployment models

- External IAM: use an existing OIDC provider and `values-external-iam.yaml`.
- Internal IAM: deploy bundled ZITADEL and use `values-internal-iam.yaml`.

## Production dependencies

The chart expects production Postgres, Kafka or NATS, Elasticsearch or OpenSearch, and Debezium Connect endpoints to be provided through values. It does not bundle those dependencies by default.

## External IAM install

```bash
helm upgrade --install goflow ./charts/goflow \
  --namespace goflow --create-namespace \
  -f ./charts/goflow/values-external-iam.yaml \
  --set images.command.repository=REGISTRY/workflow-command \
  --set images.query.repository=REGISTRY/workflow-query \
  --set images.runtime.repository=REGISTRY/workflow-runtime \
  --set images.syncWorker.repository=REGISTRY/sync-worker \
  --set images.frontend.repository=REGISTRY/frontend \
  --set postgresql.existingSecret=goflow-postgres \
  --set iam.auth.issuerPublicUrl=https://login.example.com \
  --set iam.auth.issuerInternalUrl=https://login.example.com \
  --set iam.frontend.oidcAuthority=https://login.example.com \
  --set iam.frontend.oidcClientId=workflow-frontend
```

## Internal ZITADEL IAM install

```bash
helm upgrade --install goflow ./charts/goflow \
  --namespace goflow --create-namespace \
  -f ./charts/goflow/values-internal-iam.yaml \
  --set images.command.repository=REGISTRY/workflow-command \
  --set images.query.repository=REGISTRY/workflow-query \
  --set images.runtime.repository=REGISTRY/workflow-runtime \
  --set images.syncWorker.repository=REGISTRY/sync-worker \
  --set images.frontend.repository=REGISTRY/frontend \
  --set postgresql.existingSecret=goflow-postgres \
  --set zitadel.masterkey=REPLACE_WITH_32_CHAR_MASTERKEY \
  --set zitadel.bootstrap.adminPassword=REPLACE_WITH_ADMIN_PASSWORD
```

## Required external secrets

If `postgresql.existingSecret` is set, the secret must contain `PG_DSN` by default. Override `postgresql.existingSecretKey` if your secret uses another key.

For external IAM introspection mode, provide `AUTH_INTROSPECTION_CLIENT_SECRET` in the same secret or let the chart create it from `iam.auth.introspectionClientSecret`.

## Routing

The gateway exposes the frontend at `/`, command API at `/api/`, and query API at `/api/query/`. Backend services stay internal through `ClusterIP` services by default.

## Production hardening

- Replace all default values for passwords, master keys, and local development hosts.
- Use TLS-enabled ingress and production OIDC issuer URLs.
- Keep `iam.auth.enforceAudience=true` unless a documented compatibility exception is required.
- Prefer externally managed Postgres, Kafka/NATS, Elasticsearch/OpenSearch, and secret management.
- Use signed GoFlow release images and verify SBOM/provenance artifacts when available.
