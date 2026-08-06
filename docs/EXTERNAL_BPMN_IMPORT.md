# External BPMN Import Guide

ArtificialFlow accepts BPMN 2.0 XML with `artificialflow:` extensions. Diagrams exported from other modelers often use third-party extension prefixes (commonly `zeebe:` and related legacy attribute forms) that are **not executed natively**.

## Import outcomes

| Outcome | Meaning |
| :--- | :--- |
| Supported | Deploys and runs with equivalent ArtificialFlow semantics. |
| Rewrite | Deploy may succeed after mapping attributes or replacing elements. |
| Blocked | Parser or runtime rejects; must remodel. |

## Extension mapping (lint targets)

| External / legacy form | ArtificialFlow | Outcome |
| :--- | :--- | :--- |
| `zeebe:taskDefinition type="…"` on service task | `artificialflow:taskType="…"` | Rewrite |
| `zeebe:assignmentDefinition assignee="…"` | `artificialflow:assignee="…"` | Rewrite |
| `zeebe:assignmentDefinition candidateGroups="…"` | `artificialflow:candidateGroups="…"` | Rewrite |
| `zeebe:calledElement processId="…"` | Call activity `calledElement` / ArtificialFlow call binding | Rewrite |
| Legacy Modeler `assignee` / `candidateGroups` attributes | `artificialflow:assignee` / `artificialflow:candidateGroups` | Rewrite |
| Legacy Modeler `decisionRef` on business rule task | `artificialflow:decisionRef` | Rewrite (DMN runtime evaluates JSON decision tables) |
| Send task / connector outbound | External job / connector (`artificialflow:taskType`) | Supported |
| External decision-table bindings | ArtificialFlow DMN deploy | Supported with DMN runtime (not blocked) |

## Core element outcomes

| Element | Outcome |
| :--- | :--- |
| User / service / script tasks, exclusive/parallel/inclusive/event gateways | Supported (after extension rewrite) |
| Timer / message boundary events | Supported |
| Error boundary events | Supported |
| `<sendTask>` | Supported (external job / connector; default type `io.artificialflow.connector.send`) |
| Business rule task with decisionRef | Supported with DMN runtime (JSON decision tables; see [DMN.md](DMN.md)) |
| Complex compensation | Partial — validate with tests |

## Lint tool

```bash
# Reports unsupported elements and suggested rewrites for external/legacy attributes.
go run ./tools/external-bpmn-lint --file path/to/process.bpmn
```

Exit code `0` = no blockers; `1` = blockers present; `2` = usage error.

## Bake-off tip

Keep a small library of remodeled ArtificialFlow BPMN fixtures under `tests/bpmn/external-import/` and `examples/golden-demo/` rather than claiming silent third-party XML compatibility.
