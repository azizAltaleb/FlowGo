package worker

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"workflow-engine/backend/libs/model"
)

func TestClientActivateJobs(t *testing.T) {
	var captured ActivateJobsRequest
	var capturedProtocolHeader string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/jobs/activate" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		capturedProtocolHeader = r.Header.Get(HeaderWorkerProtocolVersion)

		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("failed to decode activate request: %v", err)
		}

		_ = json.NewEncoder(w).Encode(ActivateJobsResponse{
			Jobs: []model.Job{{
				Key:   101,
				Type:  "payment",
				State: "ACTIVATED",
			}},
		})
	}))
	defer ts.Close()

	client, err := NewClient(ClientConfig{BaseURL: ts.URL, HTTPClient: ts.Client()})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	jobs, err := client.ActivateJobs(context.Background(), ActivateJobsRequest{
		Type:           "payment",
		Worker:         "worker-1",
		MaxJobs:        2,
		TimeoutMs:      1500,
		LockDurationMs: 30000,
	})
	if err != nil {
		t.Fatalf("activate jobs failed: %v", err)
	}

	if captured.Type != "payment" || captured.Worker != "worker-1" || captured.MaxJobs != 2 || captured.TimeoutMs != 1500 || captured.LockDurationMs != 30000 {
		t.Fatalf("unexpected activate payload: %+v", captured)
	}
	if capturedProtocolHeader != WorkerProtocolVersion {
		t.Fatalf("expected protocol header %q, got %q", WorkerProtocolVersion, capturedProtocolHeader)
	}

	if len(jobs) != 1 || jobs[0].Key != 101 {
		t.Fatalf("unexpected jobs response: %+v", jobs)
	}
}

func TestClientExtendJobLock(t *testing.T) {
	called := false
	var captured ExtendJobLockRequest
	var capturedProtocolHeader string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/jobs/42/extend-lock" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		capturedProtocolHeader = r.Header.Get(HeaderWorkerProtocolVersion)
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("failed to decode extend-lock request: %v", err)
		}
		called = true
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Job lock extended"))
	}))
	defer ts.Close()

	client, err := NewClient(ClientConfig{BaseURL: ts.URL, HTTPClient: ts.Client()})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	err = client.ExtendJobLock(context.Background(), 42, ExtendJobLockRequest{
		Worker:         "worker-2",
		LockDurationMs: 45000,
	})
	if err != nil {
		t.Fatalf("extend lock failed: %v", err)
	}

	if !called {
		t.Fatalf("expected extend-lock endpoint to be called")
	}
	if captured.Worker != "worker-2" || captured.LockDurationMs != 45000 {
		t.Fatalf("unexpected extend-lock payload: %+v", captured)
	}
	if capturedProtocolHeader != WorkerProtocolVersion {
		t.Fatalf("expected protocol header %q, got %q", WorkerProtocolVersion, capturedProtocolHeader)
	}
}

func TestClientGetCapabilities(t *testing.T) {
	var capturedProtocolHeader string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/jobs/capabilities" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		capturedProtocolHeader = r.Header.Get(HeaderWorkerProtocolVersion)

		_ = json.NewEncoder(w).Encode(CapabilitiesResponse{
			ProtocolVersion: WorkerProtocolVersion,
			Capabilities: []string{
				"activate",
				"complete",
				"fail",
				"extend-lock",
			},
		})
	}))
	defer ts.Close()

	client, err := NewClient(ClientConfig{BaseURL: ts.URL, HTTPClient: ts.Client()})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	capabilities, err := client.GetCapabilities(context.Background())
	if err != nil {
		t.Fatalf("get capabilities failed: %v", err)
	}

	if capturedProtocolHeader != WorkerProtocolVersion {
		t.Fatalf("expected protocol header %q, got %q", WorkerProtocolVersion, capturedProtocolHeader)
	}
	if capabilities.ProtocolVersion != WorkerProtocolVersion {
		t.Fatalf("expected protocol version %q, got %q", WorkerProtocolVersion, capabilities.ProtocolVersion)
	}
	if len(capabilities.Capabilities) == 0 {
		t.Fatalf("expected non-empty capabilities list")
	}
}
