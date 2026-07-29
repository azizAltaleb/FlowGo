#!/usr/bin/env node
/**
 * Scripted golden-demo bake-off against a running ArtificialFlow gateway.
 * Requires ARTIFICIALFLOW_BASE_URL and ARTIFICIALFLOW_TOKEN (admin or client).
 */
import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const base = (process.env.ARTIFICIALFLOW_BASE_URL || "http://localhost:9100/api").replace(/\/$/, "");
const token = process.env.ARTIFICIALFLOW_TOKEN || "";
const actingUser = process.env.ARTIFICIALFLOW_ACTING_USER || "approver";

if (!token) {
  console.error("Set ARTIFICIALFLOW_TOKEN to a bearer token with deploy/start/job/inbox rights");
  process.exit(2);
}

const headers = {
  Authorization: `Bearer ${token}`,
  "Content-Type": "application/json",
  "X-ArtificialFlow-Acting-Username": actingUser,
  "X-ArtificialFlow-Acting-Subject": actingUser,
};

const __dirname = dirname(fileURLToPath(import.meta.url));
const bpmn = readFileSync(join(__dirname, "order-approval.bpmn"), "utf8");
const decisionJSON = readFileSync(join(__dirname, "invoice_decision.json"), "utf8");

async function req(method, path, body, { raw = false, contentType } = {}) {
  const reqHeaders = { ...headers };
  let payload;
  if (body === undefined) {
    payload = undefined;
  } else if (raw) {
    payload = body;
    reqHeaders["Content-Type"] = contentType || "application/xml";
  } else {
    payload = JSON.stringify(body);
    reqHeaders["Content-Type"] = contentType || "application/json";
  }
  const res = await fetch(`${base}${path}`, {
    method,
    headers: reqHeaders,
    body: payload,
  });
  const text = await res.text();
  let json;
  try {
    json = text ? JSON.parse(text) : null;
  } catch {
    json = { raw: text };
  }
  if (!res.ok) {
    throw new Error(`${method} ${path} → ${res.status}: ${text}`);
  }
  return json;
}

try {
  const decision = await req("POST", "/decisions", {
    decision_id: "invoice_decision",
    name: "Invoice routing",
    resource: decisionJSON,
  });
  console.log("decision deployed", decision.decisionId || decision.decision_id || "invoice_decision");
} catch (err) {
  console.warn("decision deploy skipped/failed:", err.message);
}

// Command API expects a raw BPMN XML body on POST /workflows.
const deployed = await req("POST", "/workflows", bpmn, { raw: true });
const workflowId = deployed.id || deployed.workflow_id || deployed.workflowId;
console.log("deployed", workflowId);

const instance = await req("POST", "/instances", {
  workflow_id: workflowId,
  variables: { orderId: `golden-${Date.now()}` },
});
const instanceId = instance.id || instance.instance_id || instance.instanceId;
console.log("started", instanceId);

// Worker activate/complete loop for golden-validate (this instance only)
const workerName = "golden-demo-worker";
let workerDone = false;
for (let i = 0; i < 30; i++) {
  const activated = await req("POST", "/jobs/activate", {
    type: "golden-validate",
    worker: workerName,
    maxJobs: 5,
    timeoutMs: 2000,
  });
  const jobs = activated.jobs || activated.Jobs || [];
  for (const job of jobs) {
    const key = job.key || job.Key;
    const jobInstance = job.processInstanceKey || job.process_instance_key || job.instanceId;
    if (jobInstance != null && String(jobInstance) !== String(instanceId)) {
      // Leave foreign jobs alone so parallel bake-offs do not steal each other.
      continue;
    }
    await req("POST", `/jobs/${key}/complete`, {
      worker: workerName,
      variables: { validated: true },
    });
    console.log("worker completed job", key);
    workerDone = true;
    break;
  }
  if (workerDone) break;
  await new Promise((r) => setTimeout(r, 1000));
}
if (!workerDone) {
  throw new Error(`no golden-validate job activated for instance ${instanceId}`);
}

// Inbox claim/complete
let match;
for (let i = 0; i < 30; i++) {
  const inbox = await req("GET", "/inbox");
  const items = Array.isArray(inbox) ? inbox : inbox.instances || [];
  match = items.find((item) => String(item.id) === String(instanceId));
  if (match) break;
  await new Promise((r) => setTimeout(r, 1000));
}
if (!match) {
  throw new Error("instance not visible in inbox yet; check CQRS lag / assignment");
}
const tasks = await req("GET", `/inbox/instances/${match.id}/tasks`);
const taskList = tasks.tasks || tasks || [];
const task = taskList.find((t) => t.canClaim || t.canComplete) || taskList[0];
if (!task) {
  throw new Error("no inbox tasks for instance");
}
const executionId = task.executionId || task.execution_id || task.id;
if (task.canClaim) {
  await req("POST", `/inbox/instances/${match.id}/tasks/${executionId}/claim`, {});
}
await req("POST", `/inbox/instances/${match.id}/tasks/${executionId}/complete`, {});
console.log("inbox task completed", executionId);
console.log("golden demo bake-off OK");
