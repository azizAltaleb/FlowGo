# Deployment Guide

FlowGo supports the same application stack with either an existing OIDC provider
or a bundled ZITADEL installation:

- **External IAM**: FlowGo validates identities created and managed by your OIDC
  provider. FlowGo does not create external users, clients, roles, or tokens.
- **Bundled IAM**: FlowGo installs ZITADEL and bootstraps the FlowGo project,
  applications, roles, initial administrator, and internal introspection client.

Docker Compose is intended for local development and evaluation. The Helm chart
is the production deployment path.

## Deployment options

- `docker-compose.external-iam.yml`: full local stack connected to an existing
  OIDC provider.
- `docker-compose.zitadel.yml`: full local stack with bundled ZITADEL.
- `docker-compose.release.yml`: override for either Compose option that uses
  published images instead of local builds.
- `charts/flowgo/values-external-iam.yaml`: Kubernetes with external IAM and
  operator-managed infrastructure.
- `charts/flowgo/values-internal-iam.yaml`: Kubernetes with bundled ZITADEL and
  operator-managed FlowGo infrastructure.

Both IAM modes deploy the command API, runtime, query API, sync worker,
frontend, and gateway. The Compose profiles also include FlowGo Postgres,
Kafka, Debezium Connect, and Elasticsearch. Bundled IAM additionally includes
ZITADEL and its database.

## Shared prerequisites

For Docker Compose:

- Docker Engine or Docker Desktop with Docker Compose v2.
- Free local ports `5432`, `8080`, `8081`, `8083`, `8092`, `9092`, `9100`, and
  `9200`; bundled IAM also requires `9180`.
- Enough Docker memory for Elasticsearch, Kafka, Debezium, FlowGo, and, in
  bundled mode, ZITADEL.
- `curl` for the post-deployment checks.
- Node.js 20 or newer only when building or testing the Node.js SDK locally.

For Kubernetes:

- A Kubernetes cluster, Helm 3, and `kubectl`.
- An ingress controller, DNS, and TLS certificates for the FlowGo hostname.
- Reachable Postgres and Elasticsearch/OpenSearch.
- Reachable Kafka plus Debezium Connect for the default Kafka projection, or a
  reachable NATS deployment when `eventBus.type=nats`.
- A container registry accessible by the cluster.
- Kubernetes Secrets or an external secret manager for every credential.
- A storage class supporting `ReadWriteMany` when using bundled ZITADEL with
  the chart defaults; its three bootstrap/auth PVCs request RWX access.
- Backup, monitoring, and restore procedures for stateful dependencies.

## External IAM prerequisites

Complete these steps in the external provider before starting FlowGo:

1. Create a backend API resource/audience. Its identifier must match
   `AUTH_CLIENT_ID` or `iam.auth.clientId`; the examples use
   `workflow-backend`.
2. Create a public browser client. The examples use `workflow-frontend`.
   Enable Authorization Code with PKCE and do not assign a client secret.
3. Register the FlowGo origin as the login redirect, silent redirect,
   post-logout redirect, and allowed web origin. Local Compose uses
   `http://localhost:9100`; Kubernetes uses the public FlowGo HTTPS origin.
4. Create a confidential machine-to-machine client with the client-credentials
   grant for SDK, worker, and automation access.
5. Create and assign the exact role names described in
   [IAM roles](iam.md#external-iam-role-provisioning).
6. Emit those roles in a claim configured by `AUTH_CLAIM_ROLES_PATH` or
   `iam.auth.claimRolesPath`.
7. Include the FlowGo backend audience in access tokens. Audience enforcement
   is enabled in the supplied external-IAM configurations.
8. Confirm that the OIDC discovery document and JWKS are reachable from the
   FlowGo containers or pods, while the public issuer is reachable from user
   browsers.

Use `AUTH_TOKEN_MODE=jwt` for self-contained JWT access tokens. For opaque
tokens, use `AUTH_TOKEN_MODE=introspection` and configure the introspection URL,
client ID, client secret, and `basic` or `post` authentication method.

## Docker Compose: external IAM

### 1. Configure the provider values

Replace every `https://login.example.com` placeholder in
`docker-compose.external-iam.yml`. Apply the same backend settings to both
`app` and `workflow-query`:

```yaml
AUTH_ISSUER_INTERNAL_URL: https://login.example.com
AUTH_ISSUER_PUBLIC_URL: https://login.example.com
AUTH_CLIENT_ID: workflow-backend
AUTH_TOKEN_MODE: jwt
AUTH_CLAIM_ROLES_PATH: roles,realm_access.roles,groups
AUTH_ENFORCE_AUDIENCE: "true"
AUTH_ALLOW_INSECURE_ISSUER: "false"
```

Configure the browser client under `frontend`:

```yaml
FRONTEND_AUTH_OIDC_AUTHORITY: https://login.example.com
FRONTEND_AUTH_OIDC_CLIENT_ID: workflow-frontend
```

`AUTH_ISSUER_INTERNAL_URL` is used for server-side provider access.
`AUTH_ISSUER_PUBLIC_URL` is the issuer seen by the browser and written into
tokens. They are normally the same HTTPS URL. If a local IAM container has
different Docker and browser addresses, use the Docker address internally, the
browser address publicly, and set `AUTH_ALLOW_INSECURE_ISSUER: "true"` only for
that local test arrangement.

For introspection, set the following values on both backend services and keep
the secret out of source control:

```yaml
AUTH_TOKEN_MODE: introspection
AUTH_INTROSPECTION_URL: https://login.example.com/oauth2/introspect
AUTH_INTROSPECTION_CLIENT_ID: workflow-introspection
AUTH_INTROSPECTION_CLIENT_SECRET: replace-me
AUTH_INTROSPECTION_AUTH_METHOD: basic
```

No custom Docker network is required for an HTTPS provider. If the provider
runs in another Compose project, attach `app` and `workflow-query` to a shared
external Docker network or expose a hostname reachable from those containers.

### 2. Validate and start

```bash
docker compose -f docker-compose.external-iam.yml config --quiet
make smoke-core
make up-external-iam
```

To use published images:

```bash
FLOWGO_IMAGE_TAG=0.2.0 make up-external-iam-release
```

### 3. External-IAM postconditions

The deployment is ready when all of these conditions hold:

```bash
make ps-external-iam
curl -fsS http://localhost:9100/api/health
curl -fsS http://localhost:9100/api/query/health
curl -fsS http://localhost:8092/health
curl -fsS http://localhost:8083/connectors/flowgo-postgres-connector/status
```

- Compose reports no exited or unhealthy required service.
- Both API health calls return `{"status":"ok"}`.
- The connector and all connector tasks report `RUNNING`.
- `http://localhost:9100` redirects to the configured provider and a user with
  `flowgo admin` or `flowgo modeler` can sign in.
- An SDK token returns an authenticated principal containing `flowgo client`:

```bash
curl -fsS \
  -H "Authorization: Bearer ${FLOWGO_TOKEN}" \
  http://localhost:9100/api/identity/me
```

Database migration and connector registration are automatic. If the connector
is missing after the command service has initialized the database, run
`make init-connector` as a recovery action and inspect
`make logs-external-iam`.

## Docker Compose: bundled ZITADEL

### 1. Validate and start

No external IAM provisioning is needed. The bootstrap creates the project,
browser application, API/introspection application, standard roles, initial
administrator, and required system identities.

```bash
docker compose -f docker-compose.zitadel.yml config --quiet
make smoke-full
make up-zitadel
```

To use published images:

```bash
FLOWGO_IMAGE_TAG=0.2.0 make up-zitadel-release
```

The first boot can take several minutes while ZITADEL initializes and FlowGo
waits for generated client files.

Important bootstrap limitation: every bundled-ZITADEL restart reconciles the
configured initial administrator and deletes other human users holding
`flowgo admin` for the FlowGo project. Until that behavior is changed, use the
single configured human administrator and do not treat additional bundled-IAM
admin assignments as durable.

### 2. Bundled-IAM postconditions

```bash
make ps-zitadel
curl -fsS http://localhost:9100/api/health
curl -fsS http://localhost:9100/api/query/health
curl -fsS http://localhost:8092/health
curl -fsS http://localhost:8083/connectors/flowgo-postgres-connector/status
```

- FlowGo is available at `http://localhost:9100`.
- ZITADEL is available at `http://localhost:9180`.
- The local development administrator can sign in with username `admin`,
  password `admin`, and email `admin@admin.localhost`.
- The administrator has `flowgo admin`; all three standard roles are visible.
- The **SDK Clients** page can create a client and download its one-time
  service-account profile.
- Both APIs are healthy and the Debezium connector is `RUNNING`.

For either Compose mode, command/query health is process-level only. Sync
health can return HTTP 200 while starting or degraded. A deployment is not
functionally verified until authenticated identity, connector/task status, and
write-to-query projection checks pass.

The default credentials, HTTP issuer, database passwords, and exposed service
ports are local-development settings only.

## Stopping Docker Compose

```bash
make down-external-iam
make down-zitadel
```

To also delete databases, search indexes, bootstrap state, and other named
volumes:

```bash
make clean-external-iam
make clean-zitadel
```

## Kubernetes and Helm

Create an environment-specific values file rather than editing the checked-in
examples. Pin every FlowGo image tag and provide production dependency
addresses:

```yaml
postgresql:
  existingSecret: flowgo-postgres
  existingSecretKey: PG_DSN

search:
  backend: elasticsearch
  address: https://elasticsearch.example.internal:9200

eventBus:
  type: kafka
  kafka:
    brokers: kafka.example.internal:9092

syncWorker:
  env:
    connectUrl: http://debezium-connect.example.internal:8083
    connectBootstrapEnabled: "false"
```

The embedded connector configuration contains Compose development connection
values. Kafka-based production deployments should disable automatic
registration and provision the connector separately through a secret-aware
deployment process. `syncWorker.env.connectorJson` can override the embedded
configuration, but it is rendered into a ConfigMap and must not contain a
database password. Postgres must permit logical replication and the connector
identity must be able to read the captured tables and create/use the
publication and slot. NATS mode does not bootstrap the Debezium connector.

The Secret selected by `postgresql.existingSecret` must contain `PG_DSN`. In
external introspection mode, it must also contain
`AUTH_INTROSPECTION_CLIENT_SECRET`, unless the chart creates the secret from
`iam.auth.introspectionClientSecret`.

### Helm with external IAM

Add the external provider settings to your environment values:

```yaml
iam:
  mode: external
  auth:
    issuerInternalUrl: https://login.example.com
    issuerPublicUrl: https://login.example.com
    clientId: workflow-backend
    tokenMode: jwt
    claimRolesPath: roles,realm_access.roles,groups
    enforceAudience: true
    allowInsecureIssuer: false
  frontend:
    oidcAuthority: https://login.example.com
    oidcClientId: workflow-frontend

zitadel:
  enabled: false

ingress:
  enabled: true
  host: flowgo.example.com
  tls:
    enabled: true
    secretName: flowgo-tls
```

Validate and install:

```bash
make validate-helm
helm lint ./charts/flowgo -f ./charts/flowgo/values-external-iam.yaml -f ./values-production.yaml
helm upgrade --install flowgo ./charts/flowgo \
  --namespace flowgo --create-namespace \
  -f ./charts/flowgo/values-external-iam.yaml \
  -f ./values-production.yaml
```

### Helm with bundled ZITADEL

Use separate HTTPS hostnames for FlowGo and ZITADEL. Configure
`zitadel.publicUrl`, `zitadel.externalDomain`, `zitadel.bootstrap.frontendUrl`,
and both ingresses consistently. The ZITADEL master key must be exactly 32
characters.

For production, set `zitadel.existingSecret` to a Secret containing:

- `ZITADEL_MASTERKEY`
- `ZITADEL_ADMIN_PASSWORD`
- `ZITADEL_POSTGRES_PASSWORD`
- `ZITADEL_DATABASE_POSTGRES_DSN`

The bootstrap automatically supplies FlowGo's generated frontend client,
project audience, roles, and authenticated introspection credentials.

Review these current bundled-chart constraints before production use:

- The three bootstrap/auth PVCs default to `ReadWriteMany`.
- Bootstrap reconciliation retains the configured administrator and deletes
  additional human FlowGo administrators on restart.
- The generated frontend OIDC application currently uses ZITADEL development
  mode.
- Bootstrap relaxes instance password policy and does not require the initial
  administrator to change the supplied password.
- Internal owner/login bootstrap PAT expirations default to 2099 and are stored
  on persistent volumes.
- Changing the frontend hostname does not reconcile redirect URIs on an
  existing generated application, and changing the configured administrator
  password does not reset an existing administrator's password.

These are explicit operational risks, not production hardening defaults.
Restrict PVC and IAM access, use a dedicated ZITADEL instance, and test restart,
credential rotation, and upgrade behavior before approving bundled mode for
production.

```bash
make validate-helm
helm lint ./charts/flowgo -f ./charts/flowgo/values-internal-iam.yaml -f ./values-production.yaml
helm upgrade --install flowgo ./charts/flowgo \
  --namespace flowgo --create-namespace \
  -f ./charts/flowgo/values-internal-iam.yaml \
  -f ./values-production.yaml
```

### Helm postconditions

```bash
kubectl get pods -n flowgo
kubectl get ingress -n flowgo
kubectl get events -n flowgo --sort-by=.lastTimestamp
curl -fsS https://flowgo.example.com/api/health
curl -fsS https://flowgo.example.com/api/query/health
```

The installation is complete when:

- Every FlowGo Deployment is available and all pods are Ready.
- The command and query health endpoints return `{"status":"ok"}`.
- The public FlowGo page loads through TLS.
- External mode completes an OIDC login and returns the expected roles from
  `/api/identity/me`.
- Bundled mode exposes the ZITADEL TLS hostname, completes bootstrap, and lets
  the configured initial administrator sign in.
- The sync worker is healthy and, for Kafka mode, the Debezium connector and
  tasks are `RUNNING`.
- A workflow can be deployed, started, and found through the query API after
  projection.

Command and query `/health` responses only prove that the HTTP process is
running; they do not test IAM, Postgres, search, or the event transport. Sync
health can return HTTP 200 with `status: "starting"` or `"degraded"` while
`healthFailOnStale` is false. Treat the authenticated identity check, connector
status, and write-to-query projection test as required postconditions rather
than relying on pod readiness alone.

## Production requirements

- Replace every example hostname and credential; never use Compose defaults.
- Use TLS for all public endpoints and untrusted network links; protect any
  intentionally internal plaintext service route with network policy.
- Keep audience validation enabled and verify that tokens contain the configured
  backend audience.
- Keep SDK tokens short-lived and rotate client secrets or private keys.
- Store SDK profiles and IAM credentials in a secret manager.
- Do not expose SDK service-account tokens to browsers.
- Back up Postgres and search state and test restoration.
- Monitor the gateway, command, runtime, query, sync worker, IAM, database,
  event transport, connector, and search services.
- Plan schema changes carefully: command and runtime currently run GORM
  `AutoMigrate` during startup; there is no serialized migration Job or
  versioned rollback mechanism.
- Validate image signatures, SBOM/provenance artifacts, resource limits,
  network policies, pod disruption, and upgrade/rollback procedures.

See [IAM](iam.md) for role and claim configuration and
[Node.js SDK](sdk-nodejs.md) for authentication examples. Published image
names and verification guidance are in [DOCKER_IMAGES.md](DOCKER_IMAGES.md).
