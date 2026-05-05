# BPMN Support Matrix

This matrix documents the current supported BPMN feature surface for Workflowsa.

| BPMN feature | Status | Notes |
| :--- | :--- | :--- |
| Process deployment | Supported | BPMN XML can be uploaded through the modeler or API. |
| Start events | Supported | Used to start process instances. |
| End events | Supported | Completes execution paths. |
| User tasks | Supported | Includes assignment metadata and SLA-related properties. |
| Service tasks | Supported | Maps implementation/task type to job execution. |
| Script tasks | Supported | JavaScript execution through the configured script runtime. |
| Exclusive gateways | Supported | Conditional routing. |
| Parallel gateways | Supported | Concurrent execution paths. |
| Inclusive gateways | Supported | Reachability-aware joining and branching. |
| Event-based gateways | Supported | Receive/timer event patterns. |
| Boundary timer events | Supported | Interrupting and non-interrupting behavior. |
| Boundary message events | Supported | Message subscription and correlation flows. |
| Sub-processes | Supported | Nested scopes are supported. |
| Call activities | Supported | Subject to modeled deployment availability. |
| Multi-instance tasks | Supported | Parallel loop metadata is supported. |
| Send tasks | Not supported | Parser rejects unsupported references. |

Run the regression matrix:

```bash
make test-bpmn-matrix
```
