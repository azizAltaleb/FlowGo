# Docker Images

FlowGo publishes first-party images under the `azizaltaleb` Docker Hub namespace.

## Image Repositories

| Compose service | Image | Description |
| :--- | :--- | :--- |
| `app` | `azizaltaleb/workflow-command` | Command API, workflow deployment, runtime-facing command endpoints, and worker API. |
| `workflow-runtime` | `azizaltaleb/workflow-runtime` | Runtime loops for timers, SLA checks, and background workflow execution. |
| `workflow-query` | `azizaltaleb/workflow-query` | CQRS read/query API backed by Elasticsearch or OpenSearch. |
| `sync-worker` | `azizaltaleb/sync-worker` | Debezium/Kafka projection worker for query read models. |
| `frontend` | `azizaltaleb/frontend` | React admin/modeler UI served by NGINX. |

## Tags

| Tag | Meaning |
| :--- | :--- |
| `v0.1.1` | Exact FlowGo release tag. Prefer this for reproducible deployments. |
| `0.1` | Latest patch in the `0.1` line. |
| `latest` | Optional convenience tag only after the release policy is explicitly enabled. |

Production deployments should pin exact version tags or image digests.

## Compose Usage

Use the release override to switch local builds to published images:

```bash
FLOWGO_IMAGE_TAG=v0.1.1 make up-zitadel-release
```

Use a staging registry or forked namespace with:

```bash
FLOWGO_IMAGE_REGISTRY=example-registry/flowgo FLOWGO_IMAGE_TAG=v0.1.1 make up-zitadel-release
```

Validate the release override without starting containers:

```bash
make smoke-release-profiles
```

## Build Metadata

Release images include OCI labels for:

- `org.opencontainers.image.title`
- `org.opencontainers.image.description`
- `org.opencontainers.image.source`
- `org.opencontainers.image.licenses`
- `org.opencontainers.image.version`
- `org.opencontainers.image.revision`

The release workflow builds `linux/amd64` and `linux/arm64` images, requests SBOM/provenance attestations, scans published images, and signs images with Cosign.

## Security Notes

- Compose defaults are for development and evaluation.
- Replace default credentials before production use.
- Use TLS, strict OIDC audience validation, secret management, backups, and monitoring.
- Verify signed images and review SBOM/provenance artifacts for production rollouts.
