/**
 * run-all.js — k6 entry point that runs all scenarios sequentially.
 * Usage: k6 run --out json=/reports/k6.json tests/performance/k6/run-all.js
 */
import { group } from "k6";
import { workflowThroughput } from "./scenarios/workflow_throughput.js";
import { queryReadLoad } from "./scenarios/query_read_load.js";
import { workerActivateJobs } from "./scenarios/worker_activate_jobs.js";

export const options = {
  scenarios: {
    workflow_throughput: {
      executor: "constant-vus",
      vus: 10,
      duration: "30s",
      exec: "runWorkflowThroughput",
      tags: { scenario: "workflow_throughput" },
    },
    query_read_load: {
      executor: "constant-vus",
      vus: 20,
      duration: "30s",
      startTime: "35s",
      exec: "runQueryReadLoad",
      tags: { scenario: "query_read_load" },
    },
    worker_activate_jobs: {
      executor: "constant-vus",
      vus: 10,
      duration: "20s",
      startTime: "70s",
      exec: "runWorkerActivateJobs",
      tags: { scenario: "worker_activate_jobs" },
    },
  },
  thresholds: {
    http_req_duration: ["p(95)<2000"],
    http_req_failed: ["rate<0.05"],
    checks: ["rate>0.90"],
  },
};

export function runWorkflowThroughput() {
  workflowThroughput();
}

export function runQueryReadLoad() {
  queryReadLoad();
}

export function runWorkerActivateJobs() {
  workerActivateJobs();
}
