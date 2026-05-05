# Workflowsa Roadmap

This roadmap communicates the intended direction for the open-source project. It is not a contractual commitment.

## Public Launch: `v0.1.0`

- **GitHub readiness**: governance files, issue templates, security policy, dependency automation, and release notes.
- **Docker Hub readiness**: signed multi-architecture images with SBOMs for command, runtime, query, sync-worker, and frontend.
- **npm readiness**: public `@workflowsa/nodejs-sdk` package with package provenance.
- **Documentation**: quickstart, IAM guide, SDK guide, worker API guide, Helm deployment guide, and operations runbooks.

## Near Term

- **Compatibility**: publish API, worker protocol, SDK, and deployment compatibility matrix.
- **Helm hardening**: production examples for external Postgres, Kafka/NATS, Elasticsearch/OpenSearch, ingress, TLS, and secrets.
- **Observability**: finalize tracing, logs, metrics, and IAM-integrated observability guidance.
- **Testing**: expand browser E2E, SDK conformance, Helm template, and image vulnerability gates.

## Later

- **Scale guidance**: HA reference architectures and load-test evidence.
- **Plugin model**: safer extension points for custom task handlers and connectors.
- **Deployment automation**: optional GitOps examples and Kubernetes production runbooks.
