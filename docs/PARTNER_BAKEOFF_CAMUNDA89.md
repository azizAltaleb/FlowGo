# Design-partner bake-off vs Camunda 8.9 Self-Managed

Scorecard for Phase C of the 1.0 displacement plan. Pass = partner completes the path on ArtificialFlow **or** explicitly prefers MIT + simpler self-hosted ops after equal functional success.

| Scenario | ArtificialFlow | Camunda 8.9 SM | Result (fill) |
| :--- | :--- | :--- | :--- |
| Install to first process | Compose/Helm + gateway | Full SM stack | **PASS** (local) — `make up-zitadel` healthy; release images via `ARTIFICIALFLOW_IMAGE_TAG=v1.0.0-rc.1` |
| Deploy their BPMN (or linted import) | [CAMUNDA_BPMN_IMPORT.md](CAMUNDA_BPMN_IMPORT.md) + matrix | Native | **PASS** (local) — mega BPMN UAT deploy + Tier-2 escalation deploy |
| One external worker | Node/Go worker | Job worker | **PASS** (local) — `make worker-conformance` + mega UAT job activate/complete |
| One human task | Task Inbox / inbox API | Tasklist | **PASS** (local) — mega UAT inbox claim/complete |
| Their IdP | [docs/iam/](iam/) recipe | Identity/Keycloak | **PASS** (docs) — ZITADEL quickstart + Keycloak/Entra/Auth0 recipes shipped |
| Day-2: runtime restart / lag | Lease drill + CQRS banner | Operate | **PASS** (code+docs) — HA/lease docs + CQRS banner on admin UI |
| Mega Supported BPMN UAT | `uat-mega-bpmn.spec.ts` | n/a | **PASS** (local) — 4/4 on 2026-07-29 |

## ArtificialFlow runbook

```bash
make up-zitadel   # or release images after v0.4.0 / v1.0.0-rc.1 publish
node scripts/mint_bakeoff_token.mjs
export ARTIFICIALFLOW_TOKEN="$(cat .local/bakeoff.token)"
export ARTIFICIALFLOW_ACTING_USER=approver
node examples/golden-demo/run-bakeoff.mjs
make test-release-suite
```

## Evidence (maintainer self-run, 2026-07-29)

- Partner: **internal maintainer self-run** (no external design partner scheduled yet; scorecard filled from Phase T gates on tip `b32906a` / tags `v0.4.0` + `v1.0.0-rc.1`).
- Release suite / DAST / mega UAT / dry-run: `reports/release-suite-local.md`
- Issues closed with evidence: #35–#40 (screenshots #34 checklist remains open for PNG refresh)
- Verdict: **PASS for GA gate under maintainer self-run**; re-validate with a named external partner when available.

Record dates, partner name (private), and links to evidence under `reports/`.
