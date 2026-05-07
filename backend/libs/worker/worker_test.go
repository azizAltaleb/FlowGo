package worker

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/azizAltaleb/goflow/backend/libs/model"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestWorkerRunOnceCompletesJob(t *testing.T) {
	var completeReq CompleteJobRequest
	completeCalled := false

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/jobs/activate":
			_ = json.NewEncoder(w).Encode(ActivateJobsResponse{Jobs: []model.Job{{
				Key:   55,
				Type:  "payment",
				State: "ACTIVATED",
			}}})
		case "/jobs/55/complete":
			if err := json.NewDecoder(r.Body).Decode(&completeReq); err != nil {
				t.Fatalf("failed to decode complete request: %v", err)
			}
			completeCalled = true
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer ts.Close()

	client, err := NewClient(ClientConfig{BaseURL: ts.URL, HTTPClient: ts.Client()})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	w, err := NewWorker(client, WorkerConfig{
		JobType:    "payment",
		WorkerName: "worker-a",
		Handler: func(ctx context.Context, job model.Job) (map[string]any, error) {
			if job.Key != 55 {
				t.Fatalf("unexpected job key: %d", job.Key)
			}
			return map[string]any{"approved": true}, nil
		},
	})
	if err != nil {
		t.Fatalf("failed to create worker: %v", err)
	}

	if err := w.RunOnce(context.Background()); err != nil {
		t.Fatalf("run once failed: %v", err)
	}

	if !completeCalled {
		t.Fatalf("expected complete endpoint to be called")
	}
	if completeReq.Worker != "worker-a" {
		t.Fatalf("unexpected worker on completion: %s", completeReq.Worker)
	}
	if completeReq.Variables["approved"] != true {
		t.Fatalf("unexpected completion variables: %+v", completeReq.Variables)
	}
}

func TestWorkerRunOnceFailsJob(t *testing.T) {
	var failReq FailJobRequest
	failCalled := false

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/jobs/activate":
			_ = json.NewEncoder(w).Encode(ActivateJobsResponse{Jobs: []model.Job{{
				Key:   99,
				Type:  "email",
				State: "ACTIVATED",
			}}})
		case "/jobs/99/fail":
			if err := json.NewDecoder(r.Body).Decode(&failReq); err != nil {
				t.Fatalf("failed to decode fail request: %v", err)
			}
			failCalled = true
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer ts.Close()

	client, err := NewClient(ClientConfig{BaseURL: ts.URL, HTTPClient: ts.Client()})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	w, err := NewWorker(client, WorkerConfig{
		JobType:    "email",
		WorkerName: "worker-b",
		Handler: func(ctx context.Context, job model.Job) (map[string]any, error) {
			return nil, errors.New("handler failed")
		},
	})
	if err != nil {
		t.Fatalf("failed to create worker: %v", err)
	}

	if err := w.RunOnce(context.Background()); err != nil {
		t.Fatalf("run once failed: %v", err)
	}

	if !failCalled {
		t.Fatalf("expected fail endpoint to be called")
	}
	if failReq.Worker != "worker-b" {
		t.Fatalf("unexpected worker on fail request: %s", failReq.Worker)
	}
	if failReq.ErrorMessage != "handler failed" {
		t.Fatalf("unexpected fail error message: %s", failReq.ErrorMessage)
	}
}

func TestWorkerRunOnceRenewsLockDuringHandler(t *testing.T) {
	var completeReq CompleteJobRequest
	completeCalled := false
	var renewCount atomic.Int64

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/jobs/activate":
			_ = json.NewEncoder(w).Encode(ActivateJobsResponse{Jobs: []model.Job{{
				Key:   77,
				Type:  "slow-task",
				State: "ACTIVATED",
			}}})
		case "/jobs/77/extend-lock":
			var req ExtendJobLockRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("failed to decode extend-lock request: %v", err)
			}
			if req.Worker != "worker-renew" {
				t.Fatalf("unexpected worker in extend-lock: %s", req.Worker)
			}
			if req.LockDurationMs != 100 {
				t.Fatalf("unexpected lock duration in extend-lock: %d", req.LockDurationMs)
			}
			renewCount.Add(1)
			w.WriteHeader(http.StatusOK)
		case "/jobs/77/complete":
			if err := json.NewDecoder(r.Body).Decode(&completeReq); err != nil {
				t.Fatalf("failed to decode complete request: %v", err)
			}
			completeCalled = true
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer ts.Close()

	client, err := NewClient(ClientConfig{BaseURL: ts.URL, HTTPClient: ts.Client()})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	w, err := NewWorker(client, WorkerConfig{
		JobType:           "slow-task",
		WorkerName:        "worker-renew",
		LockDuration:      100 * time.Millisecond,
		LockRenewInterval: 20 * time.Millisecond,
		Handler: func(ctx context.Context, job model.Job) (map[string]any, error) {
			timer := time.NewTimer(75 * time.Millisecond)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-timer.C:
			}
			return map[string]any{"ok": true}, nil
		},
	})
	if err != nil {
		t.Fatalf("failed to create worker: %v", err)
	}

	if err := w.RunOnce(context.Background()); err != nil {
		t.Fatalf("run once failed: %v", err)
	}

	if !completeCalled {
		t.Fatalf("expected complete endpoint to be called")
	}
	if completeReq.Worker != "worker-renew" {
		t.Fatalf("unexpected worker on completion: %s", completeReq.Worker)
	}
	if renewCount.Load() < 1 {
		t.Fatalf("expected at least one lock renewal call")
	}
}

func TestNewWorkerRejectsNegativeLockRenewInterval(t *testing.T) {
	client, err := NewClient(ClientConfig{BaseURL: "http://example.com", HTTPClient: &http.Client{Timeout: time.Second}})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	_, err = NewWorker(client, WorkerConfig{
		WorkerName:        "worker-neg",
		LockRenewInterval: -1 * time.Second,
		Handler: func(ctx context.Context, job model.Job) (map[string]any, error) {
			return nil, nil
		},
	})
	if err == nil {
		t.Fatalf("expected error for negative lock renew interval")
	}
}

func TestWorkerRunOnceReturnsErrorWhenLockRenewFails(t *testing.T) {
	completeCalled := false
	failCalled := false

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/jobs/activate":
			_ = json.NewEncoder(w).Encode(ActivateJobsResponse{Jobs: []model.Job{{
				Key:   88,
				Type:  "slow-task",
				State: "ACTIVATED",
			}}})
		case "/jobs/88/extend-lock":
			http.Error(w, "lock lost", http.StatusBadRequest)
		case "/jobs/88/complete":
			completeCalled = true
			w.WriteHeader(http.StatusOK)
		case "/jobs/88/fail":
			failCalled = true
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer ts.Close()

	client, err := NewClient(ClientConfig{BaseURL: ts.URL, HTTPClient: ts.Client()})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	w, err := NewWorker(client, WorkerConfig{
		JobType:           "slow-task",
		WorkerName:        "worker-renew-fail",
		LockDuration:      100 * time.Millisecond,
		LockRenewInterval: 20 * time.Millisecond,
		Handler: func(ctx context.Context, job model.Job) (map[string]any, error) {
			timer := time.NewTimer(200 * time.Millisecond)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-timer.C:
				return map[string]any{"ok": true}, nil
			}
		},
	})
	if err != nil {
		t.Fatalf("failed to create worker: %v", err)
	}

	err = w.RunOnce(context.Background())
	if err == nil {
		t.Fatalf("expected run once to fail when lock renew fails")
	}
	if !strings.Contains(err.Error(), "failed to extend lock") {
		t.Fatalf("unexpected error: %v", err)
	}
	if completeCalled {
		t.Fatalf("did not expect complete endpoint to be called")
	}
	if failCalled {
		t.Fatalf("did not expect fail endpoint to be called when lock renewal is lost")
	}
}
