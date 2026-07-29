/**
 * query_read_load.js — Read-heavy load against workflow-query service.
 */
import http from "k6/http";
import { check, sleep } from "k6";
import { authHeaders } from "../lib/auth.js";

const QUERY_BASE = __ENV.QUERY_URL || "http://localhost:8081";

export function queryReadLoad() {
  // List instances with various filters
  const listRes = http.get(
    `${QUERY_BASE}/instances?limit=20&offset=0`,
    { headers: authHeaders(), tags: { name: "list_instances" } }
  );
  check(listRes, {
    "list instances 200": (r) => r.status === 200,
    "list instances has body": (r) => r.body && r.body.length > 0,
  });

  // List workflows
  const workflowsRes = http.get(
    `${QUERY_BASE}/workflows?limit=10`,
    { headers: authHeaders(), tags: { name: "list_workflows" } }
  );
  check(workflowsRes, {
    "list workflows 200": (r) => r.status === 200,
  });

  // Get a non-existent instance (tests 404 path)
  const notFoundRes = http.get(
    `${QUERY_BASE}/instances/999999999999999999`,
    {
      headers: authHeaders(),
      tags: { name: "get_instance_404" },
      responseCallback: http.expectedStatuses(200, 404),
    }
  );
  check(notFoundRes, {
    "404 handled correctly": (r) => r.status === 404 || r.status === 200,
  });

  sleep(0.05);
}
