# Labels

Use labels to make FlowGo issues, PRs, and agent output easy to route. Agents may suggest labels, but maintainers decide whether to apply them unless write automation is explicitly enabled.

## Kind

| Label | Use |
| :--- | :--- |
| `kind/bug` | Reproducible defect or regression. |
| `kind/feature` | New behavior or enhancement request. |
| `kind/docs` | Documentation improvement. |
| `kind/security` | Non-sensitive security hardening or dependency item. Use `SECURITY.md` for private vulnerabilities. |
| `kind/qa` | Test coverage, reproduction, QA plan, or validation issue. |
| `kind/dependency` | Dependency, Docker image, GitHub Actions, or package update. |
| `kind/release` | Release candidate, changelog, publishing, or artifact readiness. |

## Area

| Label | Use |
| :--- | :--- |
| `area/backend` | Go backend services, engine, APIs, persistence, CQRS, or sync worker. |
| `area/frontend` | React/Vite UI, modeler, dashboards, IAM UI, or frontend API client. |
| `area/sdk` | Node.js SDK, package metadata, examples, or SDK docs. |
| `area/docs` | README, docs, community files, or examples. |
| `area/deployment` | Docker Compose, Helm, Dockerfiles, release images, or operational scripts. |
| `area/security` | Auth, IAM, dependency security, scanners, hardening, or disclosure process. |
| `area/qa` | Tests, Playwright, k6, worker conformance, CQRS smoke, or quality gates. |
| `area/community` | Discussions, support routing, onboarding, labels, templates, or governance. |

## Priority

| Label | Use |
| :--- | :--- |
| `priority/p0` | Critical release blocker, security emergency, or data-loss risk. |
| `priority/p1` | High-impact regression or important release risk. |
| `priority/p2` | Normal bug, enhancement, or quality improvement. |
| `priority/p3` | Low-risk cleanup, docs polish, or backlog item. |

## Status

| Label | Use |
| :--- | :--- |
| `status/needs-triage` | Needs initial maintainer or agent classification. |
| `status/needs-repro` | Needs reproduction steps, environment, logs, BPMN, payload, or screenshot. |
| `status/blocked` | Blocked on maintainer/contributor input or external dependency. |
| `status/ready` | Ready for implementation, review, or validation. |
| `status/accepted-risk` | Maintainer accepted a known risk or skipped gate with rationale. |

## Contributor Help

| Label | Use |
| :--- | :--- |
| `good first issue` | Small, well-scoped work with clear acceptance criteria. |
| `help wanted` | Useful contribution that does not require maintainer-only context. |

## Agent Labels

| Label | Use |
| :--- | :--- |
| `agent/advisory` | Agent produced a non-blocking summary or recommendation. |
| `agent/needs-human` | Agent found a decision that requires maintainer review. |
| `agent/approved` | Maintainer approved a privileged agent run, if configured. |
