# ArtificialFlow HTTP API

Integration guide for developers who call ArtificialFlow **directly over HTTP** —
without the Node.js SDK — or who need to choose between the SDK and raw API
integration.

Stability of these surfaces is covered by [COMPATIBILITY_MATRIX.md](COMPATIBILITY_MATRIX.md)
and [STABILITY_POLICY.md](STABILITY_POLICY.md).

## Documentation map

- [Choose HTTP or an SDK](#when-to-use-the-api-vs-an-sdk)
- [Base URLs and gateway routing](#base-url-and-gateway)
- [OpenAPI, Swagger UI, and API tools](#openapi-swagger-ui-and-api-tools)
- [Authentication and authorization](#authentication)
- [Request, response, error, and CQRS conventions](#common-conventions)
- [End-to-end quickstart](#quickstart-end-to-end)
- [Command API](#command-api-reference)
- [Worker/jobs API](#worker--jobs-api)
- [Human-task Inbox API](#inbox-api)
- [Query API](#query-api-reference)
- [Integration architectures](#integration-patterns)
- [Production-readiness checklist](#production-integration-checklist)
- [Complete endpoint catalog](#endpoint-catalog-cheat-sheet)

---

## When to use the API vs an SDK

| Situation | Recommended path |
| :--- | :--- |
| Java, .NET, Ruby, PHP, or any language without an official SDK | **HTTP API** (this document) |
| Custom gateway, BFF, or ESB that already speaks REST | **HTTP API** |
| Minimal dependency / air-gapped clients that cannot pull npm packages | **HTTP API** |
| Node.js / TypeScript automation, workers, inbox BFF | [`@artificialflow/nodejs-sdk`](sdk-nodejs.md) |
| Go external workers | [`backend/libs/worker`](../backend/libs/worker/README.md) |
| Python workers (jobs only) | [`clients/python-sdk`](../clients/python-sdk/README.md) |
| Human operators / process designers | ArtificialFlow console (UI), not raw API |

The SDKs are thin clients over the same HTTP contract. Anything the SDK can do,
you can do with `curl`, `fetch`, or your language’s HTTP library against the
gateway.

---

## Base URL and gateway

Prefer the **gateway**. It is the supported public entrypoint for Compose and
Helm.

| Environment | Command (writes) | Query (reads) |
| :--- | :--- | :--- |
| Local Compose | `http://localhost:9100/api` | `http://localhost:9100/api/query` |
| Production (example) | `https://app.example.com/api` | `https://app.example.com/api/query` |
| Direct service (dev only) | `http://localhost:8080` | `http://localhost:8081` |

Gateway path rewriting:

- `GET /api/health` → command `/health`
- `POST /api/instances` → command `/instances`
- `GET /api/query/instances` → query `/instances`

There is **no `/v1` path prefix**. Compatibility is negotiated with headers
(worker protocol) and SemVer platform releases.

Public health (no auth):

```bash
curl -sS http://localhost:9100/api/health
curl -sS http://localhost:9100/api/query/health
```

---

## OpenAPI, Swagger UI, and API tools

The machine-readable contract is checked in at
[`docs/openapi.yaml`](openapi.yaml). It covers the supported gateway paths for
the Command, Query, Worker, Inbox, and bundled-IAM management APIs. Treat the
checked-in document and this guide as the release contract.

### Recommended interactive documentation

| Tool | Recommended use |
| :--- | :--- |
| [Swagger Editor](https://editor.swagger.io/) | Inspect and try the checked-in OpenAPI document |
| [Redocly CLI](https://redocly.com/docs/cli/) | Lint the contract and render a polished reference site |
| [Postman](https://www.postman.com/) or [Insomnia](https://insomnia.rest/) | Import OpenAPI, configure environments, and exercise calls |
| [Bruno](https://www.usebruno.com/) | Git-friendly local API collections without a hosted workspace |
| [OpenAPI Generator](https://openapi-generator.tech/) | Generate typed clients for Java, C#, Python, Go, and other languages |

Local preview without changing the application:

```bash
# Swagger UI
docker run --rm -p 9191:8080 \
  -e SWAGGER_JSON=/contract/openapi.yaml \
  -v "$PWD/docs:/contract:ro" \
  swaggerapi/swagger-ui
# Open http://localhost:9191

# Redoc
npx @redocly/cli preview-docs docs/openapi.yaml
```

Import `docs/openapi.yaml` into Postman, Insomnia, Bruno, or Stoplight and set:

- `server` to `http://localhost:9100/api` for local Compose;
- Bearer authentication to a short-lived machine access token;
- worker header `X-Workflow-Worker-Protocol-Version: v1`;
- an `Idempotency-Key` for job mutation requests.

### Runtime Swagger status

The command service exposes a legacy Swaggo route at
`/api/swagger/index.html`. It requires API authentication and only includes
handlers carrying Swaggo annotations, so it is not the complete `v1.1.0`
contract and should not be used for client generation. Use
[`docs/openapi.yaml`](openapi.yaml) for complete tooling imports. A future
release may serve that checked-in contract directly from the gateway.

### Contract validation

Run a mature OpenAPI linter before publishing generated clients:

```bash
npx @redocly/cli lint docs/openapi.yaml
```

Generated clients remain consumers of the HTTP contract; they do not replace
the compatibility and deprecation rules in
[STABILITY_POLICY.md](STABILITY_POLICY.md).

---

## Authentication

All protected routes require:

```http
Authorization: Bearer <access-token>
```

### Machine / integration identity

Integrations must use a non-human identity with **only** the role
`artificialflow client`.

**Bundled ZITADEL**

1. Sign in as `artificialflow admin`.
2. Open **SDK Clients** in the console.
3. Create a client and download the one-time service-account JSON profile.
4. Exchange a short-lived JWT Profile assertion at the ZITADEL token URL for an
   access token.
5. Call ArtificialFlow with that access token only.

See [iam.md](iam.md) and [sdk-nodejs.md](sdk-nodejs.md) for the JWT Profile
assertion shape. Any language that can sign RS256 JWTs can implement the
exchange.

**External OIDC (client credentials)**

```bash
TOKEN=$(curl -sS -X POST "$TOKEN_URL" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=client_credentials" \
  -d "client_id=$CLIENT_ID" \
  -d "client_secret=$CLIENT_SECRET" \
  -d "audience=workflow-backend" \
  | jq -r .access_token)
```

Audience / claim paths must match the deployment. Provider recipes:
[Keycloak](iam/KEYCLOAK.md), [Entra](iam/ENTRA.md), [Auth0](iam/AUTH0.md).

### Verify the token

```bash
curl -sS http://localhost:9100/api/identity/me \
  -H "Authorization: Bearer $TOKEN"
```

```json
{
  "authenticated": true,
  "principal": {
    "subject": "…",
    "roles": ["artificialflow client"],
    "email": "…",
    "name": "…"
  }
}
```

### Roles

| Role | Use for API integrations |
| :--- | :--- |
| `artificialflow client` | **Default for workers, automation, BFFs.** Start instances, jobs, messages/signals, query reads, inbox (with acting-user headers). |
| `artificialflow modeler` | Deploy/list workflows and decisions. **Cannot** start instances or search projected instances. |
| `artificialflow admin` | Platform ops: delete workflows/instances, incidents, job retry, admin task claim, metrics. **Do not** put admin on a long-lived SDK/worker client. |

Do not combine `artificialflow client` and `artificialflow admin` on the same
machine identity used for inbox BFFs — inbox rejects that combination.

---

## Common conventions

### Headers

| Header | When |
| :--- | :--- |
| `Authorization: Bearer …` | All protected routes |
| `Content-Type: application/json` | JSON bodies |
| `Content-Type: application/xml` | BPMN deploy (`POST /workflows`) |
| `X-Correlation-ID` | Optional request tracing |
| `Idempotency-Key` | Job mutations (`complete` / `fail` / `extend-lock`), max 128 chars |
| `X-Workflow-Worker-Protocol-Version` | Worker routes; send `v1` |
| `X-ArtificialFlow-Acting-*` | Inbox BFF acting-user contract (see [Inbox](#inbox-api)) |

Worker responses include `X-Workflow-Engine-Protocol-Version: v1`.

### Errors

Errors are plain-text bodies from `http.Error` (not a structured JSON envelope):

| Status | Typical meaning |
| ---: | :--- |
| `400` | Validation / bad id / unsupported protocol version |
| `401` | Missing or invalid token |
| `403` | Authenticated but wrong role / inbox acting-user rejected |
| `404` | Unknown workflow / instance / job |
| `409` | Conflict (e.g. task claim state) |
| `413` | BPMN body too large (limit ~20 MB) |
| `500` | Server error |

Treat non-2xx as failure; read the body for the message string.

### Command vs query

| Concern | Use |
| :--- | :--- |
| Deploy, start, cancel, jobs, inbox, messages | **Command** `/api/...` |
| Dashboard search, paginated instance lists | **Query** `/api/query/...` |

Query is **eventually consistent** (CQRS projection). After a write, poll query
or read the command instance until the projection catches up. See
[RUNBOOK_CQRS_SYNC.md](RUNBOOK_CQRS_SYNC.md).

### IDs

Most resource keys are returned as **JSON strings** even when numeric
internally (`"id": "1234567890"`). Path parameters accept those string forms.
Query instance ids and `workflowId` filters must be **numeric** strings.

---

## Quickstart (end-to-end)

Replace `$API` with `http://localhost:9100/api` and set `$TOKEN`.

```bash
export API=http://localhost:9100/api
export TOKEN=…

# 1. Verify identity
curl -sS "$API/identity/me" -H "Authorization: Bearer $TOKEN"

# 2. Deploy BPMN
curl -sS -X POST "$API/workflows" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/xml" \
  --data-binary @process.bpmn

# 3. Start an instance (BPMN process id → latest version)
curl -sS -X POST "$API/instances" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "workflow_id": "order_approval",
    "context": { "orderId": "ord-1", "amount": 1500 }
  }'

# 4. Activate and complete an external job
curl -sS -X POST "$API/jobs/activate" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Workflow-Worker-Protocol-Version: v1" \
  -d '{
    "type": "payment",
    "worker": "payments-1",
    "maxJobs": 5,
    "timeoutMs": 5000,
    "lockDurationMs": 30000
  }'

curl -sS -X POST "$API/jobs/$JOB_KEY/complete" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Workflow-Worker-Protocol-Version: v1" \
  -H "Idempotency-Key: complete-$JOB_KEY-1" \
  -d '{ "worker": "payments-1", "variables": { "paid": true } }'

# 5. Search projected instances
curl -sS "$API/query/instances?state=RUNNING&page=1&pageSize=20" \
  -H "Authorization: Bearer $TOKEN"
```

A scripted bake-off that exercises deploy → start → worker → inbox lives at
[`examples/golden-demo/run-bakeoff.mjs`](../examples/golden-demo/run-bakeoff.mjs).

---

## Command API reference

Base: `{gateway}/api`

Roles: **A** = admin, **M** = modeler, **C** = client.

### Identity

#### `GET /identity/me`

Auth: any valid token (or unauthenticated → `{ "authenticated": false }`).

#### `GET /identity/config`

Returns deployment IAM settings (`deployment_mode`, issuer URLs, claim paths,
`standard_roles`, …). Useful for BFFs that need to know how the stack is wired.

Bundled-ZITADEL administration under `/identity/management/*` is intended for
the ArtificialFlow console or trusted administrative automation. Every route
requires `artificialflow admin`, is available only in bundled-ZITADEL mode, and
must never be exposed to browser code carrying a machine credential.

| Method | Path | Purpose |
| :--- | :--- | :--- |
| `GET` | `/identity/management/clients` | List machine clients |
| `POST` | `/identity/management/clients` | Create a client and one-time credential/profile |
| `DELETE` | `/identity/management/clients/{id}` | Delete a machine client |
| `POST` | `/identity/management/clients/{id}/tokens` | Rotate/create a client token |
| `DELETE` | `/identity/management/clients/{id}/tokens/{tokenId}` | Revoke a token |
| `POST` | `/identity/management/clients/{id}/keys` | Add a JWT Profile public key |
| `DELETE` | `/identity/management/clients/{id}/keys/{keyId}` | Revoke a public key |
| `GET` | `/identity/management/users` | List users |
| `POST` | `/identity/management/users` | Create a user |
| `PUT` | `/identity/management/users/{id}` | Update a user |
| `DELETE` | `/identity/management/users/{id}` | Delete a user |
| `POST` | `/identity/management/users/{id}/terminate` | Terminate a user session/account |
| `POST` | `/identity/management/users/{id}/reactivate` | Reactivate a user |
| `GET` | `/identity/management/roles` | List roles |
| `POST` | `/identity/management/roles` | Create a custom role |
| `PUT` | `/identity/management/roles/{roleKey}` | Update a role |
| `DELETE` | `/identity/management/roles/{roleKey}` | Delete a custom role |

Prefer the console **SDK Clients** page for human administrators. Credential
creation and rotation responses can contain one-time secret material; do not
log, cache, or return those responses through a browser-facing API.

---

### Workflows (process definitions)

#### `POST /workflows` — deploy BPMN

| | |
| :--- | :--- |
| Roles | A \| M \| C |
| Body | Raw BPMN 2.0 XML |
| Content-Type | `application/xml` |
| Max size | ~20 MB |

```bash
curl -sS -X POST "$API/workflows" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/xml" \
  --data-binary @order-approval.bpmn
```

**Response** `200` — `WorkflowDefinitionResponse`:

```json
{
  "id": "123",
  "process_definition_id": "order_approval",
  "name": "Order approval",
  "version": 1,
  "resource_name": "…",
  "deployment_id": "…",
  "tenant_id": "",
  "resource_checksum": "…",
  "bpmn_xml": "…",
  "created_at": "2026-07-30T12:00:00Z",
  "steps": []
}
```

Camunda/Zeebe attributes are **not** executed natively. Lint/remap with
[EXTERNAL_BPMN_IMPORT.md](EXTERNAL_BPMN_IMPORT.md) / the Camunda import guide
before claiming compatibility.

#### `GET /workflows`

Roles: A \| M \| C. Returns an array of workflow definitions.

#### `GET /workflows/{id}`

Roles: A \| M \| C. `{id}` may be the numeric definition key or BPMN process id.

#### `DELETE /workflows/{id}`

Roles: **A** only. `204` on success.

---

### Instances

#### `POST /instances` — start

Roles: A \| C.

```json
{
  "workflow_id": "order_approval",
  "version": 2,
  "context": { "orderId": "ord-1" }
}
```

| Field | Required | Notes |
| :--- | :--- | :--- |
| `workflow_id` | yes | Numeric definition key **or** BPMN process id string |
| `version` | no | When `workflow_id` is a BPMN process id, pin that version; ignored for definition keys |
| `context` | no | Initial process variables (`map[string]any`) |

See [PROCESS_VERSIONING.md](PROCESS_VERSIONING.md).

**Response** `200` — `WorkflowInstanceResponse`:

```json
{
  "id": "456",
  "workflow_id": "123",
  "status": "ACTIVE",
  "context": { "orderId": "ord-1" },
  "created_at": "…",
  "updated_at": "…",
  "executions": [
    {
      "id": "…",
      "step_id": "…",
      "status": "…",
      "start_time": "…",
      "task": {
        "key": "…",
        "elementId": "ApproveTask",
        "executionId": "…",
        "state": "CREATED",
        "canClaim": true,
        "canComplete": false,
        "createdAt": "…",
        "updatedAt": "…"
      }
    }
  ]
}
```

#### `GET /instances`

Roles: A \| C (modeler-only principals are blocked). Active instances from the
command store (authoritative, not the search projection).

#### `GET /instances/history/completed?limit=`

Roles: scoped instance read. Default/capped limit 100.

#### `GET /instances/{id}`

Roles: scoped instance read. Full instance + executions + task metadata.

#### `DELETE /instances/{id}`

Roles: **A**. Cancels/deletes. `204`.

#### `POST /instances/{id}/variables`

Roles: A \| C.

```json
{ "variables": { "approved": true, "note": "ok" } }
```

#### `POST /instances/{id}/complete`

Roles: A \| C. Legacy step completion by `step_id` — **not** the human-task inbox
path.

```json
{ "step_id": "Activity_1" }
```

#### `GET /instances/{id}/tasks?includeCompleted=`

Roles: scoped instance read.

```json
{ "tasks": [ /* UserTaskResponse */ ] }
```

#### Admin task ops (console)

| Method | Path | Roles |
| :--- | :--- | :--- |
| `POST` | `/instances/{id}/tasks/{executionId}/claim` | A |
| `POST` | `/instances/{id}/tasks/{executionId}/complete` | A |
| `GET` | `/instances/{id}/jobs` | A |

Human-task apps for end users should use the [Inbox API](#inbox-api), not these
admin routes.

---

### Messages, signals, escalations

Roles: A \| C. Empty body fields are allowed; payload defaults to `{}`.

#### `POST /messages`

```json
{
  "message_name": "PaymentReceived",
  "correlation_key": "ord-1",
  "payload": { "amount": 1500 }
}
```

#### `POST /signals`

```json
{
  "signal_name": "GlobalCancel",
  "payload": {}
}
```

#### `POST /escalations`

```json
{
  "escalation_code": "NeedManager",
  "payload": {}
}
```

Successful publish typically returns an empty JSON object / no meaningful body.
Use instance state or query search to observe the effect.

---

### Decisions (DMN-lite)

JSON decision tables referenced by `artificialflow:decisionRef` on business-rule
tasks. Details: [DMN.md](DMN.md).

#### `POST /decisions`

Roles: A \| M \| C.

```json
{
  "decision_id": "invoice_decision",
  "name": "Invoice routing",
  "resource": "{\"id\":\"invoice_decision\",\"hitPolicy\":\"FIRST\",\"rules\":[…]}"
}
```

`resource` is a **string** containing the JSON decision table (or the raw JSON
text of the file).

#### `GET /decisions`

```json
{
  "decisions": [
    {
      "key": "1",
      "decision_id": "invoice_decision",
      "name": "Invoice routing",
      "version": 1,
      "created_at": "…",
      "updated_at": "…"
    }
  ]
}
```

#### `POST /decisions/{id}/evaluate`

Roles: A \| C. `{id}` is the decision id (e.g. `invoice_decision`).

```json
{ "inputs": { "amount": 1500 } }
```

```json
{ "outputs": { "result": "manager" } }
```

---

### Incidents, retry, metrics

| Method | Path | Roles | Notes |
| :--- | :--- | :--- | :--- |
| `GET` | `/incidents?processInstanceKey=&limit=` | A | Default limit 100 |
| `POST` | `/jobs/{key}/retry` | A | Optional body `{ "retries": 3 }` |
| `GET` | `/internal/metrics` | A | Outbox + idempotency counters |

`IncidentResponse`:

```json
{
  "key": "…",
  "id": "…",
  "processInstanceKey": "…",
  "elementInstanceKey": "…",
  "jobKey": "…",
  "errorType": "…",
  "errorMessage": "…",
  "state": "…",
  "createdAt": "…",
  "resolvedAt": null
}
```

---

## Worker / jobs API

External service tasks become lockable jobs. This is the primary integration
surface for workers in **any** language.

Protocol version: **`v1`**.

Also see the short [worker-api.md](worker-api.md) and
[WORKER_CONFORMANCE.md](WORKER_CONFORMANCE.md).

### `GET /jobs/capabilities`

Roles: A \| C.

```json
{
  "protocolVersion": "v1",
  "capabilities": ["activate", "complete", "fail", "extend-lock"]
}
```

### `POST /jobs/activate`

Roles: A \| C.

```json
{
  "type": "payment",
  "worker": "payments-1",
  "maxJobs": 5,
  "timeoutMs": 5000,
  "lockDurationMs": 30000
}
```

| Field | Meaning |
| :--- | :--- |
| `type` | Job type / topic (matches BPMN `taskType`) |
| `worker` | Stable worker name; must match on complete/fail/extend |
| `maxJobs` | Max jobs to activate in this call |
| `timeoutMs` | Long-poll / activate wait budget |
| `lockDurationMs` | How long the lock is held after activation |

**Response:**

```json
{
  "jobs": [
    {
      "key": "789",
      "type": "payment",
      "processInstanceKey": "456",
      "elementInstanceKey": "…",
      "processDefinitionKey": "123",
      "elementId": "ServiceTask_Pay",
      "worker": "payments-1",
      "retries": 3,
      "state": "ACTIVATED",
      "lockExpirationTime": "…",
      "createdAt": "…",
      "updatedAt": "…"
    }
  ]
}
```

### `POST /jobs/{key}/complete`

Roles: A \| C. Send `Idempotency-Key`.

```json
{
  "worker": "payments-1",
  "variables": { "paid": true, "txId": "…" }
}
```

### `POST /jobs/{key}/fail`

Roles: A \| C. Send `Idempotency-Key`.

```json
{
  "worker": "payments-1",
  "errorMessage": "gateway timeout",
  "retries": 2
}
```

Omit `retries` to let the engine decrement; set explicitly to override remaining
retries.

### `POST /jobs/{key}/extend-lock`

Roles: A \| C. Send `Idempotency-Key`. Use while long handlers still run.

```json
{
  "worker": "payments-1",
  "lockDurationMs": 30000
}
```

### Idempotency rules

- Scope is operation + job key, e.g. `jobs.complete:{jobKey}`.
- Replay of the same key for the same scope returns a safe `200`.
- Use **distinct** keys for complete vs fail vs extend-lock, even on the same job.
- Recommended key shape: `{operation}-{jobKey}-{attempt}` (≤ 128 chars).

### Minimal worker loop (any language)

1. `GET /jobs/capabilities` once at startup (optional).
2. Loop: `POST /jobs/activate` with your job `type`.
3. For each job: run business logic.
4. On success: `POST /jobs/{key}/complete` with `Idempotency-Key`.
5. On failure: `POST /jobs/{key}/fail` with `Idempotency-Key`.
6. If work may exceed the lock: periodically `extend-lock`.

Official connectors (HTTP, Kafka, email, …) are themselves workers against this
same API — see [`connectors/README.md`](../connectors/README.md).

gRPC `JobWorkerService` exists on the command process (`GRPC_ADDR`, default
`:50051`) but is **not** published through the nginx gateway. Prefer HTTP
`/jobs/*` unless you control pod networking.

---

## Inbox API

Human-task integration for end-user applications. Prefer a **trusted BFF** that
holds the `artificialflow client` token server-side and never exposes it to the
browser.

### Acting-user headers

When the Bearer principal is `artificialflow client` (and not admin), the BFF
**must** send the signed-in human identity:

| Header | Purpose |
| :--- | :--- |
| `X-ArtificialFlow-Acting-Subject` | Stable user id (preferred) |
| `X-ArtificialFlow-Acting-Username` | Username (used as subject fallback) |
| `X-ArtificialFlow-Acting-Email` | Email (subject **or** email required) |
| `X-ArtificialFlow-Acting-Name` | Display name |
| `X-ArtificialFlow-Acting-Roles` | Comma-separated roles / candidate groups |

Acting principals must **not** carry `artificialflow admin` or
`artificialflow client`. Candidate-group matching uses acting roles.

Browser Task Inbox humans (no platform roles) act as themselves without these
headers.

### Routes

| Method | Path | Notes |
| :--- | :--- | :--- |
| `GET` | `/inbox` | Active instances visible to the acting user |
| `GET` | `/inbox/history?limit=` | Completed history |
| `GET` | `/inbox/instances/{id}?includeCompleted=` | Instance + task metadata |
| `GET` | `/inbox/instances/{id}/tasks?includeCompleted=` | `{ "tasks": […] }` |
| `POST` | `/inbox/instances/{id}/tasks/{executionId}/claim` | → `UserTaskResponse` |
| `POST` | `/inbox/instances/{id}/tasks/{executionId}/complete` | Claimant must complete |

### Example (BFF)

```bash
curl -sS "$API/inbox" \
  -H "Authorization: Bearer $CLIENT_TOKEN" \
  -H "X-ArtificialFlow-Acting-Subject: user-42" \
  -H "X-ArtificialFlow-Acting-Username: approver" \
  -H "X-ArtificialFlow-Acting-Email: approver@example.com" \
  -H "X-ArtificialFlow-Acting-Roles: approvers"

curl -sS -X POST \
  "$API/inbox/instances/$INSTANCE_ID/tasks/$EXECUTION_ID/claim" \
  -H "Authorization: Bearer $CLIENT_TOKEN" \
  -H "X-ArtificialFlow-Acting-Subject: user-42" \
  -H "X-ArtificialFlow-Acting-Username: approver" \
  -H "X-ArtificialFlow-Acting-Roles: approvers"

curl -sS -X POST \
  "$API/inbox/instances/$INSTANCE_ID/tasks/$EXECUTION_ID/complete" \
  -H "Authorization: Bearer $CLIENT_TOKEN" \
  -H "X-ArtificialFlow-Acting-Subject: user-42" \
  -H "X-ArtificialFlow-Acting-Username: approver" \
  -H "X-ArtificialFlow-Acting-Roles: approvers"
```

`UserTaskResponse` fields: `key`, `elementId`, `executionId`, `state`,
`assignee`, `candidateUsers`, `candidateGroups`, `claimedBy`, `canClaim`,
`canComplete`, `dueDate`, `createdAt`, `updatedAt`.

---

## Query API reference

Base: `{gateway}/api/query`

Eventually consistent search / dashboard reads.

### `GET /query/instances`

Roles: A \| C.

| Query param | Default | Notes |
| :--- | :--- | :--- |
| `workflowId` | — | Numeric definition key only |
| `state` | — | `RUNNING` (mapped to `ACTIVE`), `COMPLETED`, `FAILED`, … |
| `page` | `1` | |
| `pageSize` | `20` | |

```json
{
  "instances": [
    {
      "id": "456",
      "workflow_id": "123",
      "status": "RUNNING",
      "context": {},
      "created_at": "…",
      "updated_at": "…"
    }
  ],
  "total": 1
}
```

Projected instances omit the rich `executions` / task tree. Use command
`GET /instances/{id}` or inbox routes for task detail.

### `GET /query/instances/{id}`

Roles: A \| C. `{id}` must be a numeric int64 string.

### `GET /query/workflows`

Roles: A \| M \| C. Paginated (`page`, `pageSize`). Omits `bpmn_xml` / `steps`.

```json
{
  "workflows": [
    {
      "id": "123",
      "process_definition_id": "order_approval",
      "name": "order_approval",
      "version": 1,
      "resource_name": "…",
      "deployment_id": "…",
      "tenant_id": "",
      "resource_checksum": "…",
      "created_at": "…"
    }
  ],
  "total": 1
}
```

---

## Integration patterns

### 1. Language-agnostic process client

```
Obtain machine token (client credentials or JWT Profile)
        │
        ▼
POST /workflows  (optional CI deploy)
        │
        ▼
POST /instances  { workflow_id, context }
        │
        ├─► POST /messages | /signals | /escalations   (correlation)
        │
        └─► GET /query/instances                       (status dashboards)
```

### 2. External worker (any language)

Implement the [worker loop](#minimal-worker-loop-any-language) against
`/jobs/*`. Job types must match BPMN `artificialflow:taskType` (or the engine’s
default connector types such as `io.artificialflow.connector.http`).

### 3. Human-task BFF

```
Browser ──OIDC──► Your BFF ──Bearer client token──► /inbox*
                      │
                      └── Acting-user headers from the signed-in session
```

Never ship the machine token or private key to the browser.

### 4. Decision tables from CI

Deploy JSON tables with `POST /decisions`, then reference
`artificialflow:decisionRef` from business-rule tasks in BPMN.

### 5. When an SDK is still better

Use `@artificialflow/nodejs-sdk` when you want built-in token refresh (ZITADEL
JWT Profile), worker loop helpers, typed methods, and inbox acting-user wiring
without reimplementing them. The SDK base URL is the same gateway
`…/api`.

---

## Production integration checklist

### Identity and secrets

- Use a dedicated machine identity with only `artificialflow client`.
- Keep client secrets and JWT Profile private keys in a secret manager.
- Obtain short-lived tokens server-side; never store a machine token in browser
  JavaScript or mobile application storage.
- Validate issuer and audience in production. Do not enable insecure issuer
  settings outside local development.
- Rotate credentials and test revocation before go-live.

### HTTP client behavior

- Set connect, request, and total-operation timeouts explicitly.
- Reuse connections and enable TLS certificate validation.
- Send `X-Correlation-ID` and record it with your own trace/request id.
- Retry only transient network failures, `429`, and selected `5xx` responses
  with exponential backoff and jitter.
- Do not blindly retry non-idempotent workflow starts or message publication.
  Persist a business operation id in process variables/correlation data and
  reconcile before resubmitting.
- For job complete/fail/extend-lock, reuse the same `Idempotency-Key` only when
  replaying the same operation and payload.

### Worker reliability

- Use a stable worker name per logical worker pool.
- Set lock duration above normal execution time and extend it before expiry.
- Bound `maxJobs` by worker concurrency; activation is not a concurrency
  controller.
- Treat lock ownership errors as terminal for that activation and do not commit
  external side effects twice.
- Make business-side effects idempotent using the job key or a domain operation
  key.
- Implement graceful shutdown: stop activation, finish or fail owned work, then
  exit before locks expire.

### CQRS reads

- Read command `GET /instances/{id}` when immediate post-write consistency is
  required.
- Poll Query endpoints with a bounded timeout when waiting for projection
  visibility; do not assume read-after-write consistency.
- Paginate Query results and avoid treating page totals as transactionally
  current.
- Monitor sync-worker health, projection lag, and parity operational checks.

### Compatibility and rollout

- Pin the ArtificialFlow minor line and review release notes before upgrading.
- Validate `GET /jobs/capabilities` and protocol version `v1` at worker startup.
- Diff [`docs/openapi.yaml`](openapi.yaml) in CI and regenerate clients only
  from reviewed contract changes.
- Run contract tests against a staging gateway using production IAM claim
  mappings.
- Keep a rollback path for deployment manifests and generated clients.

---

## Endpoint catalog (cheat sheet)

### Command (`/api`)

| Method | Path | Roles |
| :--- | :--- | :--- |
| `GET` | `/health` | public |
| `GET` | `/identity/me` | auth |
| `GET` | `/identity/config` | auth |
| `POST` | `/workflows` | A\|M\|C |
| `GET` | `/workflows` | A\|M\|C |
| `GET` | `/workflows/{id}` | A\|M\|C |
| `DELETE` | `/workflows/{id}` | A |
| `POST` | `/decisions` | A\|M\|C |
| `GET` | `/decisions` | A\|M\|C |
| `POST` | `/decisions/{id}/evaluate` | A\|C |
| `POST` | `/instances` | A\|C |
| `GET` | `/instances` | scoped |
| `GET` | `/instances/history/completed` | scoped |
| `GET` | `/instances/{id}` | scoped |
| `DELETE` | `/instances/{id}` | A |
| `POST` | `/instances/{id}/variables` | A\|C |
| `POST` | `/instances/{id}/complete` | A\|C |
| `GET` | `/instances/{id}/tasks` | scoped |
| `POST` | `/instances/{id}/tasks/{executionId}/claim` | A |
| `POST` | `/instances/{id}/tasks/{executionId}/complete` | A |
| `GET` | `/instances/{id}/jobs` | A |
| `GET` | `/incidents` | A |
| `POST` | `/jobs/{key}/retry` | A |
| `POST` | `/signals` | A\|C |
| `POST` | `/messages` | A\|C |
| `POST` | `/escalations` | A\|C |
| `GET` | `/jobs/capabilities` | A\|C |
| `POST` | `/jobs/activate` | A\|C |
| `POST` | `/jobs/{key}/complete` | A\|C |
| `POST` | `/jobs/{key}/fail` | A\|C |
| `POST` | `/jobs/{key}/extend-lock` | A\|C |
| `GET` | `/inbox` | inbox |
| `GET` | `/inbox/history` | inbox |
| `GET` | `/inbox/instances/{id}` | inbox |
| `GET` | `/inbox/instances/{id}/tasks` | inbox |
| `POST` | `/inbox/instances/{id}/tasks/{executionId}/claim` | inbox |
| `POST` | `/inbox/instances/{id}/tasks/{executionId}/complete` | inbox |
| `GET` | `/internal/metrics` | A |

### Query (`/api/query`)

| Method | Path | Roles |
| :--- | :--- | :--- |
| `GET` | `/health` | public |
| `GET` | `/identity/me` | auth |
| `GET` | `/instances` | A\|C |
| `GET` | `/instances/{id}` | A\|C |
| `GET` | `/workflows` | A\|M\|C |

---

## Related documentation

| Doc | Topic |
| :--- | :--- |
| [iam.md](iam.md) | Roles, OIDC, ZITADEL |
| [sdk-nodejs.md](sdk-nodejs.md) | Official Node client |
| [worker-api.md](worker-api.md) | Worker protocol summary |
| [DMN.md](DMN.md) | Decision tables |
| [PROCESS_VERSIONING.md](PROCESS_VERSIONING.md) | Start by id / version |
| [BPMN_SUPPORT_MATRIX.md](BPMN_SUPPORT_MATRIX.md) | Executable BPMN surface |
| [COMPATIBILITY_MATRIX.md](COMPATIBILITY_MATRIX.md) | Compatibility guarantees |
| [architecture.md](architecture.md) | CQRS / gateway model |

Authoritative Go DTOs:
`backend/services/workflow-command/internal/interfaces/http/dto/dto.go` and
query mappers under
`backend/services/workflow-query/internal/interfaces/http/`.
