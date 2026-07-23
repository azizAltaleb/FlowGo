package persistence

import (
	"context"
	"github.com/artificialflow/artificialflow/backend/libs/model"
	"github.com/artificialflow/artificialflow/backend/services/workflow-command/internal/application"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupGormRepositoryTest(t *testing.T) *GormRepository {
	t.Helper()

	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open("file:"+dbName+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite db: %v", err)
	}

	if err := db.AutoMigrate(&model.ProcessInstance{}, &model.Job{}, &model.IdempotencyRecord{}, &model.OutboxMessage{}); err != nil {
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

func TestListCompletedProcessInstancesWithCompletedJobsByWorkers(t *testing.T) {
	repo := setupGormRepositoryTest(t)
	ctx := context.Background()
	now := time.Now().UTC()

	instances := []model.ProcessInstance{
		{Key: 1, ID: "old-match", ProcessDefinitionKey: 100, Version: 1, State: "COMPLETED", CreatedAt: now.Add(-4 * time.Hour), EndTime: now.Add(-3 * time.Hour)},
		{Key: 2, ID: "new-other-worker", ProcessDefinitionKey: 100, Version: 1, State: "COMPLETED", CreatedAt: now.Add(-3 * time.Hour), EndTime: now.Add(-2 * time.Hour)},
		{Key: 3, ID: "new-match", ProcessDefinitionKey: 100, Version: 1, State: "COMPLETED", CreatedAt: now.Add(-2 * time.Hour), EndTime: now.Add(-time.Hour)},
		{Key: 4, ID: "active-match", ProcessDefinitionKey: 100, Version: 1, State: "ACTIVE", CreatedAt: now.Add(-time.Hour), EndTime: now},
		{Key: 5, ID: "created-job", ProcessDefinitionKey: 100, Version: 1, State: "COMPLETED", CreatedAt: now.Add(-time.Hour), EndTime: now.Add(-30 * time.Minute)},
		{Key: 6, ID: "wrong-type", ProcessDefinitionKey: 100, Version: 1, State: "COMPLETED", CreatedAt: now.Add(-time.Hour), EndTime: now.Add(-15 * time.Minute)},
	}
	for i := range instances {
		if err := repo.CreateProcessInstance(ctx, &instances[i]); err != nil {
			t.Fatalf("failed to seed process instance %d: %v", instances[i].Key, err)
		}
	}

	jobs := []model.Job{
		{Key: 101, Type: application.UserTaskJobType, ProcessInstanceKey: 1, ElementInstanceKey: 1001, ElementID: "old", Worker: " Accountant@ArtificialFlow.Local ", Retries: 1, State: "COMPLETED", CreatedAt: now.Add(-4 * time.Hour), UpdatedAt: now.Add(-3 * time.Hour)},
		{Key: 102, Type: application.UserTaskJobType, ProcessInstanceKey: 2, ElementInstanceKey: 1002, ElementID: "other", Worker: "reviewer@artificialflow.local", Retries: 1, State: "COMPLETED", CreatedAt: now.Add(-3 * time.Hour), UpdatedAt: now.Add(-2 * time.Hour)},
		{Key: 103, Type: application.UserTaskJobType, ProcessInstanceKey: 3, ElementInstanceKey: 1003, ElementID: "new", Worker: "accountant@artificialflow.local", Retries: 1, State: "COMPLETED", CreatedAt: now.Add(-2 * time.Hour), UpdatedAt: now.Add(-time.Hour)},
		{Key: 104, Type: application.UserTaskJobType, ProcessInstanceKey: 3, ElementInstanceKey: 1004, ElementID: "duplicate", Worker: "accountant@artificialflow.local", Retries: 1, State: "COMPLETED", CreatedAt: now.Add(-2 * time.Hour), UpdatedAt: now.Add(-time.Hour)},
		{Key: 105, Type: application.UserTaskJobType, ProcessInstanceKey: 4, ElementInstanceKey: 1005, ElementID: "active", Worker: "accountant@artificialflow.local", Retries: 1, State: "COMPLETED", CreatedAt: now.Add(-time.Hour), UpdatedAt: now},
		{Key: 106, Type: application.UserTaskJobType, ProcessInstanceKey: 5, ElementInstanceKey: 1006, ElementID: "created", Worker: "accountant@artificialflow.local", Retries: 1, State: "CREATED", CreatedAt: now.Add(-time.Hour), UpdatedAt: now},
		{Key: 107, Type: "service", ProcessInstanceKey: 6, ElementInstanceKey: 1007, ElementID: "wrong-type", Worker: "accountant@artificialflow.local", Retries: 1, State: "COMPLETED", CreatedAt: now.Add(-time.Hour), UpdatedAt: now},
	}
	for i := range jobs {
		if err := repo.CreateJob(ctx, &jobs[i]); err != nil {
			t.Fatalf("failed to seed job %d: %v", jobs[i].Key, err)
		}
	}

	matches, err := repo.ListCompletedProcessInstancesWithCompletedJobsByWorkers(
		ctx,
		application.UserTaskJobType,
		[]string{"accountant@artificialflow.local", "ACCOUNTANT@artificialflow.LOCAL", " "},
		10,
	)
	if err != nil {
		t.Fatalf("failed to list completed process instances by workers: %v", err)
	}
	if len(matches) != 2 {
		t.Fatalf("expected two matching instances, got %#v", matches)
	}
	if matches[0].Key != 3 || matches[1].Key != 1 {
		t.Fatalf("expected matches ordered newest first without duplicates, got %#v", matches)
	}

	limited, err := repo.ListCompletedProcessInstancesWithCompletedJobsByWorkers(ctx, application.UserTaskJobType, []string{"accountant@artificialflow.local"}, 1)
	if err != nil {
		t.Fatalf("failed to list limited completed process instances by workers: %v", err)
	}
	if len(limited) != 1 || limited[0].Key != 3 {
		t.Fatalf("expected limit after worker filtering to return newest match, got %#v", limited)
	}
}

func TestUserTaskQueriesIncludeCanonicalAndLegacyTypes(t *testing.T) {
	repo := setupGormRepositoryTest(t)
	ctx := context.Background()
	now := time.Now().UTC()

	jobs := []model.Job{
		{Key: 201, Type: application.UserTaskJobType, ProcessInstanceKey: 20, ElementInstanceKey: 2001, Retries: 1, State: "CREATED", CreatedAt: now},
		{Key: 202, Type: application.UserTaskJobType, ProcessInstanceKey: 20, ElementInstanceKey: 2002, Retries: 1, State: "CREATED", CreatedAt: now.Add(time.Second)},
		{Key: 203, Type: "service", ProcessInstanceKey: 20, ElementInstanceKey: 2003, Retries: 1, State: "CREATED", CreatedAt: now.Add(2 * time.Second)},
	}
	for i := range jobs {
		if err := repo.CreateJob(ctx, &jobs[i]); err != nil {
			t.Fatalf("failed to seed job %d: %v", jobs[i].Key, err)
		}
	}

	for _, requestedType := range []string{application.UserTaskJobType, application.UserTaskJobType} {
		listed, err := repo.ListJobsByProcessInstanceAndType(ctx, 20, requestedType)
		if err != nil {
			t.Fatalf("failed to list %q jobs: %v", requestedType, err)
		}
		if len(listed) != 2 || listed[0].Key != 201 || listed[1].Key != 202 {
			t.Fatalf("expected both user-task types for %q, got %#v", requestedType, listed)
		}

		activatable, err := repo.ListActivatableJobs(ctx, requestedType, 10)
		if err != nil {
			t.Fatalf("failed to activate %q jobs: %v", requestedType, err)
		}
		if len(activatable) != 2 {
			t.Fatalf("expected both user-task types to be activatable for %q, got %#v", requestedType, activatable)
		}
	}
}

func TestOutboxClaimReclaimsStaleProcessingMessages(t *testing.T) {
	repo := setupGormRepositoryTest(t)
	ctx := context.Background()
	now := time.Now()
	staleClaim := now.Add(-10 * time.Minute)
	freshClaim := now.Add(-time.Minute)

	messages := []model.OutboxMessage{
		{
			ID:        "pending",
			EventType: "JobActivated",
			Payload:   []byte("{}"),
			Status:    "PENDING",
			CreatedAt: now.Add(-time.Hour),
		},
		{
			ID:                  "stale-processing",
			EventType:           "JobActivated",
			Payload:             []byte("{}"),
			Status:              "PROCESSING",
			Attempts:            1,
			ProcessingStartedAt: &staleClaim,
			CreatedAt:           now.Add(-time.Hour),
		},
		{
			ID:                  "fresh-processing",
			EventType:           "JobActivated",
			Payload:             []byte("{}"),
			Status:              "PROCESSING",
			Attempts:            1,
			ProcessingStartedAt: &freshClaim,
			CreatedAt:           now.Add(-time.Hour),
		},
	}
	for i := range messages {
		if err := repo.CreateOutboxMessage(ctx, &messages[i]); err != nil {
			t.Fatalf("failed to seed outbox message %s: %v", messages[i].ID, err)
		}
	}

	count, err := repo.CountPendingOutboxMessages(ctx, now)
	if err != nil {
		t.Fatalf("failed to count claimable messages: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected pending plus stale processing to be claimable, got %d", count)
	}

	listed, err := repo.ListPendingOutboxMessages(ctx, now, 10)
	if err != nil {
		t.Fatalf("failed to list claimable messages: %v", err)
	}
	seen := map[string]bool{}
	for _, message := range listed {
		seen[message.ID] = true
	}
	if !seen["pending"] || !seen["stale-processing"] {
		t.Fatalf("expected pending and stale processing messages to be listed, got %v", seen)
	}
	if seen["fresh-processing"] {
		t.Fatalf("did not expect fresh processing message to be listed")
	}

	claimed, err := repo.ClaimOutboxMessage(ctx, "stale-processing", now)
	if err != nil {
		t.Fatalf("failed to claim stale processing message: %v", err)
	}
	if !claimed {
		t.Fatalf("expected stale processing message to be claimed")
	}

	claimedFresh, err := repo.ClaimOutboxMessage(ctx, "fresh-processing", now)
	if err != nil {
		t.Fatalf("failed to attempt fresh processing claim: %v", err)
	}
	if claimedFresh {
		t.Fatalf("did not expect fresh processing message to be claimed")
	}

	var stored model.OutboxMessage
	if err := repo.DB.WithContext(ctx).First(&stored, "id = ?", "stale-processing").Error; err != nil {
		t.Fatalf("failed to load claimed message: %v", err)
	}
	if stored.Status != "PROCESSING" {
		t.Fatalf("expected stale message to remain PROCESSING after claim, got %s", stored.Status)
	}
	if stored.Attempts != 2 {
		t.Fatalf("expected attempts to increment to 2, got %d", stored.Attempts)
	}
	if stored.ProcessingStartedAt == nil {
		t.Fatalf("expected processing_started_at to be refreshed")
	}
}
