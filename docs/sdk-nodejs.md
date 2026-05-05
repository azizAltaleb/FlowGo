# Node.js SDK

The Node.js SDK package is `@workflowsa/nodejs-sdk`.

## Install

```bash
npm install @workflowsa/nodejs-sdk
```

For local development from the repository:

```bash
cd clients/nodejs-sdk
npm ci
npm run build
```

## Authentication

Use a machine-to-machine token with the `workflowsa client` role.

Bundled ZITADEL:

1. Sign in as a Workflowsa admin.
2. Open SDK Clients.
3. Create a client token.
4. Store the token securely.

External IAM:

1. Create a machine/application client in your OIDC provider.
2. Enable client credentials flow.
3. Add the `workflowsa client` role to the configured roles claim.
4. Exchange client credentials for access tokens outside the SDK.

## Smoke Test

```bash
cd clients/nodejs-sdk
npm ci
npm test
WORKFLOWSA_TOKEN=<token> WORKFLOWSA_BASE_URL=http://localhost:9100/api node examples/sdk-smoke-test.js
```

## Publishing

The package should be published from a signed release tag through GitHub Actions with npm provenance enabled.
