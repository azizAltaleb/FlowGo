# BPMN Tier-3 — Visual collaboration

Pools, lanes, data objects/stores, text annotations, and associations are **visual-only** in ArtificialFlow:

- The modeler palette **Visual** group places Pool, Lane, Data Object, Data Store, and Annotation shapes (`visualArtifact` nodes).
- Deploy/export keeps executable process tokens only; **participants are not placed inside the executable process body**.
- **Lane → candidate groups:** when a user task is nested under a lane node, export sets `artificialflow:candidateGroups` from the lane label (hint only; not multi-party orchestration).
- Runtime executes **variables**, not BPMN data objects/stores. Data shapes are documentation for humans and imports.

See [BPMN_SUPPORT_MATRIX.md](BPMN_SUPPORT_MATRIX.md).
