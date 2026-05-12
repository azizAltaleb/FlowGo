# FlowGo Node.js SDK

TypeScript SDK for the FlowGo gateway API.

## Build

```bash
npm install
npm run build
```

## Client

The SDK calls FlowGo through the gateway API and needs an access token with the `flowgo client` role.

### Bundled ZITADEL mode

When FlowGo is deployed with `docker-compose.zitadel.yml`, create the token from FlowGo:

- Sign in as a `flowgo admin`
- Open **SDK Clients**
- Create a FlowGo Client machine identity
- Copy the generated token immediately because it is shown only once
- Set it as `FLOWGO_TOKEN`

### External IAM mode

When FlowGo is deployed with `docker-compose.external-iam.yml`, FlowGo does not create or manage SDK tokens. The external IAM administrator must prepare the provider and issue the SDK token.

External IAM requirements:

- Create or register a backend/API resource for FlowGo.
  - Default audience/client ID: `workflow-backend`
  - Match the FlowGo backend setting `AUTH_CLIENT_ID`
- Configure the FlowGo backend issuer settings.
  - `AUTH_ISSUER_INTERNAL_URL`
  - `AUTH_ISSUER_PUBLIC_URL`
  - `AUTH_TOKEN_MODE=jwt` for JWT access tokens, or `AUTH_TOKEN_MODE=introspection` for opaque tokens
- If audience validation is enabled with `AUTH_ENFORCE_AUDIENCE=true`, issue tokens whose `aud` contains `workflow-backend`.
- Create the standard FlowGo roles in the external IAM provider.
  - `flowgo client`
  - `flowgo admin`
  - `flowgo viewer`
- Map roles into a token claim read by `AUTH_CLAIM_ROLES_PATH`.
  - Default paths: `roles,realm_access.roles,groups`
  - The SDK/service account token must include `flowgo client`
- Create a machine-to-machine, service-account, or client-credentials application for the SDK integration.
  - Grant only `flowgo client`
  - Prefer short-lived tokens and rotate the client secret in the external IAM
- Issue an access token from the external IAM and set it as `FLOWGO_TOKEN`.

Generic client-credentials example:

```bash
export FLOWGO_TOKEN="$(
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
import { FlowGoClient } from '@flowgo/nodejs-sdk';

const client = new FlowGoClient({
  baseUrl: 'http://localhost:9100/api',
  token: process.env.FLOWGO_TOKEN,
});

const workflows = await client.listWorkflows({ page: 1, pageSize: 100 });
const instance = await client.startInstance('order-process', { orderId: '123' });
await client.publishMessage('MsgOrderPlaced', '123', { amount: 100 });
```

## Standalone smoke test

Use `examples/sdk-smoke-test.js` to verify the SDK against a running FlowGo deployment.

Build the SDK first:

```bash
npm install
npm run build
```

Required input:

- `FLOWGO_BASE_URL`
  - FlowGo gateway API URL.
  - Local default: `http://localhost:9100/api`
- `FLOWGO_TOKEN`
  - Access token with the `flowgo client` role.
  - Bundled ZITADEL: create it from **SDK Clients**.
  - External IAM: issue it from your provider service-account/client-credentials app.

Optional inputs:

- `FLOWGO_WORKFLOW_KEY`
  - Workflow definition key or ID to start.
- `FLOWGO_BUSINESS_KEY`
  - Business key for the started instance.
- `FLOWGO_MESSAGE_NAME`
  - BPMN message name to publish.
- `FLOWGO_MESSAGE_CORRELATION_KEY`
  - Correlation key for the message.
- `FLOWGO_WORKER_JOB_TYPE`
  - Service-task job type to activate and complete once.

Run the minimal smoke test:

```bash
FLOWGO_TOKEN="paste-token-here" \
node examples/sdk-smoke-test.js
```

Run with workflow start, message publish, and one worker activation:

```bash
FLOWGO_BASE_URL="http://localhost:9100/api" \
FLOWGO_TOKEN="paste-token-here" \
FLOWGO_WORKFLOW_KEY="order-process" \
FLOWGO_BUSINESS_KEY="order-123" \
FLOWGO_MESSAGE_NAME="MsgOrderPlaced" \
FLOWGO_MESSAGE_CORRELATION_KEY="order-123" \
FLOWGO_WORKER_JOB_TYPE="payment-service" \
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
