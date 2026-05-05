//go:build integration

package integration

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// TestStackHealthy ensures all services are up before running further tests.
func TestStackHealthy(t *testing.T) {
	waitHTTP(t, commandBase+"/health", 30*time.Second)
	waitHTTP(t, queryBase+"/health", 30*time.Second)
	waitHTTP(t, syncHealth, 30*time.Second)
}

// TestDeployWorkflow_Success deploys a BPMN workflow and expects 200.
func TestDeployWorkflow_Success(t *testing.T) {
	bpmn := minimalBPMN("proc-integration-deploy", "Integration Deploy Test")
	status, body := doBPMN(t, commandBase+"/workflows", bpmn)
	if status != http.StatusOK {
		t.Fatalf("expected 200 deploying workflow, got %d: %s", status, string(body))
	}
	var resp map[string]any
	mustDecode(t, body, &resp)
	if resp["id"] == nil || resp["id"] == "" {
		t.Fatalf("expected workflow id in response, got: %s", string(body))
	}
}

// TestDeployWorkflow_InvalidBPMN expects 4xx or 500 for malformed XML.
func TestDeployWorkflow_InvalidBPMN(t *testing.T) {
	status, _ := doBPMN(t, commandBase+"/workflows", "<not-valid-bpmn>")
	if status < 400 {
		t.Fatalf("expected error status for invalid BPMN, got %d", status)
	}
}

// TestStartInstance_Success deploys then starts an instance.
func TestStartInstance_Success(t *testing.T) {
	processID := fmt.Sprintf("proc-start-%d", time.Now().UnixNano())
	bpmn := minimalBPMN(processID, "Start Instance Test")

	// Deploy — raw BPMN XML body
	status, body := doBPMN(t, commandBase+"/workflows", bpmn)
	if status != http.StatusOK {
		t.Fatalf("deploy failed: %d %s", status, string(body))
	}
	var deployed map[string]any
	mustDecode(t, body, &deployed)
	workflowID := fmt.Sprintf("%v", deployed["id"])

	// Start instance — uses workflow_id + context
	status, body = doJSON(t, http.MethodPost, commandBase+"/instances", map[string]any{
		"workflow_id": workflowID,
		"context":     map[string]any{"env": "integration-test"},
	})
	if status != http.StatusOK {
		t.Fatalf("start instance failed: %d %s", status, string(body))
	}
	var inst map[string]any
	mustDecode(t, body, &inst)
	if inst["id"] == nil {
		t.Fatalf("expected instance id in response, got: %s", string(body))
	}
}

// TestListWorkflows_ReturnsArray ensures GET /workflows returns a JSON array.
func TestListWorkflows_ReturnsArray(t *testing.T) {
	status, body := doJSON(t, http.MethodGet, commandBase+"/workflows", nil)
	if status != http.StatusOK {
		t.Fatalf("expected 200 from GET /workflows, got %d", status)
	}
	// Response is a plain JSON array
	var arr []any
	if err := unmarshalBody(body, &arr); err != nil {
		t.Fatalf("expected JSON array from GET /workflows, got: %s", string(body))
	}
}

// TestGetWorkflow_NotFound expects 404 for unknown id.
func TestGetWorkflow_NotFound(t *testing.T) {
	status, _ := doJSON(t, http.MethodGet, commandBase+"/workflows/does-not-exist-xyz-404", nil)
	if status != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown workflow, got %d", status)
	}
}

// TestActivateJobs_EmptyWhenNoInstances expects 200 with empty jobs list.
func TestActivateJobs_EmptyWhenNoInstances(t *testing.T) {
	status, body := doJSON(t, http.MethodPost, commandBase+"/jobs/activate", map[string]any{
		"type":           "nonexistent-type-xyz",
		"worker":         "integration-test-worker",
		"maxJobs":        5,
		"lockDurationMs": 30000,
	})
	if status != http.StatusOK {
		t.Fatalf("expected 200 from /jobs/activate, got %d: %s", status, string(body))
	}
	// Response shape: {"jobs": [...]}
	var resp map[string]any
	mustDecode(t, body, &resp)
	if _, ok := resp["jobs"]; !ok {
		t.Fatalf("expected 'jobs' key in response, got: %s", string(body))
	}
}

// TestMetricsEndpoint_PrometheusFormat checks /metrics returns Prometheus text format.
func TestMetricsEndpoint_PrometheusFormat(t *testing.T) {
	resp, err := httpClient.Get(commandBase + "/metrics")
	if err != nil {
		t.Fatalf("GET /metrics: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from /metrics, got %d", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/plain") {
		t.Fatalf("expected Prometheus text content-type, got: %q", ct)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "go_gc_duration_seconds") && !strings.Contains(string(body), "http_requests_total") {
		t.Fatalf("expected Prometheus metrics in body, got: %s", string(body)[:200])
	}
}

// TestQueryService_Health checks workflow-query /health returns 200.
func TestQueryService_Health(t *testing.T) {
	status, body := doJSON(t, http.MethodGet, queryBase+"/health", nil)
	if status != http.StatusOK {
		t.Fatalf("expected 200 from query /health, got %d: %s", status, string(body))
	}
}

// TestQueryMetrics_PrometheusFormat checks workflow-query /metrics endpoint.
func TestQueryMetrics_PrometheusFormat(t *testing.T) {
	resp, err := httpClient.Get(queryBase + "/metrics")
	if err != nil {
		t.Fatalf("GET query /metrics: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from query /metrics, got %d", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/plain") {
		t.Fatalf("expected Prometheus text content-type from query, got: %q", ct)
	}
}

// TestAuthMiddleware_Returns401WithoutToken when auth is enforced.
func TestAuthMiddleware_Returns401WithoutToken(t *testing.T) {
	// Both 200 (auth disabled) and 401 (auth enabled) are acceptable.
	resp, err := httpClient.Get(commandBase + "/identity/me")
	if err != nil {
		t.Fatalf("GET /identity/me: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 200 or 401 from /identity/me, got %d", resp.StatusCode)
	}
}
