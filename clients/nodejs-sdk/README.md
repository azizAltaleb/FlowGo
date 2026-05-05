# Workflowsa Node.js SDK

TypeScript SDK for the Workflowsa gateway API.

## Build

```bash
npm install
npm run build
```

## Client

The SDK calls Workflowsa through the gateway API and needs an access token with the `workflowsa client` role.

### Bundled ZITADEL mode

When Workflowsa is deployed with `docker-compose.zitadel.yml`, create the token from Workflowsa:

- Sign in as a `workflowsa admin`
- Open **SDK Clients**
- Create a Workflowsa Client machine identity
- Copy the generated token immediately because it is shown only once
- Set it as `WORKFLOWSA_TOKEN`

### External IAM mode

When Workflowsa is deployed with `docker-compose.external-iam.yml`, Workflowsa does not create or manage SDK tokens. The external IAM administrator must prepare the provider and issue the SDK token.

External IAM requirements:

- Create or register a backend/API resource for Workflowsa.
  - Default audience/client ID: `workflow-backend`
  - Match the Workflowsa backend setting `AUTH_CLIENT_ID`
- Configure the Workflowsa backend issuer settings.
  - `AUTH_ISSUER_INTERNAL_URL`
  - `AUTH_ISSUER_PUBLIC_URL`
  - `AUTH_TOKEN_MODE=jwt` for JWT access tokens, or `AUTH_TOKEN_MODE=introspection` for opaque tokens
- If audience validation is enabled with `AUTH_ENFORCE_AUDIENCE=true`, issue tokens whose `aud` contains `workflow-backend`.
- Create the standard Workflowsa roles in the external IAM provider.
  - `workflowsa client`
  - `workflowsa admin`
  - `workflowsa viewer`
- Map roles into a token claim read by `AUTH_CLAIM_ROLES_PATH`.
  - Default paths: `roles,realm_access.roles,groups`
  - The SDK/service account token must include `workflowsa client`
- Create a machine-to-machine, service-account, or client-credentials application for the SDK integration.
  - Grant only `workflowsa client`
  - Prefer short-lived tokens and rotate the client secret in the external IAM
- Issue an access token from the external IAM and set it as `WORKFLOWSA_TOKEN`.

Generic client-credentials example:

```bash
export WORKFLOWSA_TOKEN="$(
  curl -sS -X POST "$OIDC_TOKEN_URL" \
    -H 'content-type: application/x-www-form-urlencoded' \
    -d grant_type=client_credentials \
    -d client_id="$OIDC_CLIENT_ID" \
    -d client_secret="$OIDC_CLIENT_SECRET" \
    -d scope="openid profile" \
  | jq -r '.access_token'
)"
```

Use your provider-specific token endpoint, client ID, client secret, scope, and audience/resource parameter if required.

```ts
import { WorkflowsaClient } from '@workflowsa/nodejs-sdk';

const client = new WorkflowsaClient({
  baseUrl: 'http://localhost:9100/api',
  token: process.env.WORKFLOWSA_TOKEN,
});

const workflows = await client.listWorkflows({ page: 1, pageSize: 100 });
const instance = await client.startInstance('order-process', { orderId: '123' });
await client.publishMessage('MsgOrderPlaced', '123', { amount: 100 });
```

## Standalone smoke test

Use `examples/sdk-smoke-test.js` to verify the SDK against a running Workflowsa deployment.

Build the SDK first:

```bash
npm install
npm run build
```

Required input:

- `WORKFLOWSA_BASE_URL`
  - Workflowsa gateway API URL.
  - Local default: `http://localhost:9100/api`
- `WORKFLOWSA_TOKEN`
  - Access token with the `workflowsa client` role.
  - Bundled ZITADEL: create it from **SDK Clients**.
  - External IAM: issue it from your provider service-account/client-credentials app.

Optional inputs:

- `WORKFLOWSA_WORKFLOW_KEY`
  - Workflow definition key or ID to start.
- `WORKFLOWSA_BUSINESS_KEY`
  - Business key for the started instance.
- `WORKFLOWSA_MESSAGE_NAME`
  - BPMN message name to publish.
- `WORKFLOWSA_MESSAGE_CORRELATION_KEY`
  - Correlation key for the message.
- `WORKFLOWSA_WORKER_JOB_TYPE`
  - Service-task job type to activate and complete once.

Run the minimal smoke test:

```bash
WORKFLOWSA_TOKEN="paste-token-here" \
node examples/sdk-smoke-test.js
```

Run with workflow start, message publish, and one worker activation:

```bash
WORKFLOWSA_BASE_URL="http://localhost:9100/api" \
WORKFLOWSA_TOKEN="paste-token-here" \
WORKFLOWSA_WORKFLOW_KEY="order-process" \
WORKFLOWSA_BUSINESS_KEY="order-123" \
WORKFLOWSA_MESSAGE_NAME="MsgOrderPlaced" \
WORKFLOWSA_MESSAGE_CORRELATION_KEY="order-123" \
WORKFLOWSA_WORKER_JOB_TYPE="payment-service" \
node examples/sdk-smoke-test.js
```

## Worker

```ts
const worker = client.createWorker('payment-service', async (job) => {
  return { paymentStatus: 'success' };
}, {
  workerName: 'payment-worker',
  autoStart: true,
});

process.on('SIGINT', () => worker.stop());
```

## API coverage

- Workflow deploy/list/search/get/delete
- Instance start/list/search/get/update variables/complete task/delete
- Signal and message publish
- Job activation, completion, failure, lock extension, capabilities
- Engine metrics
- Identity config and current principal
- Bundled ZITADEL identity user/role/client token management
