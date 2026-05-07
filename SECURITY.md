# Security Policy

## Supported Versions

GoFlow is pre-1.0 software. Security fixes are provided for the latest released minor version and the `main` branch unless a release note states otherwise.

| Version | Supported |
| :--- | :--- |
| `main` | Yes |
| `0.x` latest minor | Yes |
| Older `0.x` minors | Best effort |

## Reporting a Vulnerability

Please do not open public GitHub issues for suspected vulnerabilities.

Report security issues by emailing the maintainers at `security@goflow.dev` or by using GitHub private vulnerability reporting after the public repository is enabled.

Include:

- **Impact**: affected component and expected severity.
- **Reproduction**: steps, request payloads, logs, or proof of concept.
- **Environment**: GoFlow version, deployment mode, IAM mode, and Docker/Helm details.
- **Mitigation**: any workaround you have identified.

We aim to acknowledge reports within 3 business days and provide a remediation plan after validation.

## Security Expectations

- **Local defaults**: Docker Compose defaults are for development and evaluation only.
- **Credentials**: replace all default passwords and generated bootstrap credentials before production use.
- **IAM**: production deployments should enforce HTTPS issuer URLs, strict audience validation, short-lived tokens, and least-privilege roles.
- **Secrets**: use Kubernetes Secrets, an external secret manager, or equivalent platform secret storage.
- **Images**: use signed release images and verify SBOM/provenance artifacts when available.

## Scope

In scope:

- GoFlow command, query, runtime, sync-worker, frontend, Helm chart, Docker images, and Node.js SDK.
- Authentication, authorization, token handling, workflow execution isolation, and supply-chain issues.

Out of scope:

- Vulnerabilities in third-party services deployed and configured independently, unless GoFlow documentation or defaults make them exploitable.
