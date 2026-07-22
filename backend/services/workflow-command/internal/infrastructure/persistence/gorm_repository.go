package persistence

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/azizAltaleb/flowgo/backend/libs/logger"
	"github.com/azizAltaleb/flowgo/backend/libs/model"
	"github.com/azizAltaleb/flowgo/backend/services/workflow-command/internal/domain/repository"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GormRepository struct {
	DB *gorm.DB
}

const outboxProcessingLease = 5 * time.Minute

// Ensure GormRepository implements repository.Repository
var _ repository.Repository = &GormRepository{}

func NewGormRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{DB: db}
}

func NewPostgresRepository(dsn string) (*GormRepository, error) {
	// Use custom GORM logger that integrates with our structured logging
	// and suppresses noisy "record not found" messages
	gormLogger := logger.NewGormLogger("gorm")

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %v", err)
	}

	// Auto-migrate
	if err := db.AutoMigrate(
		&model.Process{},
		&model.ProcessInstance{},
		&model.ElementInstance{},
		&model.Variable{},
		&model.Job{},
		&model.Incident{},
		&model.Timer{},
		&model.MessageSubscription{},
		&model.IdempotencyRecord{},
		&model.OutboxMessage{},
	); err != nil {
		return nil, fmt.Errorf("failed to migrate postgres schema: %v", err)
	}

	return NewGormRepository(db), nil
}

func (s *GormRepository) WithTx(ctx context.Context, fn func(txRepo repository.Repository) error) error {
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txRepo := &GormRepository{DB: tx}
		return fn(txRepo)
	})
}

// --- ENGINE OPERATIONS ---

func (s *GormRepository) CreateProcess(ctx context.Context, process *model.Process) error {
	return s.DB.WithContext(ctx).Create(process).Error
}

func (s *GormRepository) GetProcess(ctx context.Context, key int64) (*model.Process, error) {
	var p model.Process
	if err := s.DB.WithContext(ctx).First(&p, "key = ?", key).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *GormRepository) DeleteProcess(ctx context.Context, key int64) error {
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Find all instances
		var instances []model.ProcessInstance
		if err := tx.Where("process_definition_key = ?", key).Find(&instances).Error; err != nil {
			return err
		}

		// 2. Delete all instances and their related data
		for _, instance := range instances {
			if err := tx.Delete(&model.Variable{}, "process_instance_key = ?", instance.Key).Error; err != nil {
				return err
			}
			if err := tx.Delete(&model.ElementInstance{}, "process_instance_key = ?", instance.Key).Error; err != nil {
				return err
			}
			if err := tx.Delete(&model.Job{}, "process_instance_key = ?", instance.Key).Error; err != nil {
				return err
			}
			if err := tx.Delete(&model.Incident{}, "process_instance_key = ?", instance.Key).Error; err != nil {
				return err
			}
			if err := tx.Delete(&model.Timer{}, "process_instance_key = ?", instance.Key).Error; err != nil {
				return err
			}
			if err := tx.Delete(&model.MessageSubscription{}, "process_instance_key = ?", instance.Key).Error; err != nil {
				return err
			}
			if err := tx.Delete(&model.ProcessInstance{}, "key = ?", instance.Key).Error; err != nil {
				return err
			}
		}

		// 3. Delete the process definition
		if err := tx.Delete(&model.Process{}, "key = ?", key).Error; err != nil {
			return err
		}

		return nil
	})
}

func (s *GormRepository) CreateProcessInstance(ctx context.Context, instance *model.ProcessInstance) error {
	return s.DB.WithContext(ctx).Create(instance).Error
}

func (s *GormRepository) GetProcessInstance(ctx context.Context, key int64) (*model.ProcessInstance, error) {
	var pi model.ProcessInstance
	if err := s.DB.WithContext(ctx).First(&pi, "key = ?", key).Error; err != nil {
		return nil, err
	}
	return &pi, nil
}

func (s *GormRepository) UpdateProcessInstance(ctx context.Context, instance *model.ProcessInstance) error {
	return s.DB.WithContext(ctx).Model(instance).Updates(map[string]interface{}{
		"state":    instance.State,
		"end_time": instance.EndTime,
	}).Error
}

func (s *GormRepository) DeleteProcessInstance(ctx context.Context, key int64) error {
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&model.Variable{}, "process_instance_key = ?", key).Error; err != nil {
			return err
		}
		if err := tx.Delete(&model.ElementInstance{}, "process_instance_key = ?", key).Error; err != nil {
			return err
		}
		if err := tx.Delete(&model.Job{}, "process_instance_key = ?", key).Error; err != nil {
			return err
		}
		if err := tx.Delete(&model.Incident{}, "process_instance_key = ?", key).Error; err != nil {
			return err
		}
		if err := tx.Delete(&model.Timer{}, "process_instance_key = ?", key).Error; err != nil {
			return err
		}
		if err := tx.Delete(&model.MessageSubscription{}, "process_instance_key = ?", key).Error; err != nil {
			return err
		}
		if err := tx.Delete(&model.ProcessInstance{}, "key = ?", key).Error; err != nil {
			return err
		}
		return nil
	})
}

func (s *GormRepository) CreateElementInstance(ctx context.Context, element *model.ElementInstance) error {
	return s.DB.WithContext(ctx).Create(element).Error
}

func (s *GormRepository) UpdateElementInstance(ctx context.Context, element *model.ElementInstance) error {
	return s.DB.WithContext(ctx).Model(element).Updates(map[string]interface{}{
		"state":    element.State,
		"end_time": element.EndTime,
	}).Error
}

func (s *GormRepository) CreateVariable(ctx context.Context, variable *model.Variable) error {
	return s.DB.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "scope_key"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"process_instance_key": variable.ProcessInstanceKey,
			"name":                 variable.Name,
			"value":                variable.Value,
			"updated_at":           variable.UpdatedAt,
		}),
	}).Create(variable).Error
}

func (s *GormRepository) UpdateVariable(ctx context.Context, variable *model.Variable) error {
	return s.DB.WithContext(ctx).Model(variable).Updates(map[string]interface{}{
		"value":      variable.Value,
		"updated_at": variable.UpdatedAt,
	}).Error
}

func (s *GormRepository) CreateJob(ctx context.Context, job *model.Job) error {
	return s.DB.WithContext(ctx).Create(job).Error
}

func (s *GormRepository) GetJob(ctx context.Context, key int64) (*model.Job, error) {
	var job model.Job
	if err := s.DB.WithContext(ctx).First(&job, "key = ?", key).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

func (s *GormRepository) UpdateJob(ctx context.Context, job *model.Job) error {
	return s.DB.WithContext(ctx).Model(job).Updates(map[string]interface{}{
		"worker":               job.Worker,
		"lock_expiration_time": job.LockExpirationTime,
		"retries":              job.Retries,
		"state":                job.State,
		"assignee":             job.Assignee,
		"candidate_users":      job.CandidateUsers,
		"candidate_groups":     job.CandidateGroups,
		"updated_at":           job.UpdatedAt,
	}).Error
}

func (s *GormRepository) ListActivatableJobs(ctx context.Context, jobType string, maxJobs int) ([]model.Job, error) {
	if maxJobs <= 0 {
		maxJobs = 1
	}
	if maxJobs > 100 {
		maxJobs = 100
	}

	now := time.Now()
	query := s.DB.WithContext(ctx).
		Where("(state = ? OR (state = ? AND lock_expiration_time IS NOT NULL AND lock_expiration_time <= ?)) AND retries > 0", "CREATED", "ACTIVATED", now)

	if jobType != "" {
		query = query.Where("type = ?", jobType)
	}

	if s.DB.Dialector.Name() != "sqlite" {
		query = query.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"})
	}

	var jobs []model.Job
	if err := query.Order("created_at asc").Limit(maxJobs).Find(&jobs).Error; err != nil {
		return nil, err
	}

	return jobs, nil
}

func (s *GormRepository) ListJobsByProcessInstanceAndType(ctx context.Context, processInstanceKey int64, jobType string) ([]model.Job, error) {
	query := s.DB.WithContext(ctx).
		Where("process_instance_key = ?", processInstanceKey)
	if jobType != "" {
		query = query.Where("type = ?", jobType)
	}

	var jobs []model.Job
	if err := query.Order("created_at asc").Find(&jobs).Error; err != nil {
		return nil, err
	}
	return jobs, nil
}

func (s *GormRepository) CreateIncident(ctx context.Context, incident *model.Incident) error {
	return s.DB.WithContext(ctx).Create(incident).Error
}

func (s *GormRepository) UpdateIncident(ctx context.Context, incident *model.Incident) error {
	return s.DB.WithContext(ctx).Model(incident).Updates(map[string]interface{}{
		"state":       incident.State,
		"resolved_at": incident.ResolvedAt,
	}).Error
}

func (s *GormRepository) CreateTimer(ctx context.Context, timer *model.Timer) error {
	return s.DB.WithContext(ctx).Create(timer).Error
}

func (s *GormRepository) GetTimer(ctx context.Context, key int64) (*model.Timer, error) {
	var timer model.Timer
	if err := s.DB.WithContext(ctx).First(&timer, "key = ?", key).Error; err != nil {
		return nil, err
	}
	return &timer, nil
}

func (s *GormRepository) UpdateTimer(ctx context.Context, timer *model.Timer) error {
	return s.DB.WithContext(ctx).Model(timer).Updates(map[string]interface{}{
		"state": timer.State,
	}).Error
}

func (s *GormRepository) CreateMessageSubscription(ctx context.Context, subscription *model.MessageSubscription) error {
	return s.DB.WithContext(ctx).Create(subscription).Error
}

func (s *GormRepository) GetMessageSubscription(ctx context.Context, key int64) (*model.MessageSubscription, error) {
	var sub model.MessageSubscription
	if err := s.DB.WithContext(ctx).First(&sub, "key = ?", key).Error; err != nil {
		return nil, err
	}
	return &sub, nil
}

func (s *GormRepository) UpdateMessageSubscription(ctx context.Context, subscription *model.MessageSubscription) error {
	return s.DB.WithContext(ctx).Model(subscription).Updates(map[string]interface{}{
		"state": subscription.State,
	}).Error
}

func (s *GormRepository) GetIdempotencyRecord(ctx context.Context, key, operation string) (*model.IdempotencyRecord, error) {
	var record model.IdempotencyRecord
	err := s.DB.WithContext(ctx).
		Where("key = ? AND operation = ?", key, operation).
		First(&record).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &record, nil
}

func (s *GormRepository) CreateIdempotencyRecord(ctx context.Context, record *model.IdempotencyRecord) error {
	return s.DB.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(record).Error
}

func (s *GormRepository) DeleteIdempotencyRecordsBefore(ctx context.Context, cutoff time.Time, limit int) (int64, error) {
	query := s.DB.WithContext(ctx).
		Where("created_at < ?", cutoff).
		Order("created_at asc")
	if limit > 0 {
		query = query.Limit(limit)
	}

	var records []model.IdempotencyRecord
	if err := query.Find(&records).Error; err != nil {
		return 0, err
	}
	if len(records) == 0 {
		return 0, nil
	}

	var deleted int64
	for _, rec := range records {
		result := s.DB.WithContext(ctx).
			Where("key = ? AND operation = ?", rec.Key, rec.Operation).
			Delete(&model.IdempotencyRecord{})
		if result.Error != nil {
			return deleted, result.Error
		}
		deleted += result.RowsAffected
	}

	return deleted, nil
}

func (s *GormRepository) CreateOutboxMessage(ctx context.Context, message *model.OutboxMessage) error {
	return s.DB.WithContext(ctx).Create(message).Error
}

func (s *GormRepository) ListPendingOutboxMessages(ctx context.Context, now time.Time, limit int) ([]model.OutboxMessage, error) {
	if limit <= 0 {
		limit = 100
	}

	var messages []model.OutboxMessage
	if err := s.DB.WithContext(ctx).
		Scopes(claimableOutboxMessages(now)).
		Order("created_at asc").
		Limit(limit).
		Find(&messages).Error; err != nil {
		return nil, err
	}

	return messages, nil
}

func (s *GormRepository) ClaimOutboxMessage(ctx context.Context, id string, claimedAt time.Time) (bool, error) {
	updates := map[string]any{
		"status":                "PROCESSING",
		"attempts":              gorm.Expr("attempts + ?", 1),
		"last_error":            "",
		"next_attempt":          nil,
		"processing_started_at": claimedAt,
	}

	result := s.DB.WithContext(ctx).
		Model(&model.OutboxMessage{}).
		Where("id = ?", id).
		Scopes(claimableOutboxMessages(claimedAt)).
		Updates(updates)

	if result.Error != nil {
		return false, result.Error
	}

	return result.RowsAffected > 0, nil
}

func (s *GormRepository) MarkOutboxMessagePublishFailed(ctx context.Context, id, lastError string, nextAttempt time.Time) error {
	return s.DB.WithContext(ctx).
		Model(&model.OutboxMessage{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":                "PENDING",
			"last_error":            lastError,
			"next_attempt":          nextAttempt,
			"processing_started_at": nil,
		}).Error
}

func (s *GormRepository) MarkOutboxMessageTerminalFailed(ctx context.Context, id, lastError string, failedAt time.Time) error {
	return s.DB.WithContext(ctx).
		Model(&model.OutboxMessage{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":                "FAILED",
			"last_error":            lastError,
			"next_attempt":          nil,
			"processing_started_at": nil,
			"published_at":          failedAt,
		}).Error
}

func (s *GormRepository) MarkOutboxMessagePublished(ctx context.Context, id string, publishedAt time.Time) error {
	return s.DB.WithContext(ctx).
		Model(&model.OutboxMessage{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":                "PUBLISHED",
			"last_error":            "",
			"next_attempt":          nil,
			"processing_started_at": nil,
			"published_at":          publishedAt,
		}).Error
}

func (s *GormRepository) CountPendingOutboxMessages(ctx context.Context, now time.Time) (int64, error) {
	var count int64
	err := s.DB.WithContext(ctx).
		Model(&model.OutboxMessage{}).
		Scopes(claimableOutboxMessages(now)).
		Count(&count).Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

func claimableOutboxMessages(now time.Time) func(*gorm.DB) *gorm.DB {
	staleBefore := now.Add(-outboxProcessingLease)
	return func(db *gorm.DB) *gorm.DB {
		return db.Where(
			"(status = ? AND (next_attempt IS NULL OR next_attempt <= ?)) OR "+
				"(status = ? AND ((processing_started_at IS NOT NULL AND processing_started_at <= ?) OR (processing_started_at IS NULL AND created_at <= ?)))",
			"PENDING", now,
			"PROCESSING", staleBefore, staleBefore,
		)
	}
}

func (s *GormRepository) ListDueTimers(ctx context.Context, now time.Time) ([]model.Timer, error) {
	var timers []model.Timer
	if err := s.DB.WithContext(ctx).Where("state = ? AND due_date <= ?", "CREATED", now).Find(&timers).Error; err != nil {
		return nil, err
	}
	return timers, nil
}

func (s *GormRepository) ListOverdueJobs(ctx context.Context, now time.Time) ([]model.Job, error) {
	var jobs []model.Job
	// Find Active jobs with DueDate set, past, and NOT yet breached
	if err := s.DB.WithContext(ctx).Where("state IN ? AND due_date IS NOT NULL AND due_date <= ? AND breached_at IS NULL", []string{"CREATED", "ACTIVATED"}, now).Find(&jobs).Error; err != nil {
		return nil, err
	}
	return jobs, nil
}

func (s *GormRepository) ListMessageSubscriptions(ctx context.Context, messageName, correlationKey string) ([]model.MessageSubscription, error) {
	query := s.DB.WithContext(ctx).Where("state = ? AND message_name = ?", "OPEN", messageName)
	if correlationKey != "" {
		query = query.Where("correlation_key = ?", correlationKey)
	}

	var subs []model.MessageSubscription
	if err := query.Find(&subs).Error; err != nil {
		return nil, err
	}
	return subs, nil
}

// --- QUERY HELPERS ---

func (s *GormRepository) GetProcessByBpmnProcessID(ctx context.Context, bpmnProcessID string) (*model.Process, error) {
	var p model.Process
	if err := s.DB.WithContext(ctx).Where("bpmn_process_id = ?", bpmnProcessID).Order("version desc").First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *GormRepository) GetProcessInstanceWithState(ctx context.Context, key int64) (*model.ProcessInstance, []model.ElementInstance, []model.Variable, error) {
	var pi model.ProcessInstance
	if err := s.DB.WithContext(ctx).First(&pi, "key = ?", key).Error; err != nil {
		return nil, nil, nil, err
	}

	var elements []model.ElementInstance
	if err := s.DB.WithContext(ctx).Where("process_instance_key = ?", key).Find(&elements).Error; err != nil {
		return nil, nil, nil, err
	}

	var variables []model.Variable
	if err := s.DB.WithContext(ctx).Where("process_instance_key = ?", key).Find(&variables).Error; err != nil {
		return nil, nil, nil, err
	}

	return &pi, elements, variables, nil
}

func (s *GormRepository) GetElementInstance(ctx context.Context, key int64) (*model.ElementInstance, error) {
	var el model.ElementInstance
	if err := s.DB.WithContext(ctx).First(&el, "key = ?", key).Error; err != nil {
		return nil, err
	}
	return &el, nil
}

func (s *GormRepository) ListActiveElementInstances(ctx context.Context, processInstanceKey int64) ([]model.ElementInstance, error) {
	return s.listActiveElementInstances(ctx, processInstanceKey, nil)
}

func (s *GormRepository) ListActiveElementInstancesByTypes(ctx context.Context, processInstanceKey int64, elementTypes []string) ([]model.ElementInstance, error) {
	return s.listActiveElementInstances(ctx, processInstanceKey, elementTypes)
}

func (s *GormRepository) listActiveElementInstances(ctx context.Context, processInstanceKey int64, elementTypes []string) ([]model.ElementInstance, error) {
	var elements []model.ElementInstance
	query := s.DB.WithContext(ctx).Where("state IN ?", []string{"ACTIVATED", "ACTIVATING", "COMPLETING"})
	if processInstanceKey != 0 {
		query = query.Where("process_instance_key = ?", processInstanceKey)
	}
	if len(elementTypes) > 0 {
		query = query.Where("bpmn_element_type IN ?", elementTypes)
	}
	if err := query.Find(&elements).Error; err != nil {
		return nil, err
	}
	return elements, nil
}

func (s *GormRepository) ListActiveProcessInstances(ctx context.Context) ([]*model.ProcessInstance, error) {
	var instances []*model.ProcessInstance
	if err := s.DB.WithContext(ctx).Where("state = ?", "ACTIVE").Find(&instances).Error; err != nil {
		return nil, err
	}
	return instances, nil
}

func (s *GormRepository) ListProcessInstancesByState(ctx context.Context, state string, limit int) ([]*model.ProcessInstance, error) {
	var instances []*model.ProcessInstance
	query := s.DB.WithContext(ctx)
	if state != "" {
		query = query.Where("state = ?", state)
	}
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Order("end_time desc, created_at desc").Find(&instances).Error; err != nil {
		return nil, err
	}
	return instances, nil
}

func (s *GormRepository) ListCompletedProcessInstancesWithCompletedJobsByWorkers(ctx context.Context, jobType string, workers []string, limit int) ([]*model.ProcessInstance, error) {
	normalizedWorkers := normalizeQueryValues(workers)
	if len(normalizedWorkers) == 0 {
		return []*model.ProcessInstance{}, nil
	}
	if limit <= 0 {
		limit = 100
	}

	var instances []*model.ProcessInstance
	err := s.DB.WithContext(ctx).
		Model(&model.ProcessInstance{}).
		Distinct("process_instance.*").
		Joins("JOIN job ON job.process_instance_key = process_instance.key").
		Where("process_instance.state = ?", "COMPLETED").
		Where("job.type = ?", jobType).
		Where("job.state = ?", "COMPLETED").
		Where("LOWER(TRIM(job.worker)) IN ?", normalizedWorkers).
		Order("process_instance.end_time desc, process_instance.created_at desc").
		Limit(limit).
		Find(&instances).Error
	if err != nil {
		return nil, err
	}
	return instances, nil
}

func normalizeQueryValues(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		normalized := strings.ToLower(strings.TrimSpace(value))
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	return out
}

func (s *GormRepository) ListProcesses(ctx context.Context) ([]*model.Process, error) {
	var processes []*model.Process
	if err := s.DB.WithContext(ctx).Order("created_at desc").Find(&processes).Error; err != nil {
		return nil, err
	}
	return processes, nil
}
