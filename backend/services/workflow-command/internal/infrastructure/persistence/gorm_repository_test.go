package persistence

import (
	"context"
	"testing"
	"time"
	"workflow-engine/backend/libs/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupGormRepositoryTest(t *testing.T) *GormRepository {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite db: %v", err)
	}

	if err := db.AutoMigrate(&model.IdempotencyRecord{}, &model.OutboxMessage{}); err != nil {
		t.Fatalf("failed to migrate schema: %v", err)
	}

	return NewGormRepository(db)
}

func TestDeleteIdempotencyRecordsBeforeRespectsLimit(t *testing.T) {
	repo := setupGormRepositoryTest(t)
	now := time.Now()

	records := []model.IdempotencyRecord{
		{Key: "k-old-1", Operation: "jobs.complete:1", CreatedAt: now.Add(-72 * time.Hour)},
		{Key: "k-old-2", Operation: "jobs.complete:2", CreatedAt: now.Add(-48 * time.Hour)},
		{Key: "k-new", Operation: "jobs.complete:3", CreatedAt: now.Add(-2 * time.Hour)},
	}
	for i := range records {
		if err := repo.CreateIdempotencyRecord(context.Background(), &records[i]); err != nil {
			t.Fatalf("failed to seed idempotency record: %v", err)
		}
	}

	deleted, err := repo.DeleteIdempotencyRecordsBefore(context.Background(), now.Add(-24*time.Hour), 1)
	if err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("expected one deleted record, got %d", deleted)
	}

	var oldCount int64
	if err := repo.DB.WithContext(context.Background()).
		Model(&model.IdempotencyRecord{}).
		Where("created_at < ?", now.Add(-24*time.Hour)).
		Count(&oldCount).Error; err != nil {
		t.Fatalf("failed to query remaining old records: %v", err)
	}
	if oldCount != 1 {
		t.Fatalf("expected one old record to remain due to limit, got %d", oldCount)
	}
}

func TestMarkOutboxMessageTerminalFailed(t *testing.T) {
	repo := setupGormRepositoryTest(t)
	now := time.Now()

	message := &model.OutboxMessage{
		ID:        "outbox-terminal-1",
		EventType: "JobActivated",
		Payload:   []byte("{}"),
		Status:    "PENDING",
		CreatedAt: now,
	}
	if err := repo.CreateOutboxMessage(context.Background(), message); err != nil {
		t.Fatalf("failed to seed outbox message: %v", err)
	}

	if err := repo.MarkOutboxMessageTerminalFailed(context.Background(), message.ID, "publish failed", now); err != nil {
		t.Fatalf("failed to mark terminal failure: %v", err)
	}

	var stored model.OutboxMessage
	if err := repo.DB.WithContext(context.Background()).First(&stored, "id = ?", message.ID).Error; err != nil {
		t.Fatalf("failed to load outbox message: %v", err)
	}
	if stored.Status != "FAILED" {
		t.Fatalf("expected status FAILED, got %s", stored.Status)
	}
	if stored.LastError != "publish failed" {
		t.Fatalf("expected last_error to be persisted, got %q", stored.LastError)
	}
	if stored.NextAttempt != nil {
		t.Fatalf("expected next_attempt to be nil for terminal failure")
	}
}
