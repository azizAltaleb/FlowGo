import { afterEach, describe, expect, it, vi } from "vitest";
import { api } from "./api";

describe("api.deployWorkflow", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("includes backend validation details from failed deploy responses", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response("bpmn validation failed: missing process", {
          status: 400,
          statusText: "Bad Request",
        }),
      ),
    );

    await expect(api.deployWorkflow("<invalid />")).rejects.toThrow(
      "bpmn validation failed: missing process",
    );
  });
});

describe("api.getWorkflows", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("reads the query-side workflow catalog", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      Response.json({
        workflows: [
          {
            id: "workflow-1",
            process_definition_id: "Process_Order",
            name: "Order",
            version: 1,
            resource_name: "order.bpmn",
            deployment_id: "deployment-1",
            tenant_id: "default",
            resource_checksum: "checksum",
            bpmn_xml: "<xml />",
            created_at: new Date().toISOString(),
          },
        ],
        total: 1,
      }),
    );
    vi.stubGlobal("fetch", fetchMock);

    const workflows = await api.getWorkflows();

    expect(fetchMock).toHaveBeenCalledWith(
      "/api/query/workflows?page=1&pageSize=100",
      expect.objectContaining({
        headers: expect.objectContaining({
          "X-Correlation-ID": expect.any(String),
        }),
      }),
    );
    expect(workflows).toHaveLength(1);
    expect(workflows[0].process_definition_id).toBe("Process_Order");
  });
});

describe("SDK client key APIs", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("creates a client by uploading only public-key credential material", async () => {
    const fetchMock = vi.fn().mockResolvedValue(Response.json({ key_id: "key-1" }, { status: 201 }));
    vi.stubGlobal("fetch", fetchMock);

    await api.createIdentityManagementClientKey({
      name: "Orders Worker",
      public_key: "-----BEGIN PUBLIC KEY-----\nPUBLIC\n-----END PUBLIC KEY-----\n",
      key_expires_at: "2027-01-01T00:00:00Z",
    });

    const [, init] = fetchMock.mock.calls[0];
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/identity/management/clients",
      expect.objectContaining({ method: "POST" }),
    );
    expect(JSON.parse(String(init.body))).toEqual({
      name: "Orders Worker",
      public_key: "-----BEGIN PUBLIC KEY-----\nPUBLIC\n-----END PUBLIC KEY-----\n",
      key_expires_at: "2027-01-01T00:00:00Z",
    });
    expect(String(init.body)).not.toContain("PRIVATE KEY");
  });

  it("adds and revokes overlapping public keys on the key endpoints", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(Response.json({ key_id: "key-2" }, { status: 201 }))
      .mockResolvedValueOnce(new Response(null, { status: 204 }));
    vi.stubGlobal("fetch", fetchMock);

    await api.addIdentityManagementClientKey("client / 1", {
      public_key: "PUBLIC",
      key_expires_at: "2027-01-01T00:00:00Z",
    });
    await api.revokeIdentityManagementClientKey("client / 1", "key / 2");

    expect(fetchMock.mock.calls[0][0]).toBe("/api/identity/management/clients/client%20%2F%201/keys");
    expect(fetchMock.mock.calls[1][0]).toBe("/api/identity/management/clients/client%20%2F%201/keys/key%20%2F%202");
    expect(fetchMock.mock.calls[1][1]).toEqual(expect.objectContaining({ method: "DELETE" }));
  });
});
