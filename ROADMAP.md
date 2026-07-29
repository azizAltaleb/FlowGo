# ArtificialFlow Roadmap

This roadmap communicates the intended direction for the open-source project. It is not a contractual commitment.

## Public Launch: `v0.1.1`

- **GitHub readiness**: governance files, issue templates, security policy, dependency automation, and release notes.
- **Docker Hub readiness**: signed multi-architecture images with SBOMs for command, runtime, query, sync-worker, and frontend.
- **npm readiness**: public `@artificialflow/nodejs-sdk` package with package provenance.
- **Documentation**: quickstart, IAM guide, SDK guide, worker API guide, Helm deployment guide, and operations runbooks.

## P0 — Trust / 1.0 readiness

Compete with Camunda 8 self-managed on reliability, upgrade, and BPMN honesty before marketing as an alternative.

- **1.0 contract**: freeze worker, inbox, SDK, Helm, IAM, and BPMN semantics ([docs/COMPATIBILITY_MATRIX.md](docs/COMPATIBILITY_MATRIX.md), [docs/STABILITY_POLICY.md](docs/STABILITY_POLICY.md)).
- **HA evidence**: supported replica topology, runtime safety, k6 capacity numbers ([docs/HA_REFERENCE.md](docs/HA_REFERENCE.md)).
- **Day-2 ops**: upgrade guide, CQRS rebuild, outbox drain, lag SLOs ([docs/UPGRADE.md](docs/UPGRADE.md)).
- **BPMN honesty**: matrix aligned with runtime; Camunda/Zeebe import guide ([docs/BPMN_SUPPORT_MATRIX.md](docs/BPMN_SUPPORT_MATRIX.md), [docs/CAMUNDA_BPMN_IMPORT.md](docs/CAMUNDA_BPMN_IMPORT.md)).
- **Golden demo**: human task + worker + timer bake-off path under `examples/golden-demo/`.

## P1 — Daily experience

- **Browser Task Inbox** on existing `/inbox` APIs.
- **Instance ops**: failed job retry, stuck-job playbooks, clearer worker vs user-task states.
- **CQRS productization**: lag indicator, rebuild action, metrics guidance.
- **Worker DX**: published Go worker library, Node + Go samples; Python worker after Go is solid.
- **IAM recipes**: Keycloak, Entra ID, Auth0; harden bundled ZITADEL bootstrap constraints.

## P2 — Selective Camunda parity

- **DMN**: real decision-table evaluation for `decisionRef` business-rule tasks.
- **Connectors**: HTTP first, then a small official set (~8), not a marketplace.
- **Versioning**: start-on-version and open-instance migrate vs drain guidance.

## Explicitly deferred

- Camunda Optimize / process mining.
- Connector marketplace.
- Zeebe throughput leaderboard competition.
- Real multi-tenancy (tenant fields exist but deployments use `default`).
- Managed cloud / support SKU until after P1 design-partner traction.
