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

- Use a machine token with the `flowgo client` role.
- Confirm the role claim path matches `AUTH_CLAIM_ROLES_PATH`.
- Confirm the token issuer matches `AUTH_ISSUER_PUBLIC_URL`.

## Helm Install Fails

- Render templates locally:

```bash
helm lint ./charts/flowgo
helm template flowgo ./charts/flowgo -f ./charts/flowgo/values-external-iam.yaml
```

- Verify required external secrets exist.
- Verify image repositories and tags are set.
