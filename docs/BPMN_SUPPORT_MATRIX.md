# BPMN Support Matrix

This matrix documents the ArtificialFlow BPMN feature surface.
Statuses must match runtime behavior (honesty over fake parity).

**Status legend**

| Status | Meaning |
| :--- | :--- |
| Supported | Parser + runtime execute the element as documented. |
| Partial | Usable for common patterns; edge cases need fixtures before production reliance. |
| Not supported | Deploy/lint fails or element is not modeled; do not claim support. |
| Visual-only | Modeler/XML round-trip only; not an execution primitive (Tier-3). |

## Core / Tier-1

| BPMN feature | Status | Notes |
| :--- | :--- | :--- |
| Process deployment | Supported | BPMN XML via modeler or API. |
| None start / end events | Supported | Standard start and end. |
| Message start events | Supported | `PublishMessage` / `POST /messages` starts matching definitions. |
| Message catch / throw / boundary | Supported | Correlation covered by regression tests. |
| Timer start / catch / boundary | Supported | ISO-8601 duration (`PT…`), absolute `timeDate` (RFC3339), and repeating `timeCycle` (`R[n]/PT…`, `R/PT…`). Intermediate catch is one-shot; non-interrupting boundary and start/ESP cycles re-arm until exhausted/cancelled. |
| Signal start events | Supported | `PublishSignal` / `POST /signals` / throw starts matching definitions. |
| Signal catch / throw / boundary | Supported | Throw→catch + boundary + event sub-process (ESP) signal start covered by regression tests. |
| Error boundary / end | Supported | Boundary covered by regression tests. |
| User tasks | Supported | Assignee / candidates / due date. Overdue due dates create `SLA_DUE_DATE_BREACHED` incidents and emit escalation `user-task.due-date.breached` once per job (`CheckSLAs`). List via `GET /incidents`. |
| Service tasks | Supported | External jobs via `taskType` / topic. |
| Script tasks | Supported | JavaScript via configured script runtime. |
| Business rule tasks | Supported | `decisionRef` → JSON decision tables ([DMN.md](DMN.md)). UI may be hidden. |
| Send tasks | Supported | External job (default type `io.artificialflow.connector.send`). |
| Receive tasks | Supported | Waits for correlated message. |
| Manual tasks | Supported | Waits for `CompleteTask` (no job assignment). |
| Exclusive gateway (XOR) | Supported | Conditional + default flows. |
| Parallel gateway (AND) | Supported | Fork/join. |
| Inclusive gateway (OR) | Supported | Reachability-aware join/branch. |
| Event-based gateway | Supported | Competing receive/timer (and similar) events. |
| Sub-processes (embedded) | Supported | Nested scopes. |
| Call activities | Supported | Requires called definition deployed. Optional `artificialflow:calledElementVersion` binds a specific version; default is latest. Input/output mappings via extension properties / `ioMapping`. |
| Multi-instance tasks | Supported | Parallel / sequential loop via BPMN XML (`multiInstanceLoopCharacteristics`); covered by parser + runtime fixtures. |
| Sequence flows (conditional / default) | Supported | Core token routing. |
| External / legacy Modeler extensions | Not native | See [EXTERNAL_BPMN_IMPORT.md](EXTERNAL_BPMN_IMPORT.md). |

## Tier-2

| BPMN feature | Status | Notes |
| :--- | :--- | :--- |
| Escalation events | Supported | Throw/end/boundary/catch + `PublishEscalation` / `POST /escalations` / SDK `publishEscalation`; start fan-out supported. |
| Conditional events | Supported | Catch/boundary/start wait until condition true (`CheckConditionals` / timer tick). |
| Terminate end event | Supported | Cancels sibling tokens; completes instance. |
| Cancel event + transaction sub-process | Supported | `<transaction>` + cancel end + cancel boundary. |
| Link catch/throw | Supported | Throw jumps to matching `linkEventDefinition` name. |
| Compensation | Supported | Boundary handlers + throw/end compensate (XML + unit fixtures). |
| Event sub-process | Supported | `triggeredByEvent=true`; started by matching message/signal/escalation/timer (`PT…`) / etc. |
| Complex gateway | Not supported | Rare; defer. |
| Ad-hoc sub-process | Not supported | |

## Tier-3 (visual / collaboration)

| BPMN feature | Status | Notes |
| :--- | :--- | :--- |
| Data object / store / input / output | Visual-only | Modeler shapes; variables remain the execution model. |
| Pool / lane / message flow | Visual-only | Collaboration export (`participant`/`processRef`, `laneSet`/`flowNodeRef`); lane label → user-task `candidateGroups` hint. |
| Text annotation / group / association | Visual-only | Documentation artifacts; annotation text body round-trips. |

## Examples

- Gateway patterns: [examples/bpmn/gateways/](../examples/bpmn/gateways/)
- Golden demo: [examples/golden-demo/](../examples/golden-demo/)
- Visual collaboration notes: [BPMN_VISUAL_COLLABORATION.md](BPMN_VISUAL_COLLABORATION.md)

## Tests

```bash
make validate-bpmn-capabilities
make test-bpmn-matrix
make test-bpmn-exhaustive
go test ./backend/services/workflow-command/tests -run 'TestDeployWorkflowFromBPMN_(Escalation|Conditional|MessageStart|TimerStart|SignalStart|SignalBoundary|EventSubProcess|Transaction|Compensation|MultiInstance|BoundaryError)|TestManualTaskWaits|TestCheckSLAs_PublishesDueDateBreachedEscalationOnce' -count=1
```

Canonical catalog: [tests/bpmn/matrix/capabilities.yml](../tests/bpmn/matrix/capabilities.yml).

## Connectors

Official connectors are external workers. Set service/send task `artificialflow:taskType` to the job type (modeler Properties panel offers a connector picker). Connector inputs are stored as `artificialflow:property` name/value pairs (and node `connectorInputs`); on job creation the engine copies known keys into instance variables when not already set. Start variables with the same keys can supply or override them.

| Job type | Inputs | Package |
| :--- | :--- | :--- |
| `io.artificialflow.connector.http` | `url`, `method`, `headers`, `body`, `timeoutMs`, `failOnNon2xx` | `connectors/http` |
| `io.artificialflow.connector.webhook` | `webhookUrl`, `payload`, `webhookToken` | `connectors/webhook` |
| `io.artificialflow.connector.kafka` | `kafkaTopic`, `kafkaKey`, `kafkaValue` | `connectors/kafka` |
| `io.artificialflow.connector.email` | `emailTo`, `emailSubject`, `emailBody` | `connectors/email` |
| `io.artificialflow.connector.s3` | `s3Bucket`, `s3Key`, `s3Body`, `contentType` | `connectors/s3` |

See [connectors/README.md](../connectors/README.md). Descriptor registry: `connectors/internal/common/descriptors.go` and `frontend/src/lib/connector-descriptors.ts`.

## SLA / incidents

| Surface | Status | Notes |
| :--- | :--- | :--- |
| Due-date breach escalation | Supported | Code `user-task.due-date.breached` with job/instance payload. |
| SLA incident | Supported | `ErrorType=SLA_DUE_DATE_BREACHED`; `GET /incidents`; Incidents UI. |
| Email / policy UI | Out of band | Model escalation → connector (e.g. email worker); no built-in mailer. |
