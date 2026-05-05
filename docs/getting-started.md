# Getting Started

The fastest path is the bundled ZITADEL Docker Compose stack.

## Prerequisites

- Docker
- Docker Compose
- Node.js 20+ if you want to run the SDK smoke test locally

## Start Workflowsa with Bundled ZITADEL

```bash
docker compose -f docker-compose.zitadel.yml up -d --build
```

Open:

- Workflowsa: <http://localhost:9100>
- ZITADEL: <http://localhost:9180>

The local default admin login is:

- Username: `admin`
- Password: `admin`
- Email: `admin@admin.localhost`

These credentials are for local development only.

## Create an SDK Client Token

1. Sign in to Workflowsa as `admin`.
2. Open the SDK Clients page.
3. Create a client with the `workflowsa client` role.
4. Copy the one-time token immediately.

## Run the Node.js SDK Smoke Test

```bash
cd clients/nodejs-sdk
npm ci
npm test
WORKFLOWSA_TOKEN=<token> WORKFLOWSA_BASE_URL=http://localhost:9100/api node examples/sdk-smoke-test.js
```

## Stop the Stack

```bash
docker compose -f docker-compose.zitadel.yml down
```

To remove local data volumes:

```bash
docker compose -f docker-compose.zitadel.yml down -v
```
