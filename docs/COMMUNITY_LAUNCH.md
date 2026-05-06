# Community Launch Checklist

Use this checklist for repository settings and community setup that cannot be fully enforced from source files alone.

## GitHub Repository Settings

- Enable branch protection on the default branch.
- Require pull request reviews before merge.
- Require the CI and Security workflows before merge.
- Enable Dependabot alerts and security updates.
- Enable secret scanning and private vulnerability reporting.
- Enable CodeQL/code scanning for public repositories.
- Enable Discussions when maintainers are ready to support community questions.

## Repository Topics

Add these topics to improve discovery:

- `bpmn`
- `workflow-engine`
- `workflow-automation`
- `golang`
- `typescript`
- `react`
- `docker`
- `oidc`
- `zitadel`
- `sdk`

## Starter Issues

Create initial issues after the public repository is ready.

| Label | Suggested issue |
| :--- | :--- |
| `good first issue` | Improve README screenshots and quickstart screenshots. |
| `documentation` | Expand production Helm examples for managed dependencies. |
| `sdk` | Add more Node.js SDK examples for worker processing. |
| `runtime` | Add BPMN compatibility examples for advanced gateway behavior. |
| `iam` | Add external IAM provider setup recipes. |
| `frontend` | Improve modeler validation and inline BPMN lint feedback. |
| `help wanted` | Add browser E2E coverage for the bundled ZITADEL quickstart. |

## Release Readiness

Before announcing a public release, confirm:

- CI and Security pass on the release commit.
- `make release-dry-run` passes locally or in a controlled CI runner.
- Docker Hub repositories exist and repository descriptions are populated.
- npm scope/package ownership is confirmed.
- Release notes include known limitations and upgrade guidance.
