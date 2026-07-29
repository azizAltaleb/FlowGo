# Camunda / Zeebe BPMN Import Guide

ArtificialFlow accepts BPMN 2.0 XML with `artificialflow:` extensions. Camunda 8 Modeler diagrams often use `zeebe:` and `camunda:` attributes that are **not executed natively**.

## Import outcomes

| Outcome | Meaning |
| :--- | :--- |
| Supported | Deploys and runs with equivalent ArtificialFlow semantics. |
| Rewrite | Deploy may succeed after mapping attributes or replacing elements. |
| Blocked | Parser or runtime rejects; must remodel. |

## Extension mapping (lint targets)

| Camunda / Zeebe | ArtificialFlow | Outcome |
| :--- | :--- | :--- |
| `zeebe:taskDefinition type="…"` on service task | `artificialflow:taskType="…"` | Rewrite |
| `zeebe:assignmentDefinition assignee="…"` | `artificialflow:assignee="…"` | Rewrite |
| `zeebe:assignmentDefinition candidateGroups="…"` | `artificialflow:candidateGroups="…"` | Rewrite |
| `zeebe:calledElement processId="…"` | Call activity `calledElement` / ArtificialFlow call binding | Rewrite |
| `camunda:assignee` / `camunda:candidateGroups` | `artificialflow:assignee` / `artificialflow:candidateGroups` | Rewrite |
| `camunda:decisionRef` on business rule task | `artificialflow:decisionRef` | Rewrite (evaluation requires DMN runtime) |
| Send task / connector outbound | Service task + worker or future HTTP connector | Rewrite / Blocked |
| DMN decision tables as Zeebe decisions | ArtificialFlow DMN deploy (P2) | Blocked until DMN ships |

## Core element outcomes

| Element | Outcome |
| :--- | :--- |
| User / service / script tasks, exclusive/parallel/inclusive/event gateways | Supported (after extension rewrite) |
| Timer / message boundary events | Supported |
| Error boundary events | Supported |
| `<sendTask>` | Blocked — remodel |
| Business rule task with decisionRef | Rewrite metadata; runtime not evaluated until DMN |
| Complex compensation | Partial — validate with tests |

## Lint tool

```bash
# Reports unsupported elements and suggested rewrites for Camunda/Zeebe attributes.
go run ./tools/camunda-bpmn-lint --file path/to/process.bpmn
```

Exit code `0` = no blockers; `1` = blockers present; `2` = usage error.

## Bake-off tip

Keep a small library of remodeled ArtificialFlow BPMN fixtures under `tests/bpmn/camunda-import/` and `examples/golden-demo/` rather than claiming silent Camunda XML compatibility.
