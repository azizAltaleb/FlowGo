# Node.js SDK

The Node.js SDK package is `@flowgo/nodejs-sdk`.

## Prerequisites

- Node.js 20 or newer.
- A running FlowGo gateway.
- A non-human IAM identity with only the `flowgo client` role.
- An access token whose issuer and audience match the FlowGo deployment.
- `curl` and `jq` only when using the shell-based external-IAM smoke example.

The SDK always calls FlowGo with `Authorization: Bearer <access-token>`. The
difference between bundled and external IAM is how that access token is
obtained and refreshed.

## Installation

```bash
npm install @flowgo/nodejs-sdk
```

For local development from the repository:

```bash
cd clients/nodejs-sdk
npm ci
npm run build
```

The local package can be verified with:

```bash
npm test
npm run validate:package
```

## Common client configuration

```ts
import { FlowGoClient } from '@flowgo/nodejs-sdk';

const client = new FlowGoClient({
  baseUrl: 'https://flowgo.example.com/api',
});
```

`baseUrl` defaults to `http://localhost:9100/api`. The query URL defaults to
`${baseUrl}/query`, which matches the FlowGo gateway route. Set `queryBaseUrl`
only when calling the query service through a different address.

Configure exactly one of `auth` or `token`. The SDK rejects a client configured
with both.

## Bundled ZITADEL authentication

Bundled IAM uses a ZITADEL JWT Profile service-account credential.

### Create the credential

1. Sign in to FlowGo as a `flowgo admin`.
2. Open **SDK Clients**.
3. Create a client. FlowGo assigns only `flowgo client`.
4. Generate/download the service-account profile. The browser generates an
   RSA-2048 key pair and uploads only the public key.
5. Move the downloaded JSON directly into a secret manager. The private key is
   shown only once and cannot be recovered from FlowGo.

### Configure the SDK

```ts
import fs from 'node:fs';
import {
  FlowGoClient,
  type ZitadelJwtProfile,
} from '@flowgo/nodejs-sdk';

const profile = JSON.parse(
  fs.readFileSync('/run/secrets/flowgo-sdk-profile.json', 'utf8'),
) as ZitadelJwtProfile;

const client = new FlowGoClient({
  baseUrl: 'https://flowgo.example.com/api',
  auth: {
    type: 'zitadel-jwt-profile',
    profile,
    refreshSkewMs: 30_000,
    assertionTtlMs: 300_000,
  },
});
```

The downloaded profile contains:

```json
{
  "type": "serviceaccount",
  "keyId": "key-id",
  "key": "-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----\n",
  "userId": "machine-user-id",
  "issuer": "https://iam.flowgo.example.com",
  "tokenUrl": "https://iam.flowgo.example.com/oauth/v2/token",
  "scopes": [
    "openid",
    "urn:zitadel:iam:org:projects:roles",
    "urn:zitadel:iam:org:project:id:project-id:aud"
  ]
}
```

The SDK:

- creates an RS256 assertion valid for no more than five minutes;
- exchanges it at ZITADEL's token endpoint;
- caches the short-lived access token;
- coalesces concurrent refreshes; and
- refreshes before expiration.

`refreshSkewMs` defaults to 30 seconds. `assertionTtlMs` defaults to five
minutes and cannot exceed five minutes. Both automatic providers keep one token
in process memory, do not use refresh tokens, and do not automatically retry or
back off after a rejected exchange. Recreating the client loses the cache.

The private key is used locally to sign assertions and is never sent to FlowGo.
Do not put the JSON profile in source control, logs, command-line arguments,
frontend storage, or a plain environment variable.

Rotate a bundled credential by creating a replacement key, deploying the new
profile, verifying it, and then revoking the old key. Revoking the key blocks
new exchanges; an already-issued access token remains valid until expiry.

Raw legacy PATs remain accepted only for staged migration where the deployment
enables them. New integrations should use JWT Profile.

Bundled client keys default to 90 days and cannot exceed 365 days. Create a
replacement before expiry. The SDK's identity client-token methods target the
legacy PAT endpoints; with the default policy, PAT creation and rotation are
disabled. Create new private-key credentials through **SDK Clients**.

## External IAM authentication

### Provider prerequisites

In the external provider:

1. Create a confidential machine-to-machine application.
2. Enable the OAuth 2.0 client-credentials grant.
3. Assign only `flowgo client`.
4. Include the role in the claim configured by FlowGo's
   `AUTH_CLAIM_ROLES_PATH`.
5. Include the FlowGo backend audience, normally `workflow-backend`.
6. Use short access-token lifetimes and establish a secret-rotation procedure.

### SDK-managed client credentials

The SDK can acquire, cache, and refresh a standard client-credentials token:

```ts
import { FlowGoClient } from '@flowgo/nodejs-sdk';

const client = new FlowGoClient({
  baseUrl: 'https://flowgo.example.com/api',
  auth: {
    type: 'oauth-client-credentials',
    profile: {
      type: 'oauth-client-credentials',
      tokenUrl: 'https://login.example.com/oauth2/token',
      clientId: process.env.FLOWGO_CLIENT_ID!,
      clientSecret: process.env.FLOWGO_CLIENT_SECRET!,
    },
    refreshSkewMs: 30_000,
  },
});
```

For local Keycloak with the SDK running on the host, the token URL is typically:

```text
http://localhost:8180/realms/flowgo/protocol/openid-connect/token
```

The provider must accept `client_id` and `client_secret` in the form body and
return a Bearer `access_token` with numeric `expires_in`. Token endpoints must
use HTTPS, except loopback URLs such as `http://localhost` during local testing.
Add the optional `scopes` array only when your provider requires explicit
client-credentials scopes.

Some providers require HTTP Basic client authentication or additional
`audience`/`resource` parameters. In that case, use the provider's OAuth
library and pass a token callback:

```ts
const client = new FlowGoClient({
  baseUrl: 'https://flowgo.example.com/api',
  token: () => getProviderAccessToken(),
});
```

Implement `getProviderAccessToken` with the provider's supported OAuth library,
including its required caching and refresh behavior. The callback must return
only the token text, without the `Bearer ` prefix.

### Pre-acquired external token

A static token is convenient for a short smoke test:

```ts
const client = new FlowGoClient({
  baseUrl: 'http://localhost:9100/api',
  token: process.env.FLOWGO_TOKEN,
});
```

Do not use a static token for a long-running worker unless another component
refreshes and replaces it.

Calling `client.setToken(value)` disables any configured automatic auth
provider for that client instance and switches subsequent requests to the
supplied raw token.

## Smoke Test

Build the local SDK before running `examples/sdk-smoke-test.js`:

```bash
cd clients/nodejs-sdk
npm ci
npm test
npm run validate:package
```

Set exactly one authentication input.

Bundled ZITADEL:

```bash
FLOWGO_ZITADEL_PROFILE_FILE=/run/secrets/flowgo-service-account.json \
  FLOWGO_BASE_URL=http://localhost:9100/api \
  node examples/sdk-smoke-test.js
```

External IAM:

```bash
export FLOWGO_TOKEN="$(
  curl -fsS -X POST "${OIDC_TOKEN_URL}" \
    -H 'content-type: application/x-www-form-urlencoded' \
    -d grant_type=client_credentials \
    -d client_id="${OIDC_CLIENT_ID}" \
    -d client_secret="${OIDC_CLIENT_SECRET}" |
  jq -er '.access_token'
)"

FLOWGO_BASE_URL=http://localhost:9100/api \
  node examples/sdk-smoke-test.js

unset FLOWGO_TOKEN
```

Add provider-specific scopes and audience/resource parameters when required.
Never include `Bearer ` in `FLOWGO_TOKEN`.

Optional smoke-test inputs:

- `FLOWGO_WORKFLOW_KEY`: workflow definition key or ID to start.
- `FLOWGO_BUSINESS_KEY`: business key for the new instance.
- `FLOWGO_MESSAGE_NAME` and `FLOWGO_MESSAGE_CORRELATION_KEY`: message to
  publish.
- `FLOWGO_WORKER_JOB_TYPE`: activate and complete one service job.

The smoke test is successful when it retrieves the authenticated principal,
lists workflows, and completes every optional operation that was configured.
The principal must contain `flowgo client` and the expected audience.

## Human task inbox applications

Keep the `flowgo client` credential in a trusted backend-for-frontend. Never
send it to browser code. Authenticate the human user separately, reject
admin/client identities from the end-user inbox, and pass the acting identity
on each inbox request:

```ts
const actingUser = {
  subject: signedInUser.subject,
  username: signedInUser.username,
  email: signedInUser.email,
  name: signedInUser.name,
  roles: signedInUser.roles,
};

const inbox = await client.listInboxItems({ actingUser });
```

## Operational requirements

- Store external client secrets and bundled service-account profiles in a
  secret manager.
- Keep machine identities limited to `flowgo client`.
- Use separate credentials per application or worker.
- Rotate credentials with overlap and test the replacement before revocation.
- Do not log tokens, client secrets, JWT assertions, or private profiles.
- Treat `401` as a token acquisition/issuer/audience/expiry problem.
- Treat `403` as a valid token with an insufficient or prohibited role.
- Stop workers cleanly during shutdown so locked jobs can be recovered.

## Publishing

The package should be published from a signed release tag through GitHub Actions with npm provenance enabled. Manual workflow dispatches default to validation and package dry-run only unless publishing is explicitly enabled by maintainers.
