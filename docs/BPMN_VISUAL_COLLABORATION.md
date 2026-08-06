# BPMN Tier-3 — Visual collaboration

Pools, lanes, data objects/stores, text annotations, groups, associations, and message flows are **visual-only** in ArtificialFlow:

- The modeler palette **Visual** group places Pool, Lane, Data Object, Data Store, Annotation, Group, Message Flow, and Association shapes (`visualArtifact` nodes).
- Export emits BPMN-compliant collaboration artifacts:
  - **Pool** → `bpmn:collaboration` / `bpmn:participant` with `processRef` (not inside the executable process body).
  - **Lane** → `bpmn:laneSet` / `bpmn:lane` with `flowNodeRef` for contained nodes.
  - **Annotation** → `bpmn:textAnnotation` with `bpmn:text` body.
  - **Group / association / message flow** → corresponding BPMN tags under process or collaboration.
- **Lane → candidate groups:** when a user task is nested under a lane node, export sets `artificialflow:candidateGroups` from the lane label (hint only; not multi-party orchestration).
- Runtime executes **variables**, not BPMN data objects/stores. Data shapes are documentation for humans and imports.
- Deploy lint rejects sequence flows wired to visual-only shapes.

See [BPMN_SUPPORT_MATRIX.md](BPMN_SUPPORT_MATRIX.md). Capability catalog: [tests/bpmn/matrix/capabilities.yml](../tests/bpmn/matrix/capabilities.yml).
