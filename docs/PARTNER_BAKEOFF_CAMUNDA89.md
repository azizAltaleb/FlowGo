# Design-partner bake-off vs Camunda 8.9 Self-Managed

Scorecard for Phase C of the 1.0 displacement plan. Pass = partner completes the path on ArtificialFlow **or** explicitly prefers MIT + simpler self-hosted ops after equal functional success.

| Scenario | ArtificialFlow | Camunda 8.9 SM | Result (fill) |
| :--- | :--- | :--- | :--- |
| Install to first process | Compose/Helm + gateway | Full SM stack | |
| Deploy their BPMN (or linted import) | [CAMUNDA_BPMN_IMPORT.md](CAMUNDA_BPMN_IMPORT.md) + matrix | Native | |
| One external worker | Node/Go worker | Job worker | |
| One human task | Task Inbox / inbox API | Tasklist | |
| Their IdP | [docs/iam/](iam/) recipe | Identity/Keycloak | |
| Day-2: runtime restart / lag | Lease drill + CQRS banner | Operate | |
| Mega Supported BPMN UAT | `uat-mega-bpmn.spec.ts` | n/a | |

## ArtificialFlow runbook

```bash
make up-zitadel   # or release images after v0.4.0 publish
node scripts/mint_bakeoff_token.mjs
export ARTIFICIALFLOW_TOKEN="$(cat .local/bakeoff.token)"
export ARTIFICIALFLOW_ACTING_USER=approver
node examples/golden-demo/run-bakeoff.mjs
make test-release-suite
```

Record dates, partner name (private), and links to evidence under `reports/`.
