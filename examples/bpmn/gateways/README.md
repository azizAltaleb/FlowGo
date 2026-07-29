# Gateway compatibility examples

BPMN fixtures for ArtificialFlow gateway behavior (issue #37). Align with [docs/BPMN_SUPPORT_MATRIX.md](../../../docs/BPMN_SUPPORT_MATRIX.md).

| File | Gateway | Notes |
| :--- | :--- | :--- |
| `exclusive-default.bpmn` | Exclusive (XOR) | Conditional flows + default |
| `parallel-fork-join.bpmn` | Parallel (AND) | Fork then join |
| `inclusive-or.bpmn` | Inclusive (OR) | Multi-path branch/join |
| `event-based.bpmn` | Event-based | Message vs timer race |

Deploy via API or Modeler, then start with variables as noted in each file’s documentation annotations.
