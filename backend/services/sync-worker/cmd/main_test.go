package main

import (
	"strings"
	"testing"
	"time"

	"workflow-engine/backend/services/sync-worker/internal/infrastructure/messaging"
)

func TestParseTopicsEnv_DeduplicatesAndTrims(t *testing.T) {
	topics := parseTopicsEnv("  a.b.c , workflowsa.public.job, a.b.c , , workflowsa.public.variable ")
	if len(topics) != 3 {
		t.Fatalf("expected 3 unique topics, got %d (%v)", len(topics), topics)
	}
	if topics[0] != "a.b.c" || topics[1] != "workflowsa.public.job" || topics[2] != "workflowsa.public.variable" {
		t.Fatalf("unexpected topic ordering/content: %v", topics)
	}
}

func TestEnvBool(t *testing.T) {
	t.Setenv("SYNC_HEALTH_FAIL_ON_STALE", "true")
	if !envBool("SYNC_HEALTH_FAIL_ON_STALE", false) {
		t.Fatalf("expected envBool to parse true")
	}

	t.Setenv("SYNC_HEALTH_FAIL_ON_STALE", "0")
	if envBool("SYNC_HEALTH_FAIL_ON_STALE", true) {
		t.Fatalf("expected envBool to parse false")
	}

	t.Setenv("SYNC_HEALTH_FAIL_ON_STALE", "invalid")
	if !envBool("SYNC_HEALTH_FAIL_ON_STALE", true) {
		t.Fatalf("expected envBool to use default=true for invalid value")
	}
}

func TestEvaluateSyncHealth(t *testing.T) {
	now := time.Now().UTC()

	tests := []struct {
		name             string
		snapshot         messaging.ConsumerStatsSnapshot
		sloSec           int64
		failOnStale      bool
		wantStatus       string
		wantStale        bool
		wantHTTPStatus   int
		wantLagNegative1 bool
	}{
		{
			name:             "starting when no messages yet and slo configured",
			snapshot:         messaging.ConsumerStatsSnapshot{},
			sloSec:           120,
			wantStatus:       "starting",
			wantStale:        false,
			wantHTTPStatus:   200,
			wantLagNegative1: true,
		},
		{
			name: "ok when freshness within slo",
			snapshot: messaging.ConsumerStatsSnapshot{
				LastProcessedAt: now.Add(-30 * time.Second),
			},
			sloSec:         120,
			wantStatus:     "ok",
			wantStale:      false,
			wantHTTPStatus: 200,
		},
		{
			name: "degraded when stale and fail disabled",
			snapshot: messaging.ConsumerStatsSnapshot{
				LastProcessedAt: now.Add(-5 * time.Minute),
			},
			sloSec:         120,
			failOnStale:    false,
			wantStatus:     "degraded",
			wantStale:      true,
			wantHTTPStatus: 200,
		},
		{
			name: "unhealthy when stale and fail enabled",
			snapshot: messaging.ConsumerStatsSnapshot{
				LastProcessedAt: now.Add(-5 * time.Minute),
			},
			sloSec:         120,
			failOnStale:    true,
			wantStatus:     "unhealthy",
			wantStale:      true,
			wantHTTPStatus: 503,
		},
		{
			name: "ok when slo disabled",
			snapshot: messaging.ConsumerStatsSnapshot{
				LastProcessedAt: now.Add(-20 * time.Minute),
			},
			sloSec:         0,
			wantStatus:     "ok",
			wantStale:      false,
			wantHTTPStatus: 200,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			status, lagSec, stale, httpStatus := evaluateSyncHealth(tc.snapshot, tc.sloSec, tc.failOnStale)
			if status != tc.wantStatus {
				t.Fatalf("expected status=%q, got %q", tc.wantStatus, status)
			}
			if stale != tc.wantStale {
				t.Fatalf("expected stale=%v, got %v", tc.wantStale, stale)
			}
			if httpStatus != tc.wantHTTPStatus {
				t.Fatalf("expected httpStatus=%d, got %d", tc.wantHTTPStatus, httpStatus)
			}
			if tc.wantLagNegative1 {
				if lagSec != -1 {
					t.Fatalf("expected lagSec=-1 for starting state, got %d", lagSec)
				}
				return
			}
			if lagSec < 0 {
				t.Fatalf("expected non-negative lagSec, got %d", lagSec)
			}
		})
	}
}

func TestEnvInt_AndParseDurationEnv(t *testing.T) {
	t.Setenv("SYNC_MAX_PROCESS_RETRIES", "7")
	if got := envInt("SYNC_MAX_PROCESS_RETRIES", 1); got != 7 {
		t.Fatalf("expected envInt=7, got %d", got)
	}

	t.Setenv("SYNC_RETRY_BACKOFF", "2s")
	if got := parseDurationEnv("SYNC_RETRY_BACKOFF", time.Second); got != 2*time.Second {
		t.Fatalf("expected duration 2s, got %s", got)
	}

	t.Setenv("SYNC_RETRY_BACKOFF", "invalid")
	if got := parseDurationEnv("SYNC_RETRY_BACKOFF", 3*time.Second); got != 3*time.Second {
		t.Fatalf("expected fallback duration 3s, got %s", got)
	}
}

func TestNormalizeProjectionContract(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "", want: "hybrid"},
		{in: "hybrid", want: "hybrid"},
		{in: "EVENT_FIRST", want: "event-first"},
		{in: "eventfirst", want: "event-first"},
		{in: "events", want: "event-first"},
		{in: "debezium", want: "debezium"},
		{in: "custom", want: "custom"},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			if got := normalizeProjectionContract(tc.in); got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func TestValidateProjectionContract(t *testing.T) {
	tests := []struct {
		name     string
		contract string
		topics   []string
		event    string
		wantErr  string
	}{
		{
			name:     "hybrid accepts event plus debezium",
			contract: "hybrid",
			topics:   []string{"workflow.events.v1", "workflowsa.public.process_instance"},
			event:    "workflow.events.v1",
		},
		{
			name:     "hybrid rejects missing debezium topics",
			contract: "hybrid",
			topics:   []string{"workflow.events.v1"},
			event:    "workflow.events.v1",
			wantErr:  "Debezium table topic",
		},
		{
			name:     "hybrid rejects missing event topic",
			contract: "hybrid",
			topics:   []string{"workflowsa.public.process_instance"},
			event:    "workflow.events.v1",
			wantErr:  "requires event topic",
		},
		{
			name:     "event first accepts only event topic",
			contract: "event-first",
			topics:   []string{"workflow.events.v1"},
			event:    "workflow.events.v1",
		},
		{
			name:     "event first rejects debezium topics",
			contract: "event-first",
			topics:   []string{"workflow.events.v1", "workflowsa.public.variable"},
			event:    "workflow.events.v1",
			wantErr:  "does not allow Debezium",
		},
		{
			name:     "debezium contract accepts debezium only",
			contract: "debezium",
			topics:   []string{"workflowsa.public.process"},
			event:    "workflow.events.v1",
		},
		{
			name:     "debezium contract rejects empty debezium set",
			contract: "debezium",
			topics:   []string{"workflow.events.v1"},
			event:    "workflow.events.v1",
			wantErr:  "requires at least one Debezium",
		},
		{
			name:     "unsupported contract is rejected",
			contract: "something-else",
			topics:   []string{"workflow.events.v1", "workflowsa.public.process"},
			event:    "workflow.events.v1",
			wantErr:  "unsupported SYNC_PROJECTION_CONTRACT",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateProjectionContract(tc.contract, tc.topics, tc.event)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}
