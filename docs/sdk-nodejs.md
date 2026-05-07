# Node.js SDK

The Node.js SDK package is `@gofl0w/nodejs-sdk`.

## Install

```bash
npm install @gofl0w/nodejs-sdk
```

For local development from the repository:

```bash
cd clients/nodejs-sdk
npm ci
npm run build
```

## Authentication

Use a machine-to-machine token with the `goflow client` role.

Bundled ZITADEL:

1. Sign in as a GoFlow admin.
2. Open SDK Clients.
3. Create a client token.
4. Store the token securely.

External IAM:

1. Create a machine/application client in your OIDC provider.
2. Enable client credentials flow.
3. Add the `goflow client` role to the configured roles claim.
4. Exchange client credentials for access tokens outside the SDK.

## Smoke Test

```bash
cd clients/nodejs-sdk
npm ci
npm test
npm run validate:package
GOFLOW_TOKEN=<token> GOFLOW_BASE_URL=http://localhost:9100/api node examples/sdk-smoke-test.js
```

## Publishing

The package should be published from a signed release tag through GitHub Actions with npm provenance enabled. Manual workflow dispatches default to validation and package dry-run only unless publishing is explicitly enabled by maintainers.
