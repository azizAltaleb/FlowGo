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
| Message start events | Supported | `PublishMessage` starts matching definitions. |
| Message catch / throw / boundary | Supported | Correlation covered by regression tests. |
| Timer start / catch / boundary | Supported | Duration/date patterns; boundary interrupting and non-interrupting. |
| Signal start events | Supported | `PublishSignal` / throw starts matching definitions. |
| Signal catch / throw / boundary | Supported | Throw→catch + boundary covered by regression tests. |
| Error boundary / end | Supported | Boundary covered by regression tests. |
| User tasks | Supported | Assignee / candidates / due date. |
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
| Call activities | Supported | Requires called definition deployed. |
| Multi-instance tasks | Supported | Parallel loop metadata. |
| Sequence flows (conditional / default) | Supported | Core token routing. |
| External / legacy Modeler extensions | Not native | See [EXTERNAL_BPMN_IMPORT.md](EXTERNAL_BPMN_IMPORT.md). |

## Tier-2

| BPMN feature | Status | Notes |
| :--- | :--- | :--- |
| Escalation events | Supported | Throw/end/boundary/catch + `PublishEscalation`; start fan-out supported. |
| Conditional events | Supported | Catch/boundary/start wait until condition true (`CheckConditionals` / timer tick). |
| Terminate end event | Supported | Cancels sibling tokens; completes instance. |
| Cancel event + transaction sub-process | Supported | `<transaction>` + cancel end + cancel boundary. |
| Link catch/throw | Supported | Throw jumps to matching `linkEventDefinition` name. |
| Compensation | Supported | Boundary handlers + throw/end compensate (XML + unit fixtures). |
| Event sub-process | Supported | `triggeredByEvent=true`; started by matching message/signal/escalation/etc. |
| Complex gateway | Not supported | Rare; defer. |
| Ad-hoc sub-process | Not supported | |

## Tier-3 (visual / collaboration)

| BPMN feature | Status | Notes |
| :--- | :--- | :--- |
| Data object / store / input / output | Visual-only | Modeler shapes; variables remain the execution model. |
| Pool / lane / message flow | Visual-only | Collaboration shapes; lane label → user-task `candidateGroups` hint on export. |
| Text annotation / group / association | Visual-only | Documentation artifacts. |

## Examples

- Gateway patterns: [examples/bpmn/gateways/](../examples/bpmn/gateways/)
- Golden demo: [examples/golden-demo/](../examples/golden-demo/)
- Visual collaboration notes: [BPMN_VISUAL_COLLABORATION.md](BPMN_VISUAL_COLLABORATION.md)

## Tests

```bash
make test-bpmn-matrix
make test-bpmn-exhaustive
go test ./backend/services/workflow-command/tests -run 'TestDeployWorkflowFromBPMN_(Escalation|Conditional|MessageStart|Transaction|Compensation)|TestManualTaskWaits' -count=1
```
