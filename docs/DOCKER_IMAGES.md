# Docker Images

ArtificialFlow publishes first-party images under the `artificialflow` Docker
Hub namespace. For one transition release, each build is also tagged under
`azizaltaleb`; canonical and legacy references must resolve to the exact same
multi-platform manifest digest.

## Image Repositories

| Compose service | Image | Description |
| :--- | :--- | :--- |
| `app` | `artificialflow/workflow-command` | Command API, workflow deployment, runtime-facing command endpoints, and worker API. |
| `workflow-runtime` | `artificialflow/workflow-runtime` | Runtime loops for timers, SLA checks, and background workflow execution. |
| `workflow-query` | `artificialflow/workflow-query` | CQRS read/query API backed by Elasticsearch or OpenSearch. |
| `sync-worker` | `artificialflow/sync-worker` | Debezium/Kafka projection worker for query read models. |
| `frontend` | `artificialflow/frontend` | React admin/modeler UI served by NGINX. |

## Tags

| Tag | Meaning |
| :--- | :--- |
| `v1.0.0-rc.1` | Exact ArtificialFlow release tag. Prefer this for reproducible deployments. |
| `0.3` | Latest patch in the `0.3` line. |
| `latest` | Optional convenience tag only after the release policy is explicitly enabled. |

Production deployments should pin exact version tags or image digests.

The deprecated transition aliases are `azizaltaleb/workflow-command`,
`azizaltaleb/workflow-runtime`, `azizaltaleb/workflow-query`,
`azizaltaleb/sync-worker`, and `azizaltaleb/frontend`. They are not separate
builds. The release workflow pushes both names from one Buildx invocation,
checks their manifest digests match, scans the canonical digest, and signs both
repository references.

## Compose Usage

Use the release override to switch local builds to published images:

```bash
ARTIFICIALFLOW_IMAGE_TAG=v1.0.0-rc.1 make up-zitadel-release
```

Use a staging registry or forked namespace with:

```bash
ARTIFICIALFLOW_IMAGE_REGISTRY=example-registry/artificialflow ARTIFICIALFLOW_IMAGE_TAG=v1.0.0-rc.1 make up-zitadel-release
```

To test the one-release legacy aliases explicitly:

```bash
ARTIFICIALFLOW_IMAGE_REGISTRY=azizaltaleb ARTIFICIALFLOW_IMAGE_TAG=v1.0.0-rc.1 make smoke-release-profiles
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

## Docker Hub release credentials

`DOCKERHUB_USERNAME` is the Docker Hub user that authenticates the workflow; it
is not the `artificialflow` organization name unless the account itself has
that name. The user must have push permission to all five repositories in both
the `artificialflow` organization and the legacy `azizaltaleb` namespace for
the transition release.

`DOCKERHUB_TOKEN` is a Docker Hub personal access token for that same user with
write access to those repositories. Do not store an account password. Store
both values as GitHub Actions repository or environment secrets, use a
dedicated release identity where possible, and rotate/revoke the token after
the legacy alias window. A missing repository permission can otherwise leave
only part of a release published; verify all ten references and matching
digests before announcing the release.

## Security Notes

- Compose defaults are for development and evaluation.
- Replace default credentials before production use.
- Use TLS, strict OIDC audience validation, secret management, backups, and monitoring.
- Verify signed images and review SBOM/provenance artifacts for production rollouts.
