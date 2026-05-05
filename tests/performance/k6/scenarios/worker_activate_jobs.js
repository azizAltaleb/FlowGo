/**
 * worker_activate_jobs.js — Simulate external workers polling for and completing jobs.
 */
import http from "k6/http";
import { check, sleep } from "k6";

const BASE = __ENV.COMMAND_URL || "http://localhost:8080";

export function workerActivateJobs() {
  // Poll for jobs (realistic worker behaviour)
  const activateRes = http.post(
    `${BASE}/jobs/activate`,
    JSON.stringify({
      type: "perf-task",
      worker: `k6-worker-${__VU}`,
      maxJobs: 5,
      lockDurationMs: 30000,
    }),
    {
      headers: { "Content-Type": "application/json" },
      tags: { name: "activate_jobs" },
    }
  );

  check(activateRes, {
    "activate jobs 200": (r) => r.status === 200,
  });

  if (activateRes.status === 200) {
    let jobs = [];
    try {
      const body = JSON.parse(activateRes.body);
      jobs = Array.isArray(body) ? body : body.jobs || [];
    } catch (_) {}

    for (const job of jobs) {
      const jobKey = job.key || job.id;
      if (!jobKey) continue;

      const completeRes = http.post(
        `${BASE}/jobs/${jobKey}/complete`,
        JSON.stringify({ worker: `k6-worker-${__VU}`, variables: { done: true } }),
        {
          headers: { "Content-Type": "application/json" },
          tags: { name: "complete_job" },
        }
      );
      check(completeRes, {
        "complete job 200/204": (r) => r.status === 200 || r.status === 204,
      });
    }
  }

  // Get worker capabilities
  const capsRes = http.get(`${BASE}/jobs/capabilities`, {
    tags: { name: "job_capabilities" },
  });
  check(capsRes, {
    "capabilities 200": (r) => r.status === 200,
  });

  sleep(0.2);
}
