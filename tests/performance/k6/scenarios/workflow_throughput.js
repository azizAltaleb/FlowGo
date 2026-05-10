/**
 * workflow_throughput.js — Deploy + start + complete workflow lifecycle under load.
 */
import http from "k6/http";
import { check, sleep } from "k6";

const BASE = __ENV.COMMAND_URL || "http://localhost:8080";

const MINIMAL_BPMN = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL"
  xmlns:goflow="http://goflow.com/schema/1.0/bpmn"
  id="Definitions_perf" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="perf-workflow" name="Perf Test" isExecutable="true">
    <bpmn:startEvent id="start1">
      <bpmn:outgoing>flow1</bpmn:outgoing>
    </bpmn:startEvent>
    <bpmn:serviceTask id="task1" name="Do Work" goflow:taskType="perf-task">
      <bpmn:incoming>flow1</bpmn:incoming>
      <bpmn:outgoing>flow2</bpmn:outgoing>
    </bpmn:serviceTask>
    <bpmn:endEvent id="end1">
      <bpmn:incoming>flow2</bpmn:incoming>
    </bpmn:endEvent>
    <bpmn:sequenceFlow id="flow1" sourceRef="start1" targetRef="task1"/>
    <bpmn:sequenceFlow id="flow2" sourceRef="task1" targetRef="end1"/>
  </bpmn:process>
</bpmn:definitions>`;

let deployedWorkflowId = null;

export function setup() {
  const res = http.post(
    `${BASE}/workflows`,
    MINIMAL_BPMN,
    { headers: { "Content-Type": "text/xml; charset=utf-8" } }
  );
  if (res.status === 200) {
    const body = JSON.parse(res.body);
    return { workflowId: body.id };
  }
  return { workflowId: null };
}

export function workflowThroughput(data) {
  const workflowId = (data && data.workflowId) || deployedWorkflowId;

  if (!workflowId) {
    const deployRes = http.post(
      `${BASE}/workflows`,
      MINIMAL_BPMN,
      { headers: { "Content-Type": "text/xml; charset=utf-8" } }
    );
    if (deployRes.status !== 200) {
      return;
    }
    deployedWorkflowId = JSON.parse(deployRes.body).id;
    return;
  }

  // Start instance — uses workflow_id + context
  const startRes = http.post(
    `${BASE}/instances`,
    JSON.stringify({
      workflow_id: workflowId,
      context: { load_test: true, vu: __VU, iter: __ITER },
    }),
    { headers: { "Content-Type": "application/json" } }
  );

  check(startRes, {
    "instance started (200)": (r) => r.status === 200,
    "instance has id": (r) => {
      try {
        return JSON.parse(r.body).id != null;
      } catch (_) {
        return false;
      }
    },
  });

  if (startRes.status === 200) {
    const instance = JSON.parse(startRes.body);

    // Activate the service task job
    const activateRes = http.post(
      `${BASE}/jobs/activate`,
      JSON.stringify({
        type: "perf-task",
        worker: `k6-worker-${__VU}`,
        maxJobs: 1,
        lockDurationMs: 30000,
      }),
      { headers: { "Content-Type": "application/json" } }
    );

    if (activateRes.status === 200) {
      const jobs = JSON.parse(activateRes.body);
      const jobList = Array.isArray(jobs) ? jobs : (jobs.jobs || []);
      if (jobList.length > 0) {
        const jobKey = jobList[0].key || jobList[0].id;
        if (jobKey) {
          const completeRes = http.post(
            `${BASE}/jobs/${jobKey}/complete`,
            JSON.stringify({ variables: { result: "ok" } }),
            { headers: { "Content-Type": "application/json" } }
          );
          check(completeRes, {
            "job completed (200/204)": (r) => r.status === 200 || r.status === 204,
          });
        }
      }
    }
  }

  sleep(0.1);
}
