import type { WorkflowDefinition } from "@/lib/api";

export type ProcessRef = {
  definitionKey: string;
  processId: string;
  processName: string;
};

/** Build a lookup from definition key (workflow.id / instance.workflow_id) to process identity. */
export function buildProcessLookup(workflows: WorkflowDefinition[]): Map<string, ProcessRef> {
  const map = new Map<string, ProcessRef>();
  for (const wf of workflows) {
    const definitionKey = String(wf.id);
    map.set(definitionKey, {
      definitionKey,
      processId: wf.process_definition_id || definitionKey,
      processName: wf.name || wf.process_definition_id || definitionKey,
    });
  }
  return map;
}

export function resolveProcessRef(
  lookup: Map<string, ProcessRef>,
  workflowId: string | undefined | null,
): ProcessRef {
  const key = String(workflowId || "");
  if (!key) {
    return { definitionKey: "—", processId: "—", processName: "Unknown process" };
  }
  return (
    lookup.get(key) || {
      definitionKey: key,
      processId: key,
      processName: `Process ${key}`,
    }
  );
}
