/**
 * Mega BPMN UAT — exercises Supported matrix elements via API + light UI checks.
 * Covers GitHub #40 quickstart path pieces (deploy/start) when AUTH is available.
 *
 * Env:
 *   ARTIFICIALFLOW_BASE_URL (default http://localhost:9100/api)
 *   ARTIFICIALFLOW_TOKEN (required for API path)
 *   ARTIFICIALFLOW_ACTING_USER (default approver)
 *   UAT_MEGA_SCALE (default 3 parallel instances)
 */
import { test, expect } from "@playwright/test";
import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const base = (process.env.ARTIFICIALFLOW_BASE_URL || "http://localhost:9100/api").replace(/\/$/, "");
const token = process.env.ARTIFICIALFLOW_TOKEN || "";
const actingUser = process.env.ARTIFICIALFLOW_ACTING_USER || "approver";
const scale = Math.max(1, Number(process.env.UAT_MEGA_SCALE || 3));

const __dirname = dirname(fileURLToPath(import.meta.url));
const megaBpmn = readFileSync(join(__dirname, "../../fixtures/mega-process.bpmn"), "utf8");

async function api(method: string, path: string, body?: unknown, raw = false) {
  const headers: Record<string, string> = {
    Authorization: `Bearer ${token}`,
    "X-ArtificialFlow-Acting-Username": actingUser,
    "X-ArtificialFlow-Acting-Subject": actingUser,
  };
  if (!raw) headers["Content-Type"] = "application/json";
  else headers["Content-Type"] = "application/xml";
  const res = await fetch(`${base}${path}`, {
    method,
    headers,
    body: body === undefined ? undefined : raw ? String(body) : JSON.stringify(body),
  });
  const text = await res.text();
  let json: any = null;
  try {
    json = text ? JSON.parse(text) : null;
  } catch {
    json = { raw: text };
  }
  return { res, json, text };
}

test.describe("uat-mega-bpmn", () => {
  test.skip(!token, "ARTIFICIALFLOW_TOKEN required");

  test("deploy mega process, run worker+inbox path, scale instances", async ({ page }) => {
    const deploy = await api("POST", "/workflows", megaBpmn, true);
    expect(deploy.res.status, deploy.text).toBeLessThan(300);
    const workflowId = deploy.json.id || deploy.json.workflow_id;
    expect(workflowId).toBeTruthy();

    const instanceIds: string[] = [];
    for (let i = 0; i < scale; i++) {
      const started = await api("POST", "/instances", {
        workflow_id: workflowId,
        context: { orderId: `uat-mega-${Date.now()}-${i}`, skipHuman: false },
      });
      expect(started.res.status, started.text).toBeLessThan(300);
      instanceIds.push(String(started.json.id));
    }

    // Complete service jobs
    for (let attempt = 0; attempt < 60; attempt++) {
      const activated = await api("POST", "/jobs/activate", {
        type: "uat-mega-validate",
        worker: "uat-mega-worker",
        maxJobs: 10,
        timeoutMs: 2000,
      });
      const jobs = activated.json?.jobs || [];
      for (const job of jobs) {
        const key = job.key || job.Key;
        await api("POST", `/jobs/${key}/complete`, {
          worker: "uat-mega-worker",
          variables: { validated: true },
        });
      }
      if (jobs.length === 0 && attempt > 5) break;
    }

    // Inbox claim/complete for first instance
    let matched = false;
    for (let i = 0; i < 30; i++) {
      const inbox = await api("GET", "/inbox");
      const items = Array.isArray(inbox.json) ? inbox.json : inbox.json?.instances || [];
      const hit = items.find((item: any) => String(item.id) === instanceIds[0]);
      if (hit) {
        const tasks = await api("GET", `/inbox/instances/${hit.id}/tasks`);
        const list = tasks.json?.tasks || tasks.json || [];
        const task = list[0];
        expect(task).toBeTruthy();
        const executionId = task.executionId || task.execution_id || task.id;
        if (task.canClaim) {
          await api("POST", `/inbox/instances/${hit.id}/tasks/${executionId}/claim`, {});
        }
        await api("POST", `/inbox/instances/${hit.id}/tasks/${executionId}/complete`, {});
        matched = true;
        break;
      }
      await new Promise((r) => setTimeout(r, 1000));
    }
    expect(matched).toBeTruthy();

    // UI smoke: palette labels (modeler) when UI reachable
    await page.goto("http://localhost:9100/modeler?new=true", { waitUntil: "domcontentloaded" });
    // May redirect to login — only assert when modeler chrome is visible
    const palette = page.locator("text=Service Task");
    if (await palette.count()) {
      await expect(palette.first()).toBeVisible();
      const rail = page.locator("text=Events").locator("xpath=ancestor::div[contains(@class,'overflow')]").first();
      if (await rail.count()) {
        const overflowX = await rail.evaluate((el) => getComputedStyle(el).overflowX);
        expect(overflowX === "hidden" || overflowX === "auto" || overflowX === "scroll").toBeTruthy();
      }
    }
  });

  test("negative: unsupported ad-hoc subprocess reference fails deploy", async () => {
    const bad = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="D" targetNamespace="t">
  <bpmn:process id="bad_adhoc" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:adHocSubProcess id="adhoc"/>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="adhoc"/>
    <bpmn:sequenceFlow id="f2" sourceRef="adhoc" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`;
    const deploy = await api("POST", "/workflows", bad, true);
    expect(deploy.res.status).toBeGreaterThanOrEqual(400);
  });

  test("negative: escalation start is rejected (Tier-2 honesty)", async () => {
    const bad = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="D2" targetNamespace="t">
  <bpmn:process id="bad_escalation" isExecutable="true">
    <bpmn:startEvent id="start"><bpmn:escalationEventDefinition/></bpmn:startEvent>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`;
    const deploy = await api("POST", "/workflows", bad, true);
    expect(deploy.res.status).toBeGreaterThanOrEqual(400);
    expect(deploy.text.toLowerCase()).toContain("escalation");
  });
});
