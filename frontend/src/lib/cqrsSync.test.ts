import { beforeEach, describe, expect, it, vi } from "vitest";
import { api } from "./api";
import {
  consumePendingInstanceSync,
  consumePendingWorkflowSync,
  pollUntil,
  setPendingInstanceSync,
  setPendingWorkflowSync,
  waitForWorkflowInCatalog,
} from "./cqrsSync";

vi.mock("./api", () => ({
  api: {
    getWorkflows: vi.fn(),
    getInstances: vi.fn(),
  },
}));

describe("cqrs sync helpers", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    window.sessionStorage.clear();
  });

  it("polls until a predicate matches", async () => {
    let attempts = 0;

    const result = await pollUntil(
      async () => {
        attempts += 1;
        return attempts;
      },
      (value) => value >= 2,
      { intervalMs: 1, timeoutMs: 50 },
    );

    expect(result).toBe("synced");
    expect(attempts).toBe(2);
  });

  it("returns timeout when a predicate never matches", async () => {
    const result = await pollUntil(async () => 1, () => false, {
      intervalMs: 1,
      timeoutMs: 3,
    });

    expect(result).toBe("timeout");
  });

  it("matches workflow catalog entries by command id or BPMN identity", async () => {
    vi.mocked(api.getWorkflows)
      .mockResolvedValueOnce([])
      .mockResolvedValueOnce([
        {
          id: "query-key",
          process_definition_id: "Process_Order",
          name: "Order",
          version: 3,
          resource_name: "order.bpmn",
          deployment_id: "deployment-1",
          tenant_id: "default",
          resource_checksum: "checksum",
          bpmn_xml: "<xml />",
          created_at: new Date().toISOString(),
        },
      ]);

    const result = await waitForWorkflowInCatalog(
      "command-key",
      { processDefinitionId: "Process_Order", version: 3 },
      { intervalMs: 1, timeoutMs: 50 },
    );

    expect(result).toBe("synced");
    expect(api.getWorkflows).toHaveBeenCalledTimes(2);
  });

  it("stores and consumes pending cross-page sync hints", () => {
    setPendingWorkflowSync("workflow-1");
    setPendingInstanceSync("instance-1");

    expect(consumePendingWorkflowSync()).toBe("workflow-1");
    expect(consumePendingWorkflowSync()).toBeNull();
    expect(consumePendingInstanceSync()).toBe("instance-1");
    expect(consumePendingInstanceSync()).toBeNull();
  });

  it("writes only canonical pending-sync keys", () => {
    setPendingWorkflowSync("workflow-1");
    setPendingInstanceSync("instance-1");

    expect(window.sessionStorage.getItem("artificialflow.pendingWorkflowSync")).toBe("workflow-1");
    expect(window.sessionStorage.getItem("artificialflow.pendingInstanceSync")).toBe("instance-1");
  });
});
