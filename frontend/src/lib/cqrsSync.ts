import { api, type WorkflowDefinition, type WorkflowInstance } from "./api";

export type CqrsSyncResult = "synced" | "timeout";

export const CQRS_SYNC_INTERVAL_MS = 500;
export const CQRS_SYNC_TIMEOUT_MS = 15_000;

type PollOptions = {
  intervalMs?: number;
  timeoutMs?: number;
};

type WorkflowMatch = {
  processDefinitionId?: string;
  version?: number;
};

const pendingWorkflowKey = "artificialflow.pendingWorkflowSync";
const pendingInstanceKey = "artificialflow.pendingInstanceSync";

const sleep = (ms: number) => new Promise((resolve) => window.setTimeout(resolve, ms));

const sameID = (left: unknown, right: string) => String(left) === String(right);

export async function pollUntil<T>(
  fetchSnapshot: () => Promise<T>,
  predicate: (snapshot: T) => boolean,
  options: PollOptions = {},
): Promise<CqrsSyncResult> {
  const intervalMs = options.intervalMs ?? CQRS_SYNC_INTERVAL_MS;
  const timeoutMs = options.timeoutMs ?? CQRS_SYNC_TIMEOUT_MS;
  const deadline = Date.now() + timeoutMs;

  while (true) {
    const snapshot = await fetchSnapshot();
    if (predicate(snapshot)) {
      return "synced";
    }
    if (Date.now() >= deadline) {
      return "timeout";
    }
    await sleep(Math.min(intervalMs, Math.max(0, deadline - Date.now())));
  }
}

const workflowMatches = (
  workflow: WorkflowDefinition,
  workflowId: string,
  match: WorkflowMatch = {},
) => {
  if (sameID(workflow.id, workflowId)) {
    return true;
  }
  if (!match.processDefinitionId) {
    return false;
  }
  return (
    workflow.process_definition_id === match.processDefinitionId &&
    (match.version === undefined || workflow.version === match.version)
  );
};

export const waitForWorkflowInCatalog = (
  workflowId: string,
  match?: WorkflowMatch,
  options?: PollOptions,
) =>
  pollUntil(
    () => api.getWorkflows(),
    (workflows) => workflows.some((workflow) => workflowMatches(workflow, workflowId, match)),
    options,
  );

export const waitForWorkflowRemovedFromCatalog = (workflowId: string, options?: PollOptions) =>
  pollUntil(
    () => api.getWorkflows(),
    (workflows) => workflows.every((workflow) => !sameID(workflow.id, workflowId)),
    options,
  );

export const waitForInstanceInList = (instanceId: string, options?: PollOptions) =>
  pollUntil(
    () => api.getInstances(),
    (instances) => instances.some((instance: WorkflowInstance) => sameID(instance.id, instanceId)),
    options,
  );

export const waitForInstanceRemovedFromList = (instanceId: string, options?: PollOptions) =>
  pollUntil(
    () => api.getInstances(),
    (instances) => instances.every((instance: WorkflowInstance) => !sameID(instance.id, instanceId)),
    options,
  );

const setPendingSync = (key: string, id: string) => {
  if (typeof window === "undefined") return;
  window.sessionStorage.setItem(key, id);
};

const consumePendingSync = (key: string): string | null => {
  if (typeof window === "undefined") return null;
  const id = window.sessionStorage.getItem(key);
  if (id) {
    window.sessionStorage.removeItem(key);
  }
  return id;
};

export const setPendingWorkflowSync = (id: string) => setPendingSync(pendingWorkflowKey, id);
export const consumePendingWorkflowSync = () => consumePendingSync(pendingWorkflowKey);
export const setPendingInstanceSync = (id: string) => setPendingSync(pendingInstanceKey, id);
export const consumePendingInstanceSync = () => consumePendingSync(pendingInstanceKey);
