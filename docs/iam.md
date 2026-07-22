# IAM Guide

FlowGo supports external OIDC and bundled ZITADEL. Authentication proves the
identity and validates the token; authorization uses exact FlowGo role names
from the configured roles claim.

## External IAM

Use external mode when your organization already operates an OIDC provider.
FlowGo validates its tokens but does not create or manage identities in that
provider.

### External IAM provisioning checklist

The external IAM administrator must create:

1. A backend API resource or audience matching `AUTH_CLIENT_ID`; the deployment
   examples use `workflow-backend`.
2. A public browser client matching `FRONTEND_AUTH_OIDC_CLIENT_ID`; the examples
   use `workflow-frontend`.
3. A confidential machine-to-machine client for each SDK, worker, or automation
   integration.
4. The three standard FlowGo roles listed below.
5. Token mappings that put assigned roles into one of the configured
   `AUTH_CLAIM_ROLES_PATH` claims.

Configure the browser client for Authorization Code with PKCE and no client
secret. Its redirect URI, silent redirect URI, post-logout redirect URI, and
allowed web origin must be the FlowGo origin:

- Local Compose: `http://localhost:9100`
- Kubernetes: the public HTTPS FlowGo origin, for example
  `https://flowgo.example.com`

The frontend uses the browser origin for all three redirect URIs.

### External IAM role provisioning

Role names are matched case-insensitively after trimming, but providers and
automation should use these exact lowercase names:

- `flowgo admin`: human platform administrators. This role can administer
  workflows and instances and access privileged platform operations.
- `flowgo modeler`: human process designers. This role can access the process
  catalog and modeler but is not a platform administrator.
- `flowgo client`: non-human SDK, worker, API, and automation identities. It can
  start instances, publish messages/signals, and operate jobs. It must not be
  used as a general human login role.

Recommended assignments:

- FlowGo administrator: `flowgo admin`
- Process designer: `flowgo modeler`
- SDK/service account: `flowgo client` only
- Backend-for-frontend calling SDK inbox APIs: `flowgo client` only; pass the
  signed-in user's acting identity separately and keep the service token on the
  server

Do not grant `flowgo admin` to an SDK client. Some inbox operations explicitly
reject integration identities that also have the admin role.

The default role paths are:

```text
roles,realm_access.roles,groups
```

FlowGo checks those comma-separated paths in order. Change
`AUTH_CLAIM_ROLES_PATH` in both the command and query services, or
`iam.auth.claimRolesPath` in Helm, when your provider uses another claim.

Other claim mappings are available for provider-specific tokens:

```text
AUTH_CLAIM_SUBJECT_PATH
AUTH_CLAIM_SCOPES_PATH
AUTH_CLAIM_TENANT_PATH
AUTH_CLAIM_EMAIL_PATH
AUTH_CLAIM_NAME_PATH
```

### External token validation

For JWT access tokens:

```text
AUTH_TOKEN_MODE=jwt
AUTH_ISSUER_INTERNAL_URL=https://login.example.com
AUTH_ISSUER_PUBLIC_URL=https://login.example.com
AUTH_CLIENT_ID=workflow-backend
AUTH_ENFORCE_AUDIENCE=true
AUTH_ALLOW_INSECURE_ISSUER=false
```

The provider must expose OIDC discovery and signing keys, and each token's
audience must contain `workflow-backend` or the configured backend client ID.

For opaque access tokens:

```text
AUTH_TOKEN_MODE=introspection
AUTH_INTROSPECTION_URL=https://login.example.com/oauth2/introspect
AUTH_INTROSPECTION_CLIENT_ID=workflow-introspection
AUTH_INTROSPECTION_CLIENT_SECRET=<secret>
AUTH_INTROSPECTION_AUTH_METHOD=basic
```

Use `post` only if the provider requires credentials in the request body.
Protect the introspection secret with a secret manager.

`AUTH_ISSUER_INTERNAL_URL` is the endpoint reachable by FlowGo services.
`AUTH_ISSUER_PUBLIC_URL` is the browser-visible/token issuer. Keep them equal
and HTTPS when the public endpoint is reachable from FlowGo. If the deployment
requires separate internal and public addresses,
`AUTH_ALLOW_INSECURE_ISSUER=true` enables issuer-aware request rewriting; use
it only with a trusted internal route and an explicit security review.

## Bundled ZITADEL

Use bundled mode when FlowGo should deploy and bootstrap ZITADEL.

The bootstrap process creates:

- The FlowGo project and project audience.
- The public frontend OIDC application.
- A dedicated FlowGo API application used to authenticate token-introspection
  requests.
- `flowgo admin`, `flowgo modeler`, and `flowgo client`.
- The initial administrator with `flowgo admin`.
- System machine users required by bootstrap and login internals.

Operators do not manually create the standard roles, frontend client, backend
audience, or introspection client in bundled mode. Use FlowGo's
**Identity Management** page for human users and role assignments, and
**SDK Clients** for non-human clients.

Current bootstrap reconciliation keeps only the configured initial human
administrator among users assigned `flowgo admin`; restarting the bundled IAM
bootstrap deletes additional human FlowGo administrators. Do not rely on
multiple durable bundled-IAM administrators until this behavior is changed.
External IAM is not affected because FlowGo does not reconcile provider users.

Default local admin:

- Username: `admin`
- Password: `admin`
- Email: `admin@admin.localhost`

These values apply only to `docker-compose.zitadel.yml`. Kubernetes operators
must replace the sample administrator password and identity values.

## SDK Client Standard

For SDK and automation usage, use a machine identity with only
`flowgo client`. Do not use a human username/password flow for long-running
integrations.

In bundled mode, an administrator creates the machine user through
**SDK Clients**. The browser generates an RSA key pair, uploads only the public
key, and downloads a one-time service-account profile containing the private
key. FlowGo and ZITADEL never receive or persist that private key.

The Node.js SDK signs a five-minute JWT Profile assertion and exchanges it at
ZITADEL for a short-lived access token. Only the access token is sent to
FlowGo. Bundled FlowGo validates browser JWTs, SDK access tokens, and temporary
legacy PATs through authenticated introspection with project-audience
enforcement.

In external mode, create a provider-native client-credentials application,
assign only `flowgo client`, include the FlowGo audience and role claim in its
access tokens, and rotate its secret in the external provider.

Rotate bundled keys with overlap: add and distribute a replacement key, verify
it, then revoke the old key. Revocation prevents new exchanges with that key;
already-minted access tokens remain usable until their short expiry.
Terminating or deleting the machine user invalidates active tokens through
introspection.

Existing PATs remain listed and revocable during migration. New PAT creation
and rotation are disabled by default. Enable the emergency compatibility
switches only for a time-bounded rollback.

See [Node.js SDK](sdk-nodejs.md) for complete configuration examples.

## Authentication verification

Health endpoints are public, so a successful health check does not prove IAM is
configured correctly. Verify a real access token:

```bash
curl -fsS \
  -H "Authorization: Bearer ${FLOWGO_TOKEN}" \
  http://localhost:9100/api/identity/me
```

The response must show `authenticated: true`, the expected subject and issuer,
the FlowGo backend audience, and the required role.

- `401 Unauthorized` means the token is missing, expired, has the wrong issuer
  or audience, cannot be introspected, or fails signature validation.
- `403 Forbidden` means the token is valid but does not have a role permitted
  for that operation.
- Browser login loops usually indicate a mismatched issuer, redirect URI,
  client ID, web origin, or HTTP/HTTPS setting.

## Production Hardening

- Use HTTPS issuer URLs.
- Keep audience validation enabled.
- Grant only the minimum FlowGo role required by each identity.
- Keep human and machine identities separate.
- Keep assertions at five minutes or less and access tokens short-lived.
- Rotate client secrets and private keys regularly.
- Store downloaded service-account profiles in a secret manager, never source
  control or browser storage.
- Never put SDK client secrets or tokens in frontend code.
- Keep system machine users hidden from human identity management views.
