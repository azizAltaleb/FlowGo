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
| Planned (Tier-2) | Targeted after 1.0 for partner-blocking Camunda imports. |

## Core / Tier-1 (1.0 release train)

| BPMN feature | Status | Notes |
| :--- | :--- | :--- |
| Process deployment | Supported | BPMN XML via modeler or API. |
| None start / end events | Supported | Standard start and end. |
| Message start events | Partial | Parsed (`message_ref` on start); instance creation still via API start or message correlation patterns — prefer receive/boundary for production. |
| Message catch / throw / boundary | Supported | Correlation covered by regression tests. |
| Timer start / catch / boundary | Supported | Duration/date patterns; boundary interrupting and non-interrupting. |
| Signal start events | Partial | `signal_ref` parsed on start; prefer catch/throw for production. |
| Signal catch / throw / boundary | Supported | Throw→catch + boundary covered by regression tests. |
| Error boundary / end | Supported | Boundary covered by regression tests. |
| User tasks | Supported | Assignee / candidates / due date. |
| Service tasks | Supported | External jobs via `taskType` / topic. |
| Script tasks | Supported | JavaScript via configured script runtime. |
| Business rule tasks | Supported | `decisionRef` → JSON decision tables ([DMN.md](DMN.md)). UI may be hidden. |
| Send tasks | Supported | External job (default type `io.artificialflow.connector.send`); complete via worker/connector. |
| Receive tasks | Supported | Waits for correlated message. |
| Manual tasks | Supported | Pass-through / auto-complete style execution. |
| Exclusive gateway (XOR) | Supported | Conditional + default flows. |
| Parallel gateway (AND) | Supported | Fork/join. |
| Inclusive gateway (OR) | Supported | Reachability-aware join/branch. |
| Event-based gateway | Supported | Competing receive/timer (and similar) events. |
| Sub-processes (embedded) | Supported | Nested scopes. |
| Call activities | Supported | Requires called definition deployed. |
| Multi-instance tasks | Supported | Parallel loop metadata. |
| Sequence flows (conditional / default) | Supported | Core token routing. |
| Camunda / Zeebe extensions | Not native | See [CAMUNDA_BPMN_IMPORT.md](CAMUNDA_BPMN_IMPORT.md). |

## Tier-2 (engine)

| BPMN feature | Status | Notes |
| :--- | :--- | :--- |
| Escalation events | Not supported | Deploy/parse fails with clear error (no silent no-op). |
| Conditional events | Not supported | Deploy/parse fails with clear error. |
| Terminate end event | Supported | Cancels sibling tokens; completes instance. |
| Cancel event + transaction sub-process | Not supported | Deploy/parse fails with clear error. |
| Link catch/throw | Supported | Throw jumps to matching `linkEventDefinition` name. |
| Compensation (full graphs) | Partial | Subset exists; expand fixtures before claiming Supported. |
| Event sub-process | Not supported | `triggeredByEvent=true` rejected at parse. |
| Complex gateway | Not supported | Rare; defer. |
| Ad-hoc sub-process | Not supported | |

## Tier-3 (visual / collaboration)

| BPMN feature | Status | Notes |
| :--- | :--- | :--- |
| Data object / store / input / output | Visual-only | Modeler round-trip; variables remain the execution model. |
| Pool / lane / message flow | Visual-only | Collaboration shapes in modeler; not executable tokens. |
| Text annotation / group / association | Visual-only | Documentation artifacts in diagram XML. |

## Examples

- Gateway patterns: [examples/bpmn/gateways/](../examples/bpmn/gateways/)
- Golden demo: [examples/golden-demo/](../examples/golden-demo/)

## Tests

```bash
make test-bpmn-matrix
make test-bpmn-exhaustive
```
