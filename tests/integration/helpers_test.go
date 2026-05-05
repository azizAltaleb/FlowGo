//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

var (
	commandBase = envOrDefault("COMMAND_URL", "http://localhost:8080")
	queryBase   = envOrDefault("QUERY_URL", "http://localhost:8081")
	syncHealth  = envOrDefault("SYNC_HEALTH_URL", "http://localhost:8092/health")
	httpClient  = &http.Client{Timeout: 15 * time.Second}
)

func envOrDefault(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

// doJSON sends a JSON request and returns status code + response body.
func doJSON(t *testing.T, method, url string, body any) (int, []byte) {
	t.Helper()
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		reqBody = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(context.Background(), method, url, reqBody)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatalf("do request %s %s: %v", method, url, err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, respBody
}

// waitHTTP polls url until it returns 200 or timeout expires.
func waitHTTP(t *testing.T, url string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := httpClient.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(2 * time.Second)
	}
	t.Fatalf("timed out waiting for %s to be healthy", url)
}

// mustDecode decodes JSON body into dst, failing the test on error.
func mustDecode(t *testing.T, body []byte, dst any) {
	t.Helper()
	if err := json.Unmarshal(body, dst); err != nil {
		t.Fatalf("decode response: %v\nbody: %s", err, string(body))
	}
}

// unmarshalBody decodes JSON without failing — returns error for caller to handle.
func unmarshalBody(body []byte, dst any) error {
	return json.Unmarshal(body, dst)
}

// doBPMN sends a raw BPMN XML body (text/xml) to url via POST.
func doBPMN(t *testing.T, url, bpmnXML string) (int, []byte) {
	t.Helper()
	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		url,
		strings.NewReader(bpmnXML),
	)
	if err != nil {
		t.Fatalf("new BPMN request: %v", err)
	}
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatalf("do BPMN request POST %s: %v", url, err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, respBody
}

// minimalBPMN returns a minimal valid BPMN 2.0 process definition.
func minimalBPMN(processID, name string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL"
  xmlns:bpmndi="http://www.omg.org/spec/BPMN/20100524/DI"
  xmlns:dc="http://www.omg.org/spec/DD/20100524/DC"
  id="Definitions_1" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="%s" name="%s" isExecutable="true">
    <bpmn:startEvent id="start1">
      <bpmn:outgoing>flow1</bpmn:outgoing>
    </bpmn:startEvent>
    <bpmn:userTask id="task1" name="Review">
      <bpmn:incoming>flow1</bpmn:incoming>
      <bpmn:outgoing>flow2</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:endEvent id="end1">
      <bpmn:incoming>flow2</bpmn:incoming>
    </bpmn:endEvent>
    <bpmn:sequenceFlow id="flow1" sourceRef="start1" targetRef="task1"/>
    <bpmn:sequenceFlow id="flow2" sourceRef="task1" targetRef="end1"/>
  </bpmn:process>
</bpmn:definitions>`, processID, name)
}
