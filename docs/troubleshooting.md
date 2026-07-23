# Troubleshooting

## Compose Profile Validation

```bash
make smoke-profiles
```

## Cannot Log In

- Confirm the frontend runtime config points to the browser-visible OIDC issuer.
- Confirm the backend uses the correct internal and public issuer URLs.
- Confirm the frontend client ID was generated in bundled ZITADEL mode.
- In production, confirm `AUTH_ENFORCE_AUDIENCE=true` matches issued token audiences.

## Query Results Are Empty

- Confirm the sync worker is healthy.
- Confirm Kafka/Debezium topics exist and have assigned consumers.
- Confirm Elasticsearch/OpenSearch is reachable.
- Run the CQRS smoke test:

```bash
make cqrs-e2e-smoke
```

## SDK Calls Are Unauthorized

- Confirm the service-account profile belongs to an active machine user with `artificialflow client`.
- Confirm its scopes include `openid`, `urn:zitadel:iam:org:projects:roles`, and the ArtificialFlow project audience scope.
- Confirm clocks are synchronized; JWT Profile assertions are valid for only five minutes.
- Confirm the profile issuer and `/oauth/v2/token` URL use HTTPS outside loopback development.
- Confirm the role claim path matches `AUTH_CLAIM_ROLES_PATH`.
- In JWT mode, confirm the signed token issuer matches
  `AUTH_ISSUER_PUBLIC_URL`. In introspection mode, confirm the configured
  introspection endpoint accepts the token and returns `active=true`, a
  subject, the expected audience, and the required role claims.
- In bundled ZITADEL mode, confirm command and query use `AUTH_TOKEN_MODE=introspection`, `AUTH_INTROSPECTION_URL` ends in `/oauth/v2/introspect`, and the generated `/artificialflow/auth/artificialflow-api-client-id` and `artificialflow-api-client-secret` files are mounted and readable. Those legacy internal mount paths remain stable during the branding transition.
- A removed key cannot mint new access tokens. Already-minted tokens remain valid until their short expiry unless the client is terminated/deleted.
- During migration, legacy PATs remain accepted until explicitly revoked, but issuance and rotation are disabled by default.

## Helm Install Fails

- Render templates locally:

```bash
helm lint ./charts/artificialflow
helm template artificialflow ./charts/artificialflow -f ./charts/artificialflow/values-external-iam.yaml
helm template artificialflow ./charts/artificialflow -f ./charts/artificialflow/values-internal-iam.yaml
```

- Verify required external secrets exist.
- Verify image repositories and tags are set.
- Verify the IAM mode matches the values file: external IAM uses `iam.mode=external`, bundled ZITADEL uses `iam.mode=zitadel` and `zitadel.enabled=true`.
