package application

import (
	"context"
	"errors"
	pb "github.com/azizAltaleb/flowgo/backend/api/v1/go"
	"github.com/azizAltaleb/flowgo/backend/libs/model"
	"github.com/azizAltaleb/flowgo/backend/services/workflow-command/internal/domain/repository"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"
)

type txOnlyRepo struct {
	repository.Repository
	txCalls     int
	outbox      map[string]model.OutboxMessage
	idempotency map[string]model.IdempotencyRecord
}

func (r *txOnlyRepo) WithTx(_ context.Context, fn func(txRepo repository.Repository) error) error {
	r.txCalls++
	return fn(r)
}

func (r *txOnlyRepo) CreateOutboxMessage(_ context.Context, message *model.OutboxMessage) error {
	if r.outbox == nil {
		r.outbox = make(map[string]model.OutboxMessage)
	}
	r.outbox[message.ID] = *message
	return nil
}

func (r *txOnlyRepo) ListPendingOutboxMessages(_ context.Context, now time.Time, limit int) ([]model.OutboxMessage, error) {
	if limit <= 0 {
		limit = 100
	}
	out := make([]model.OutboxMessage, 0, limit)
	for _, msg := range r.outbox {
		if msg.Status != "PENDING" {
			continue
		}
		if msg.NextAttempt != nil && msg.NextAttempt.After(now) {
			continue
		}
		out = append(out, msg)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (r *txOnlyRepo) ClaimOutboxMessage(_ context.Context, id string, claimedAt time.Time) (bool, error) {
	msg, ok := r.outbox[id]
	if !ok {
		return false, nil
	}
	if msg.Status != "PENDING" {
		return false, nil
	}
	if msg.NextAttempt != nil && msg.NextAttempt.After(claimedAt) {
		return false, nil
	}
	msg.Status = "PROCESSING"
	msg.Attempts++
	msg.LastError = ""
	msg.NextAttempt = nil
	r.outbox[id] = msg
	return true, nil
}

func (r *txOnlyRepo) MarkOutboxMessagePublishFailed(_ context.Context, id, lastError string, nextAttempt time.Time) error {
	msg, ok := r.outbox[id]
	if !ok {
		return nil
	}
	msg.Status = "PENDING"
	msg.LastError = lastError
	msg.NextAttempt = &nextAttempt
	r.outbox[id] = msg
	return nil
}

func (r *txOnlyRepo) MarkOutboxMessageTerminalFailed(_ context.Context, id, lastError string, failedAt time.Time) error {
	msg, ok := r.outbox[id]
	if !ok {
		return nil
	}
	msg.Status = "FAILED"
	msg.LastError = lastError
	msg.NextAttempt = nil
	msg.PublishedAt = &failedAt
	r.outbox[id] = msg
	return nil
}

func (r *txOnlyRepo) MarkOutboxMessagePublished(_ context.Context, id string, publishedAt time.Time) error {
	msg, ok := r.outbox[id]
	if !ok {
		return nil
	}
	msg.Status = "PUBLISHED"
	msg.PublishedAt = &publishedAt
	r.outbox[id] = msg
	return nil
}

func (r *txOnlyRepo) CountPendingOutboxMessages(_ context.Context, now time.Time) (int64, error) {
	var count int64
	for _, msg := range r.outbox {
		if msg.Status != "PENDING" {
			continue
		}
		if msg.NextAttempt != nil && msg.NextAttempt.After(now) {
			continue
		}
		count++
	}
	return count, nil
}

func (r *txOnlyRepo) GetIdempotencyRecord(_ context.Context, key, operation string) (*model.IdempotencyRecord, error) {
	if r.idempotency == nil {
		return nil, nil
	}
	rec, ok := r.idempotency[key+"|"+operation]
	if !ok {
		return nil, nil
	}
	return &rec, nil
}

func (r *txOnlyRepo) CreateIdempotencyRecord(_ context.Context, record *model.IdempotencyRecord) error {
	if r.idempotency == nil {
		r.idempotency = make(map[string]model.IdempotencyRecord)
	}
	r.idempotency[record.Key+"|"+record.Operation] = *record
	return nil
}

func (r *txOnlyRepo) DeleteIdempotencyRecordsBefore(_ context.Context, cutoff time.Time, limit int) (int64, error) {
	if limit <= 0 {
		limit = len(r.idempotency)
	}

	var deleted int64
	for k, rec := range r.idempotency {
		if !rec.CreatedAt.Before(cutoff) {
			continue
		}
		delete(r.idempotency, k)
		deleted++
		if int(deleted) >= limit {
			break
		}
	}
	return deleted, nil
}

type capturedPublisher struct {
	events []string
}

func (p *capturedPublisher) Publish(_ context.Context, event proto.Message, eventType string) error {
	if event == nil {
		return errors.New("event is required")
	}
	p.events = append(p.events, eventType)
	return nil
}

func (p *capturedPublisher) Close() error {
	return nil
}

type failingPublisher struct{}

func (p *failingPublisher) Publish(_ context.Context, _ proto.Message, _ string) error {
	return errors.New("publish failed")
}

func (p *failingPublisher) Close() error {
	return nil
}

func TestWithTxFlushesBufferedEventsAfterCommit(t *testing.T) {
	repo := &txOnlyRepo{}
	publisher := &capturedPublisher{}
	engine := NewEngine(repo, publisher)

	err := engine.withTx(context.Background(), func(txEngine *Engine) error {
		return txEngine.eventPublisher.Publish(context.Background(), &pb.ProcessInstanceCreated{}, "ProcessInstanceCreated")
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if repo.txCalls != 1 {
		t.Fatalf("expected one tx call, got %d", repo.txCalls)
	}
	if len(publisher.events) != 1 {
		t.Fatalf("expected 1 flushed event, got %d", len(publisher.events))
	}
	if publisher.events[0] != "ProcessInstanceCreated" {
		t.Fatalf("expected ProcessInstanceCreated event, got %q", publisher.events[0])
	}
}

func TestWithTxSkipsBufferedEventsOnRollback(t *testing.T) {
	repo := &txOnlyRepo{}
	publisher := &capturedPublisher{}
	engine := NewEngine(repo, publisher)

	expectedErr := errors.New("force rollback")
	err := engine.withTx(context.Background(), func(txEngine *Engine) error {
		if publishErr := txEngine.eventPublisher.Publish(context.Background(), &pb.JobActivated{}, "JobActivated"); publishErr != nil {
			return publishErr
		}
		return expectedErr
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected rollback error %v, got %v", expectedErr, err)
	}

	if len(publisher.events) != 0 {
		t.Fatalf("expected no flushed events on rollback, got %d", len(publisher.events))
	}
}

func TestIdempotencyMetricsHitMiss(t *testing.T) {
	repo := &txOnlyRepo{}
	engine := NewEngine(repo, &capturedPublisher{})

	hit, err := engine.HasProcessedIdempotencyKey(context.Background(), "k1", "op1")
	if err != nil {
		t.Fatalf("unexpected error checking idempotency miss: %v", err)
	}
	if hit {
		t.Fatalf("expected miss for unknown idempotency key")
	}

	if err := engine.RecordIdempotencyKey(context.Background(), "k1", "op1"); err != nil {
		t.Fatalf("failed to record idempotency key: %v", err)
	}

	hit, err = engine.HasProcessedIdempotencyKey(context.Background(), "k1", "op1")
	if err != nil {
		t.Fatalf("unexpected error checking idempotency hit: %v", err)
	}
	if !hit {
		t.Fatalf("expected idempotency hit")
	}

	snapshot := engine.MetricsSnapshot()
	if snapshot.IdempotencyMiss != 1 {
		t.Fatalf("expected idempotency miss metric 1, got %d", snapshot.IdempotencyMiss)
	}
	if snapshot.IdempotencyHit != 1 {
		t.Fatalf("expected idempotency hit metric 1, got %d", snapshot.IdempotencyHit)
	}
}

func TestRunOutboxRelayCyclePublishesClaimedMessages(t *testing.T) {
	repo := &txOnlyRepo{outbox: map[string]model.OutboxMessage{}}
	publisher := &capturedPublisher{}
	engine := NewEngine(repo, publisher)

	payload, err := proto.Marshal(&pb.JobActivated{Key: 101})
	if err != nil {
		t.Fatalf("failed to marshal test payload: %v", err)
	}

	repo.outbox["msg-1"] = model.OutboxMessage{
		ID:        "msg-1",
		EventType: "JobActivated",
		Payload:   payload,
		Status:    "PENDING",
		CreatedAt: time.Now().Add(-time.Second),
	}

	result, err := engine.RunOutboxRelayCycle(context.Background(), 10)
	if err != nil {
		t.Fatalf("relay cycle failed: %v", err)
	}
	if result.Claimed != 1 || result.Published != 1 || result.Failed != 0 {
		t.Fatalf("unexpected relay result: %+v", result)
	}

	msg := repo.outbox["msg-1"]
	if msg.Status != "PUBLISHED" {
		t.Fatalf("expected outbox message to be published, got %s", msg.Status)
	}
	if msg.PublishedAt == nil {
		t.Fatalf("expected published_at to be set")
	}

	snapshot := engine.MetricsSnapshot()
	if snapshot.OutboxPending == 0 {
		t.Fatalf("expected outbox pending metric to be set before publish")
	}
	if snapshot.OutboxPublishSuccess == 0 {
		t.Fatalf("expected outbox success metric to increment")
	}
	if snapshot.OutboxPublishLagSec == 0 {
		t.Fatalf("expected outbox publish lag metric to be set")
	}
}

func TestRunOutboxRelayCycleSchedulesRetryOnFailure(t *testing.T) {
	repo := &txOnlyRepo{outbox: map[string]model.OutboxMessage{}}
	engine := NewEngine(repo, &failingPublisher{})

	payload, err := proto.Marshal(&pb.JobActivated{Key: 102})
	if err != nil {
		t.Fatalf("failed to marshal test payload: %v", err)
	}

	repo.outbox["msg-2"] = model.OutboxMessage{
		ID:        "msg-2",
		EventType: "JobActivated",
		Payload:   payload,
		Status:    "PENDING",
		CreatedAt: time.Now().Add(-time.Second),
	}

	result, err := engine.RunOutboxRelayCycle(context.Background(), 10)
	if err != nil {
		t.Fatalf("relay cycle failed: %v", err)
	}
	if result.Failed != 1 {
		t.Fatalf("expected one failed relay, got %+v", result)
	}

	msg := repo.outbox["msg-2"]
	if msg.Status != "PENDING" {
		t.Fatalf("expected failed message to return to pending, got %s", msg.Status)
	}
	if msg.NextAttempt == nil {
		t.Fatalf("expected next attempt to be set for retry")
	}
	if msg.LastError == "" {
		t.Fatalf("expected last_error to be set")
	}

	snapshot := engine.MetricsSnapshot()
	if snapshot.OutboxPublishFailure == 0 {
		t.Fatalf("expected outbox failure metric to increment")
	}
}

func TestRunOutboxRelayCycleMarksTerminalFailureAtMaxAttempts(t *testing.T) {
	repo := &txOnlyRepo{outbox: map[string]model.OutboxMessage{}}
	engine := NewEngine(repo, &failingPublisher{})
	engine.SetOutboxMaxAttempts(2)

	payload, err := proto.Marshal(&pb.JobActivated{Key: 103})
	if err != nil {
		t.Fatalf("failed to marshal test payload: %v", err)
	}

	repo.outbox["msg-3"] = model.OutboxMessage{
		ID:        "msg-3",
		EventType: "JobActivated",
		Payload:   payload,
		Status:    "PENDING",
		Attempts:  1,
		CreatedAt: time.Now().Add(-time.Second),
	}

	result, err := engine.RunOutboxRelayCycle(context.Background(), 10)
	if err != nil {
		t.Fatalf("relay cycle failed: %v", err)
	}
	if result.Failed != 1 {
		t.Fatalf("expected one failed relay, got %+v", result)
	}

	msg := repo.outbox["msg-3"]
	if msg.Status != "FAILED" {
		t.Fatalf("expected failed terminal status, got %s", msg.Status)
	}
	if msg.NextAttempt != nil {
		t.Fatalf("expected no next attempt for terminal failed message")
	}
}

func TestRunIdempotencyCleanupDeletesExpiredRecords(t *testing.T) {
	now := time.Now()
	repo := &txOnlyRepo{idempotency: map[string]model.IdempotencyRecord{
		"old|op": {
			Key:       "old",
			Operation: "op",
			CreatedAt: now.Add(-48 * time.Hour),
		},
		"new|op": {
			Key:       "new",
			Operation: "op",
			CreatedAt: now.Add(-2 * time.Hour),
		},
	}}

	engine := NewEngine(repo, &capturedPublisher{})
	result, err := engine.RunIdempotencyCleanup(context.Background(), 24*time.Hour, 10)
	if err != nil {
		t.Fatalf("idempotency cleanup failed: %v", err)
	}
	if result.Deleted != 1 {
		t.Fatalf("expected one deleted record, got %d", result.Deleted)
	}
	if _, ok := repo.idempotency["old|op"]; ok {
		t.Fatalf("expected old idempotency record to be removed")
	}
	if _, ok := repo.idempotency["new|op"]; !ok {
		t.Fatalf("expected new idempotency record to remain")
	}
}
