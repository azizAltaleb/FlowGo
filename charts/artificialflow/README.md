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

External IAM must issue `artificialflow admin` for platform administrators, `artificialflow modeler` for process designers, and `artificialflow client` only for SDK/service-account tokens in the claim configured by `iam.auth.claimRolesPath`. Legacy `flowgo ...` roles remain accepted for one transition release.

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

## Upgrade from the former chart

Do not upgrade a `flowgo` release with the new chart defaults. Kubernetes
Deployment and StatefulSet selectors are immutable, and the new defaults would
also select new Secret and PVC names.

First inventory the live release name, every `metadata.name`, workload selector,
Secret, PVC, and StatefulSet claim. For the standard old release named
`flowgo`, render and review the compatibility upgrade:

```bash
helm template flowgo ./charts/artificialflow \
  -f ./charts/artificialflow/values-internal-iam.yaml \
  -f ./charts/artificialflow/values-legacy-persistent-identifiers.yaml \
  > /tmp/flowgo-upgrade.yaml
```

The compatibility file preserves `flowgo` fullnames, selector labels, Secrets,
and the three standalone PVC identities. Those chart-managed Secrets and PVCs
render with `helm.sh/resource-policy: keep`; the ZITADEL StatefulSet explicitly
retains claims when deleted or scaled. If the old release used another release
name or overrides, copy the file and set all three
`compatibility.legacyResourceNames` values from the live objects. Use
`applicationState.*ExistingClaim` and
`zitadel.bootstrapStorage.existingClaim` only when intentionally attaching
claims owned outside this Helm release.

The same file pins the former stock application origin
`https://flowgo.example.com`, application and IAM ingress/TLS identities, bundled
ZITADEL public/external URLs, bootstrap frontend URL, issuer/frontend authority,
and `/flowgo/bootstrap/flowgo-frontend-client-id`. This deliberately keeps the
old URL canonical while the saved ZITADEL frontend application is reused, so its
existing redirect, logout, and origin configuration remains valid. Do not
replace any of those values with `artificialflow` URLs during the compatibility
upgrade. Register and verify both old and new redirect/logout/origin values in
the existing ZITADEL application first, then perform a separately approved DNS,
ingress, issuer, CORS, and frontend-authority migration. External-IAM upgrades
must retain the live provider issuer/client values; the bundled-IAM compatibility
overrides are applied only when `iam.mode=zitadel`.

Run `make validate-helm` before applying. Its legacy render assertion rejects
renamed resources or changed immutable selectors that would create a second
stack. See the deployment guide for the backup-first blue/green procedure for
moving to canonical resource names.

For external IAM introspection mode, provide `AUTH_INTROSPECTION_CLIENT_SECRET` in the same secret or let the chart create it from `iam.auth.introspectionClientSecret`.

For bundled ZITADEL mode, production installs should provide `zitadel.existingSecret` with `ZITADEL_MASTERKEY`, `ZITADEL_ADMIN_PASSWORD`, `ZITADEL_POSTGRES_PASSWORD`, and `ZITADEL_DATABASE_POSTGRES_DSN`, or override the generated-secret values before install.

The bundled bootstrap also creates a Basic-auth ArtificialFlow API application for ZITADEL token introspection. Its generated client ID and secret are written with restrictive permissions to a dedicated auth PVC mounted only by the bootstrap and command/query services. Bundled mode validates browser JWTs, short-lived JWT Profile access tokens, and staged legacy PATs through introspection; external IAM keeps the `iam.auth.tokenMode` and credentials supplied by the operator.

`zitadel.bootstrap.accessTokenLifetime` defaults to `900s`; client keys default to `clientKeyDefaultLifetime: 2160h` and cannot exceed `clientKeyMaxLifetime: 8760h`. New SDK clients use browser-generated RSA public keys; private profiles never enter Helm values or Kubernetes resources. `enableLegacyPatCreation` and `enableLegacyPatRotation` default to `false` and should only be enabled temporarily for rollback.

Current bundled-IAM bootstrap has material production constraints: it deletes
additional human users assigned `artificialflow admin` when reconciling the configured
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
- Use signed ArtificialFlow release images and verify SBOM/provenance artifacts when available.
