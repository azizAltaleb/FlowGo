/**
 * cqrs_lag.js — Sample write-to-query visibility lag after starting an instance.
 * Requires COMMAND_URL, QUERY_URL, and AUTH_TOKEN (bearer) when auth is enabled.
 */
import http from "k6/http";
import { check, sleep } from "k6";
import { Trend } from "k6/metrics";
import { authHeaders } from "../lib/auth.js";

const COMMAND_BASE = __ENV.COMMAND_URL || "http://localhost:8080";
const QUERY_BASE = __ENV.QUERY_URL || "http://localhost:8081";
const CONFIGURED_WORKFLOW_ID = __ENV.WORKFLOW_ID || "";
const MAX_WAIT_MS = Number(__ENV.CQRS_LAG_MAX_WAIT_MS || 15000);
const POLL_MS = Number(__ENV.CQRS_LAG_POLL_MS || 500);

const cqrsLagMs = new Trend("cqrs_lag_ms", true);

export function cqrsLag(data) {
  const workflowId = (data && data.workflowId) || CONFIGURED_WORKFLOW_ID;
  if (!workflowId) {
    check(null, { "WORKFLOW_ID provided": () => false });
    return;
  }

  const start = Date.now();
  const startRes = http.post(
    `${COMMAND_BASE}/instances`,
    JSON.stringify({
      workflow_id: workflowId,
      context: { k6: true, startedAt: start },
    }),
    { headers: authHeaders({ "Content-Type": "application/json" }), tags: { name: "start_instance" } },
  );

  const started = check(startRes, {
    "start instance 2xx": (r) => r.status >= 200 && r.status < 300,
  });
  if (!started) {
    return;
  }

  let instanceId = "";
  try {
    instanceId = String(startRes.json("id") || startRes.json("instance_id") || "");
  } catch (_) {
    instanceId = "";
  }
  if (!instanceId) {
    check(null, { "start response has instance id": () => false });
    return;
  }

  const deadline = Date.now() + MAX_WAIT_MS;
  let visible = false;
  while (Date.now() < deadline) {
    const getRes = http.get(`${QUERY_BASE}/instances/${instanceId}`, {
      headers: authHeaders(),
      tags: { name: "query_instance_poll" },
      responseCallback: http.expectedStatuses(200, 404),
    });
    if (getRes.status === 200) {
      visible = true;
      break;
    }
    sleep(POLL_MS / 1000);
  }

  const lag = Date.now() - start;
  cqrsLagMs.add(lag);
  check(null, {
    "instance visible in query before timeout": () => visible,
    "cqrs lag under max wait": () => lag <= MAX_WAIT_MS,
  });
}
