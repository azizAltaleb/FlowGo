# IAM Guide

FlowGo supports two IAM modes.

## External IAM

Use this mode when your organization already has an OIDC provider.

The external IAM administrator must:

1. Create a backend/API audience or client matching `AUTH_CLIENT_ID`.
2. Create a frontend public client for browser login.
3. Create a machine-to-machine client for SDK and worker integrations.
4. Assign FlowGo roles into a token claim configured by `AUTH_CLAIM_ROLES_PATH`.
5. Configure issuer URLs for both internal service discovery and browser-visible login.

## Bundled ZITADEL

Use this mode when FlowGo manages the local IAM provider.

The bootstrap process creates:

- FlowGo project.
- Frontend OIDC application.
- Standard roles.
- Initial admin user.
- System machine users required by bootstrap/login internals.

Default local admin:

- Username: `admin`
- Password: `admin`
- Email: `admin@admin.localhost`

## Roles

| Role | Intended holder | Access |
| :--- | :--- | :--- |
| `flowgo admin` | Platform administrators | Full platform administration. |
| `flowgo client` | SDK, API, worker, and automation clients | Programmatic workflow and worker APIs. |
| `flowgo viewer` | Auditors and read-only users | Read-only platform access. |

## SDK Client Standard

For SDK and automation usage, prefer machine-to-machine credentials and the `flowgo client` role. Do not use a human username/password flow for long-running integrations.

Bundled ZITADEL mode exposes SDK client administration in FlowGo for admins. Tokens are shown once, can be rotated, and can be revoked.

## Production Hardening

- Use HTTPS issuer URLs.
- Enforce audience validation.
- Use short-lived tokens where possible.
- Rotate machine credentials regularly.
- Avoid broad admin roles for SDK clients.
- Keep system machine users hidden from human identity management views.
