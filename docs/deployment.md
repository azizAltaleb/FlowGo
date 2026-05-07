# Deployment Guide

## Supported Deployment Profiles

| Profile | File | Intended use |
| :--- | :--- | :--- |
| External IAM Compose | `docker-compose.external-iam.yml` | Development/evaluation with an existing OIDC provider. |
| Bundled ZITADEL Compose | `docker-compose.zitadel.yml` | Development/evaluation with solution-managed ZITADEL. |
| Published images Compose override | `docker-compose.release.yml` | Runs release images from Docker Hub instead of local builds. |
| External IAM Helm | `charts/goflow/values-external-iam.yaml` | Production Kubernetes with external OIDC and managed infrastructure. |
| Bundled ZITADEL Helm | `charts/goflow/values-internal-iam.yaml` | Kubernetes deployment with bundled ZITADEL. |

## Docker Compose

Docker Compose files expose service ports for local debugging and use development credentials. They are not production hardened.

Validate the compose profiles:

```bash
make smoke-profiles
```

Run published Docker Hub images instead of building locally:

```bash
GOFLOW_IMAGE_TAG=0.1.0 make up-zitadel-release
```

Validate the published-image compose override:

```bash
make smoke-release-profiles
```

The image registry defaults to `gofl0w`; override it with `GOFLOW_IMAGE_REGISTRY` when testing a fork or staging registry.

Published image names, tag policy, and verification guidance are documented in [DOCKER_IMAGES.md](DOCKER_IMAGES.md).

## Helm

The Helm chart is in `charts/goflow`.

Production Helm deployments should provide:

- Postgres DSN through a Kubernetes Secret.
- Kafka or NATS endpoint.
- Elasticsearch or OpenSearch endpoint.
- OIDC issuer and frontend client configuration.
- TLS-enabled ingress.
- Resource limits and pod scheduling rules.

Example:

```bash
helm upgrade --install goflow ./charts/goflow \
  --namespace goflow --create-namespace \
  -f ./charts/goflow/values-external-iam.yaml
```

## Production Requirements

- Replace all default credentials.
- Use TLS for browser, API, and IAM endpoints.
- Enable strict audience validation for production OIDC tokens.
- Store secrets in Kubernetes Secrets or an external secret manager.
- Configure backups for Postgres and search indexes.
- Monitor command, query, sync-worker, runtime, gateway, IAM, Kafka/NATS, and search services.
