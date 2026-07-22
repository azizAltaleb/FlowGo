# Getting Started

Choose bundled ZITADEL for the fastest self-contained start. Choose external
IAM when you need to validate FlowGo against an existing OIDC provider.

## Prerequisites

- Docker Engine or Docker Desktop
- Docker Compose v2
- `curl` for readiness checks
- Node.js 20+ if you want to run the SDK smoke test locally

See the complete [Deployment Guide](deployment.md) before a production or
Kubernetes installation.

## Option 1: Bundled ZITADEL

```bash
make smoke-full
make up-zitadel
```

Open:

- FlowGo: <http://localhost:9100>
- ZITADEL: <http://localhost:9180>

The local default admin login is:

- Username: `admin`
- Password: `admin`
- Email: `admin@admin.localhost`

These credentials are for local development only.

The bootstrap automatically creates the FlowGo project, frontend/API clients,
standard roles, and initial administrator.

### Create a bundled-IAM SDK credential

1. Sign in to FlowGo as `admin`.
2. Open the SDK Clients page.
3. Create a client with the `flowgo client` role.
4. Download the one-time service-account JSON profile immediately.
5. Store it in a secret manager. The browser-generated private key cannot be recovered.

### Run the bundled-IAM SDK smoke test

```bash
cd clients/nodejs-sdk
npm ci
npm test
FLOWGO_ZITADEL_PROFILE_FILE=/run/secrets/flowgo-service-account.json \
  FLOWGO_BASE_URL=http://localhost:9100/api \
  node examples/sdk-smoke-test.js
```

## Option 2: External IAM

Before starting FlowGo, your IAM administrator must create:

- Backend audience/client `workflow-backend`
- Public PKCE browser client `workflow-frontend`
- `flowgo admin`, `flowgo modeler`, and `flowgo client`
- A machine-to-machine client with only `flowgo client`

Edit the issuer, client, token mode, and claim values in both backend services
and the frontend section of `docker-compose.external-iam.yml`. Follow
[External IAM prerequisites](deployment.md#external-iam-prerequisites) for the
redirect URIs, audience, and claim requirements.

```bash
make smoke-core
make up-external-iam
```

Open FlowGo at <http://localhost:9100> and sign in through your provider.

Acquire a service-account token and run:

```bash
cd clients/nodejs-sdk
npm ci
npm test
FLOWGO_TOKEN="${EXTERNAL_IAM_ACCESS_TOKEN}" \
  FLOWGO_BASE_URL=http://localhost:9100/api \
  node examples/sdk-smoke-test.js
```

The token must contain the `workflow-backend` audience and
`flowgo client` role.

## Verify the deployment

```bash
curl -fsS http://localhost:9100/api/health
curl -fsS http://localhost:9100/api/query/health
curl -fsS http://localhost:8092/health
curl -fsS http://localhost:8083/connectors/flowgo-postgres-connector/status
```

Both API health endpoints must report `ok`, and the Debezium connector and its
tasks must report `RUNNING`.

## Stop the stack

```bash
make down-zitadel
make down-external-iam
```

To remove local data volumes for the selected option:

```bash
make clean-zitadel
make clean-external-iam
```

See [IAM](iam.md) for role mapping and [Node.js SDK](sdk-nodejs.md) for complete
authentication and rotation guidance.
