# ArtificialFlow Helm Chart

This chart deploys ArtificialFlow for production Kubernetes environments.

Default ArtificialFlow image repositories point to Docker Hub under
`artificialflow/*` and use a pinned release tag. For one transition release,
the same manifests are also available under `azizaltaleb/*`. Pin image tags or
digests explicitly for production rollouts.

## Deployment models

- External IAM: use an existing OIDC provider with `iam.mode=external` and `values-external-iam.yaml`.
- Bundled ZITADEL IAM: deploy solution-managed ZITADEL with `iam.mode=zitadel`, `zitadel.enabled=true`, and `values-internal-iam.yaml`.

The chart renders the same ArtificialFlow application services for both modes: command API, runtime, query API, sync worker, frontend, and gateway. Bundled ZITADEL mode adds ZITADEL API/login, a ZITADEL Postgres StatefulSet by default, bootstrap PVCs, and an IAM ingress.

## Production dependencies

The chart expects production Postgres, Kafka or NATS, Elasticsearch or OpenSearch, and Debezium Connect endpoints to be provided through values. It does not bundle those dependencies by default.

Bundled ZITADEL also requires a storage class supporting `ReadWriteMany` for
the three bootstrap/auth PVCs with the default values.

## External IAM install

```bash
helm upgrade --install artificialflow ./charts/artificialflow \
  --namespace artificialflow --create-namespace \
  -f ./charts/artificialflow/values-external-iam.yaml \
  --set images.command.repository=REGISTRY/workflow-command \
  --set images.query.repository=REGISTRY/workflow-query \
  --set images.runtime.repository=REGISTRY/workflow-runtime \
  --set images.syncWorker.repository=REGISTRY/sync-worker \
  --set images.frontend.repository=REGISTRY/frontend \
  --set postgresql.existingSecret=artificialflow-postgres \
  --set iam.auth.issuerPublicUrl=https://login.example.com \
  --set iam.auth.issuerInternalUrl=https://login.example.com \
  --set iam.frontend.oidcAuthority=https://login.example.com \
  --set iam.frontend.oidcClientId=workflow-frontend
```

External IAM must issue `artificialflow admin` for platform administrators, `artificialflow modeler` for process designers, and `artificialflow client` only for SDK/service-account tokens in the claim configured by `iam.auth.claimRolesPath`.

## Internal ZITADEL IAM install

```bash
helm upgrade --install artificialflow ./charts/artificialflow \
  --namespace artificialflow --create-namespace \
  -f ./charts/artificialflow/values-internal-iam.yaml \
  --set images.command.repository=REGISTRY/workflow-command \
  --set images.query.repository=REGISTRY/workflow-query \
  --set images.runtime.repository=REGISTRY/workflow-runtime \
  --set images.syncWorker.repository=REGISTRY/sync-worker \
  --set images.frontend.repository=REGISTRY/frontend \
  --set postgresql.existingSecret=artificialflow-postgres \
  --set zitadel.masterkey=REPLACE_WITH_32_CHAR_MASTERKEY \
  --set zitadel.bootstrap.adminPassword=REPLACE_WITH_ADMIN_PASSWORD
```

## Required external secrets

If `postgresql.existingSecret` is set, the secret must contain `PG_DSN` by default. Override `postgresql.existingSecretKey` if your secret uses another key.

Fresh installs use canonical `artificialflow` database, Elasticsearch, Kafka,
connector, and application-state identifiers. Existing releases must explicitly
include `values-legacy-persistent-identifiers.yaml` after correcting its
resource identity values and verifying its numeric UID. Stock images use
UID/GID `10001`; UID/GID `100` requires an explicit
`APP_UID=100`/`APP_GID=100` compatibility rebuild.

## Upgrades

Use normal Helm upgrade with the ArtificialFlow chart values. Do not rename immutable selectors in place; prefer a new release when resource identities must change.


## Routing

The gateway exposes the frontend at `/`, command API at `/api/`, and query API at `/api/query/`. Backend services stay internal through `ClusterIP` services by default.

The sync worker exposes an internal `/health` endpoint on the `syncWorker.service.port` value for readiness, liveness, and operational checks.

## Validation

```bash
make validate-helm
```

The chart validates IAM combinations during rendering. For example, `iam.mode=zitadel` requires `zitadel.enabled=true`, while `iam.mode=external` requires issuer, backend client, and frontend OIDC values.

## Production hardening

- Replace all default values for passwords, master keys, and local development hosts.
- Use TLS-enabled ingress and production OIDC issuer URLs.
- Keep `iam.auth.enforceAudience=true` unless a documented compatibility exception is required.
- Prefer externally managed Postgres, Kafka/NATS, Elasticsearch/OpenSearch, and secret management.
- Use signed ArtificialFlow release images and verify SBOM/provenance artifacts when available.
