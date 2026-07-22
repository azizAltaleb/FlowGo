# FlowGo Helm Chart

This chart deploys FlowGo for production Kubernetes environments.

Default FlowGo image repositories point to Docker Hub under `azizaltaleb/*` and use a pinned release tag. Pin image tags explicitly for production rollouts.

## Deployment models

- External IAM: use an existing OIDC provider with `iam.mode=external` and `values-external-iam.yaml`.
- Bundled ZITADEL IAM: deploy solution-managed ZITADEL with `iam.mode=zitadel`, `zitadel.enabled=true`, and `values-internal-iam.yaml`.

The chart renders the same FlowGo application services for both modes: command API, runtime, query API, sync worker, frontend, and gateway. Bundled ZITADEL mode adds ZITADEL API/login, a ZITADEL Postgres StatefulSet by default, bootstrap PVCs, and an IAM ingress.

## Production dependencies

The chart expects production Postgres, Kafka or NATS, Elasticsearch or OpenSearch, and Debezium Connect endpoints to be provided through values. It does not bundle those dependencies by default.

Bundled ZITADEL also requires a storage class supporting `ReadWriteMany` for
the three bootstrap/auth PVCs with the default values.

## External IAM install

```bash
helm upgrade --install flowgo ./charts/flowgo \
  --namespace flowgo --create-namespace \
  -f ./charts/flowgo/values-external-iam.yaml \
  --set images.command.repository=REGISTRY/workflow-command \
  --set images.query.repository=REGISTRY/workflow-query \
  --set images.runtime.repository=REGISTRY/workflow-runtime \
  --set images.syncWorker.repository=REGISTRY/sync-worker \
  --set images.frontend.repository=REGISTRY/frontend \
  --set postgresql.existingSecret=flowgo-postgres \
  --set iam.auth.issuerPublicUrl=https://login.example.com \
  --set iam.auth.issuerInternalUrl=https://login.example.com \
  --set iam.frontend.oidcAuthority=https://login.example.com \
  --set iam.frontend.oidcClientId=workflow-frontend
```

External IAM must issue `flowgo admin` for FlowGo administrators, `flowgo modeler` for process designers, and `flowgo client` only for SDK/service-account tokens in the claim configured by `iam.auth.claimRolesPath`.

## Internal ZITADEL IAM install

```bash
helm upgrade --install flowgo ./charts/flowgo \
  --namespace flowgo --create-namespace \
  -f ./charts/flowgo/values-internal-iam.yaml \
  --set images.command.repository=REGISTRY/workflow-command \
  --set images.query.repository=REGISTRY/workflow-query \
  --set images.runtime.repository=REGISTRY/workflow-runtime \
  --set images.syncWorker.repository=REGISTRY/sync-worker \
  --set images.frontend.repository=REGISTRY/frontend \
  --set postgresql.existingSecret=flowgo-postgres \
  --set zitadel.masterkey=REPLACE_WITH_32_CHAR_MASTERKEY \
  --set zitadel.bootstrap.adminPassword=REPLACE_WITH_ADMIN_PASSWORD
```

## Required external secrets

If `postgresql.existingSecret` is set, the secret must contain `PG_DSN` by default. Override `postgresql.existingSecretKey` if your secret uses another key.

For external IAM introspection mode, provide `AUTH_INTROSPECTION_CLIENT_SECRET` in the same secret or let the chart create it from `iam.auth.introspectionClientSecret`.

For bundled ZITADEL mode, production installs should provide `zitadel.existingSecret` with `ZITADEL_MASTERKEY`, `ZITADEL_ADMIN_PASSWORD`, `ZITADEL_POSTGRES_PASSWORD`, and `ZITADEL_DATABASE_POSTGRES_DSN`, or override the generated-secret values before install.

The bundled bootstrap also creates a Basic-auth FlowGo API application for ZITADEL token introspection. Its generated client ID and secret are written with restrictive permissions to a dedicated auth PVC mounted only by the bootstrap and command/query services. Bundled mode validates browser JWTs, short-lived JWT Profile access tokens, and staged legacy PATs through introspection; external IAM keeps the `iam.auth.tokenMode` and credentials supplied by the operator.

`zitadel.bootstrap.accessTokenLifetime` defaults to `900s`; client keys default to `clientKeyDefaultLifetime: 2160h` and cannot exceed `clientKeyMaxLifetime: 8760h`. New SDK clients use browser-generated RSA public keys; private profiles never enter Helm values or Kubernetes resources. `enableLegacyPatCreation` and `enableLegacyPatRotation` default to `false` and should only be enabled temporarily for rollback.

Current bundled-IAM bootstrap has material production constraints: it deletes
additional human users assigned `flowgo admin` when reconciling the configured
initial administrator, creates the frontend OIDC application in development
mode, relaxes password policy, and stores long-lived bootstrap PATs on PVCs.
Review [the deployment guide](../../docs/deployment.md#helm-with-bundled-zitadel)
before approving bundled mode for production.

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
- Use signed FlowGo release images and verify SBOM/provenance artifacts when available.
