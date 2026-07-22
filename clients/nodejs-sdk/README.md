# FlowGo Node.js SDK

TypeScript SDK for the FlowGo gateway API.

## Build

```bash
npm install
npm run build
```

## Client

The SDK calls FlowGo through the gateway API and needs an access token with the `flowgo client` role.

External human task inbox applications should keep that `flowgo client` token on their backend-for-frontend (BFF). The BFF authenticates the signed-in user with IAM, rejects admin/client users from the app, then calls SDK inbox methods with the acting user's username/email/roles so FlowGo can filter tasks by assignee, candidate user, or candidate group.

### Bundled ZITADEL mode

For new clients, create a private-key credential from FlowGo's **SDK Clients** page and store the downloaded profile in a secret manager. The SDK signs a five-minute RS256 assertion and exchanges it for short-lived ZITADEL access tokens automatically:

```ts
import fs from 'node:fs';
import { FlowGoClient, ZitadelJwtProfile } from '@flowgo/nodejs-sdk';

const profile = JSON.parse(
  fs.readFileSync(process.env.FLOWGO_ZITADEL_PROFILE_FILE!, 'utf8'),
) as ZitadelJwtProfile;

const client = new FlowGoClient({
  baseUrl: 'http://localhost:9100/api',
  auth: {
    type: 'zitadel-jwt-profile',
    profile,
  },
});
```

The service-account profile has this shape:

```json
{
  "type": "serviceaccount",
  "keyId": "key-id",
  "key": "-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----\n",
  "userId": "machine-user-id",
  "issuer": "https://identity.example.com",
  "tokenUrl": "https://identity.example.com/oauth/v2/token",
  "scopes": [
    "openid",
    "urn:zitadel:iam:org:projects:roles",
    "urn:zitadel:iam:org:project:id:project-id:aud"
  ]
}
```

Keep the profile out of source control, logs, command-line arguments, and environment-variable values. Restrict its file permissions to the workload identity. Rotation should add and deploy a new key before revoking the old key.

Existing legacy clients may continue using a raw PAT during migration:

- Sign in as a `flowgo admin`
- Open **SDK Clients**
- Copy the previously issued ZITADEL Personal Access Token
- Set its raw value as `FLOWGO_TOKEN` (do not include `Bearer `)

FlowGo validates legacy PATs with authenticated ZITADEL introspection on every request. Expired or revoked PATs and PATs belonging to terminated/deleted clients return `401`. New PAT issuance and rotation may be disabled by the deployment's migration policy.

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
- Create only the FlowGo roles needed by the external IAM deployment.
  - `flowgo admin`
  - `flowgo modeler`
  - `flowgo client`
- Map roles into a token claim read by `AUTH_CLAIM_ROLES_PATH`.
  - Default paths: `roles,realm_access.roles,groups`
  - The SDK/service account token must include `flowgo client`
- Create a machine-to-machine, service-account, or client-credentials application for the SDK integration.
  - Grant only `flowgo client`
  - Prefer short-lived tokens and rotate the client secret in the external IAM
- Use the SDK's OAuth client-credentials provider, a custom token callback, or
  issue a token from the external IAM and set it as `FLOWGO_TOKEN`.

Standard client-credentials example with automatic token caching and refresh:

```ts
import { FlowGoClient } from '@flowgo/nodejs-sdk';

const client = new FlowGoClient({
  baseUrl: 'http://localhost:9100/api',
  auth: {
    type: 'oauth-client-credentials',
    profile: {
      type: 'oauth-client-credentials',
      tokenUrl: 'https://login.example.com/oauth2/token',
      clientId: process.env.FLOWGO_CLIENT_ID!,
      clientSecret: process.env.FLOWGO_CLIENT_SECRET!,
    },
  },
});
```

The built-in provider sends `client_id` and `client_secret` in the form body.
If your provider requires HTTP Basic client authentication or a custom
`audience`/`resource` parameter, use its OAuth library and supply
`token: () => getProviderAccessToken()` instead.

Generic client-credentials example:

```bash
export FLOWGO_TOKEN="$(
  curl -sS -X POST "$OIDC_TOKEN_URL" \
    -H 'content-type: application/x-www-form-urlencoded' \
    -d grant_type=client_credentials \
    -d client_id="$OIDC_CLIENT_ID" \
    -d client_secret="$OIDC_CLIENT_SECRET" \
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

## Human task inbox

Use a backend-held `flowgo client` token for external task inbox applications. Pass the signed-in user's identity to each inbox call:

```ts
import { FlowGoClient } from '@flowgo/nodejs-sdk';

const client = new FlowGoClient({
  baseUrl: process.env.FLOWGO_BASE_URL,
  token: process.env.FLOWGO_TOKEN,
});

const actingUser = {
  subject: signedInUser.subject,
  username: signedInUser.username,
  email: signedInUser.email,
  name: signedInUser.name,
  roles: signedInUser.roles,
};

const inbox = await client.listInboxItems({ actingUser });
const current = await client.getInboxInstance(inbox[0].id, { actingUser, includeCompleted: true });
const tasks = await client.listUserTasks(current.id, { actingUser });
await client.claimUserTask(current.id, tasks[0].executionId, { actingUser });
await client.completeUserTask(current.id, tasks[0].executionId, { actingUser });
const history = await client.listMyCompletedTransactions({ actingUser, limit: 50 });
```

Inbox methods require a `flowgo client` SDK token plus acting-user details. Do not expose the SDK token to the browser.

## Standalone smoke test

Use `examples/sdk-smoke-test.js` to verify the SDK against a running FlowGo deployment.

Build the SDK first:

```bash
npm install
npm run build
```

Required input (set exactly one authentication option):

- `FLOWGO_BASE_URL`
  - FlowGo gateway API URL.
  - Local default: `http://localhost:9100/api`
- `FLOWGO_TOKEN`
  - Raw token text with the `flowgo client` role; never prefix it with `Bearer `.
  - Bundled ZITADEL: use only for an existing legacy client during migration.
  - External IAM: issue it from your provider service-account/client-credentials app.
- `FLOWGO_ZITADEL_PROFILE_FILE`
  - Path to a ZITADEL service-account profile downloaded from **SDK Clients**.
  - Do not print the file contents or commit the file.

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
export FLOWGO_ZITADEL_PROFILE_FILE="/run/secrets/flowgo-sdk-profile.json"
node examples/sdk-smoke-test.js
unset FLOWGO_ZITADEL_PROFILE_FILE
```

Run with workflow start, message publish, and one worker activation:

```bash
export FLOWGO_ZITADEL_PROFILE_FILE="/run/secrets/flowgo-sdk-profile.json"
FLOWGO_BASE_URL="http://localhost:9100/api" \
FLOWGO_WORKFLOW_KEY="order-process" \
FLOWGO_BUSINESS_KEY="order-123" \
FLOWGO_MESSAGE_NAME="MsgOrderPlaced" \
FLOWGO_MESSAGE_CORRELATION_KEY="order-123" \
FLOWGO_WORKER_JOB_TYPE="payment-service" \
node examples/sdk-smoke-test.js
unset FLOWGO_ZITADEL_PROFILE_FILE
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
- Human task inbox list/detail/history/claim/complete
- Signal and message publish
- Job activation, completion, failure, lock extension, capabilities
- Engine metrics
- Identity config and current principal
- Bundled ZITADEL identity user/role and legacy client-token management
  (legacy token creation and rotation may be disabled by policy)
