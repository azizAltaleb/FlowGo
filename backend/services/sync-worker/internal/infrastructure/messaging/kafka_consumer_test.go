package messaging

import (
	"context"
	"errors"
	"testing"
	"time"
	"workflow-engine/backend/services/sync-worker/internal/application"

	"github.com/segmentio/kafka-go"
)

func TestNewKafkaConsumerAppliesConfigDefaults(t *testing.T) {
	consumer := NewKafkaConsumer(Config{
		MaxProcessRetries: -1,
		RetryBackoff:      0,
	}, nil)

	if consumer.cfg.MaxProcessRetries != 0 {
		t.Fatalf("expected max retries defaulted to 0, got %d", consumer.cfg.MaxProcessRetries)
	}
	if consumer.cfg.RetryBackoff <= 0 {
		t.Fatalf("expected retry backoff to be defaulted, got %s", consumer.cfg.RetryBackoff)
	}
}

func TestSnapshotIncludesCountersAndTopicStats(t *testing.T) {
	consumer := NewKafkaConsumer(Config{}, nil)
	now := time.Now().UTC()

	consumer.processedCount.Store(10)
	consumer.successCount.Store(8)
	consumer.failureCount.Store(2)
	consumer.retryCount.Store(3)
	consumer.dlqCount.Store(1)
	consumer.lastProcessed.Store(now.UnixNano())
	consumer.recordTopicOutcome("workflowsa.public.variable", 42, nil)

	snapshot := consumer.Snapshot()
	if snapshot.Processed != 10 || snapshot.Succeeded != 8 || snapshot.Failed != 2 {
		t.Fatalf("unexpected counters in snapshot: %+v", snapshot)
	}
	if snapshot.Retried != 3 || snapshot.DLQPublished != 1 {
		t.Fatalf("unexpected retry/dlq counters in snapshot: %+v", snapshot)
	}
	if snapshot.LastProcessedAt.IsZero() {
		t.Fatalf("expected last processed timestamp to be set")
	}

	topicStats, ok := snapshot.Topics["workflowsa.public.variable"]
	if !ok {
		t.Fatalf("expected topic stats entry")
	}
	if topicStats.LastOffset != 42 {
		t.Fatalf("expected last offset 42, got %d", topicStats.LastOffset)
	}
}

func TestProcessMessageWithRetry_SucceedsAfterRetry(t *testing.T) {
	repo := &flakyRepo{failUpsertAttempts: 1}
	service := application.NewSyncService(repo, "workflowsa")
	consumer := NewKafkaConsumer(Config{MaxProcessRetries: 2, RetryBackoff: time.Millisecond}, service)

	err := consumer.processMessageWithRetry(context.Background(), kafka.Message{
		Topic:     "workflowsa.public.process_instance",
		Partition: 0,
		Offset:    9,
		Value:     []byte(`{"before":null,"after":{"key":1},"op":"c"}`),
	})
	if err != nil {
		t.Fatalf("expected retry to eventually succeed, got error: %v", err)
	}
	if got := consumer.retryCount.Load(); got != 1 {
		t.Fatalf("expected retry count=1, got %d", got)
	}
	if repo.upsertCalls != 2 {
		t.Fatalf("expected 2 upsert attempts, got %d", repo.upsertCalls)
	}
}

func TestProcessMessageWithRetry_FailsAfterMaxAttempts(t *testing.T) {
	repo := &flakyRepo{failUpsertAttempts: 5}
	service := application.NewSyncService(repo, "workflowsa")
	consumer := NewKafkaConsumer(Config{MaxProcessRetries: 1, RetryBackoff: time.Millisecond}, service)

	err := consumer.processMessageWithRetry(context.Background(), kafka.Message{
		Topic:     "workflowsa.public.process_instance",
		Partition: 0,
		Offset:    10,
		Value:     []byte(`{"before":null,"after":{"key":1},"op":"c"}`),
	})
	if err == nil {
		t.Fatalf("expected retries to exhaust with an error")
	}
	if got := consumer.retryCount.Load(); got != 1 {
		t.Fatalf("expected retry count=1, got %d", got)
	}
	if repo.upsertCalls != 2 {
		t.Fatalf("expected 2 upsert attempts with max retries=1, got %d", repo.upsertCalls)
	}
}

type flakyRepo struct {
	failUpsertAttempts int
	upsertCalls        int
}

func (r *flakyRepo) Upsert(_ context.Context, _ string, _ string, _ map[string]any) error {
	r.upsertCalls++
	if r.upsertCalls <= r.failUpsertAttempts {
		return errors.New("injected upsert failure")
	}
	return nil
}

func (r *flakyRepo) Delete(_ context.Context, _ string, _ string) error {
	return nil
}

func (r *flakyRepo) UpdateWithScript(_ context.Context, _ string, _ string, _ string, _ map[string]any) error {
	return nil
}
