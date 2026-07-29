# Golden displacement demo

Camunda 8 bake-off path: **service worker + user task + boundary timer**.

## Process

File: [`order-approval.bpmn`](order-approval.bpmn)

1. Start instance with variables `{ "orderId": "…" }`.
2. External worker type `golden-validate` completes the service task.
3. User `approver` (or candidate group `approvers`) claims/completes the user task via inbox.
4. Optional: boundary timer `PT1H` on the user task demonstrates SLA/timer handling.

## Run (bundled ZITADEL release Compose)

```bash
export ARTIFICIALFLOW_IMAGE_TAG=v0.4.0
make up-zitadel-release

# Mint a short-lived SDK token from bundled ZITADEL (writes .local/bakeoff.token):
node ./scripts/mint_bakeoff_token.mjs
export ARTIFICIALFLOW_BASE_URL=http://localhost:9100/api
export ARTIFICIALFLOW_TOKEN="$(cat .local/bakeoff.token)"
export ARTIFICIALFLOW_ACTING_USER=approver

node ./examples/golden-demo/run-bakeoff.mjs
```

Optional: deploy `invoice_decision.json` from the Decisions UI (or API) before running processes that reference `decisionRef`. Published `v0.4.0` images soft-skip `/decisions` (404); use a source-built stack for DMN bake-offs.

Lease failover drill (Compose):

```bash
USE_RELEASE=1 ./scripts/runtime-lease-failover-compose.sh
```

## Capacity numbers

After the stack is warm, run `make test-perf` with `AUTH_TOKEN` / `WORKFLOW_ID` set and fill [docs/CAPACITY.md](../../docs/CAPACITY.md).

## Success criteria

- Instance reaches the user task after the worker completes.
- Inbox claim/complete succeeds for the acting user.
- Query/history UI shows the completed instance within CQRS lag SLO.
- Runtime lease failover unit test passes (`go test ./backend/services/workflow-command/internal/infrastructure/persistence -run LeaseFailover`).
