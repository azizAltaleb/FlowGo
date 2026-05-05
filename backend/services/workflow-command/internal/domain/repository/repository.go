package repository

import (
	"context"
	"time"
	"workflow-engine/backend/libs/model"
)

type Repository interface {
	WithTx(ctx context.Context, fn func(txRepo Repository) error) error

	// Engine-like Operations (Incremental Adoption)
	CreateProcess(ctx context.Context, process *model.Process) error
	GetProcess(ctx context.Context, key int64) (*model.Process, error)
	DeleteProcess(ctx context.Context, key int64) error // New
	CreateProcessInstance(ctx context.Context, instance *model.ProcessInstance) error
	GetProcessInstance(ctx context.Context, key int64) (*model.ProcessInstance, error)
	UpdateProcessInstance(ctx context.Context, instance *model.ProcessInstance) error
	DeleteProcessInstance(ctx context.Context, key int64) error // New
	CreateElementInstance(ctx context.Context, element *model.ElementInstance) error
	UpdateElementInstance(ctx context.Context, element *model.ElementInstance) error
	CreateVariable(ctx context.Context, variable *model.Variable) error
	UpdateVariable(ctx context.Context, variable *model.Variable) error
	CreateJob(ctx context.Context, job *model.Job) error
	GetJob(ctx context.Context, key int64) (*model.Job, error)
	UpdateJob(ctx context.Context, job *model.Job) error
	ListActivatableJobs(ctx context.Context, jobType string, maxJobs int) ([]model.Job, error)
	CreateIncident(ctx context.Context, incident *model.Incident) error
	UpdateIncident(ctx context.Context, incident *model.Incident) error
	CreateTimer(ctx context.Context, timer *model.Timer) error
	GetTimer(ctx context.Context, key int64) (*model.Timer, error)
	UpdateTimer(ctx context.Context, timer *model.Timer) error
	CreateMessageSubscription(ctx context.Context, subscription *model.MessageSubscription) error
	GetMessageSubscription(ctx context.Context, key int64) (*model.MessageSubscription, error)
	UpdateMessageSubscription(ctx context.Context, subscription *model.MessageSubscription) error
	GetIdempotencyRecord(ctx context.Context, key, operation string) (*model.IdempotencyRecord, error)
	CreateIdempotencyRecord(ctx context.Context, record *model.IdempotencyRecord) error
	CreateOutboxMessage(ctx context.Context, message *model.OutboxMessage) error
	ListPendingOutboxMessages(ctx context.Context, now time.Time, limit int) ([]model.OutboxMessage, error)
	ClaimOutboxMessage(ctx context.Context, id string, claimedAt time.Time) (bool, error)
	MarkOutboxMessagePublishFailed(ctx context.Context, id, lastError string, nextAttempt time.Time) error
	MarkOutboxMessageTerminalFailed(ctx context.Context, id, lastError string, failedAt time.Time) error
	MarkOutboxMessagePublished(ctx context.Context, id string, publishedAt time.Time) error
	CountPendingOutboxMessages(ctx context.Context, now time.Time) (int64, error)
	DeleteIdempotencyRecordsBefore(ctx context.Context, cutoff time.Time, limit int) (int64, error)
	ListDueTimers(ctx context.Context, now time.Time) ([]model.Timer, error)
	ListOverdueJobs(ctx context.Context, now time.Time) ([]model.Job, error) // New for SLA
	ListMessageSubscriptions(ctx context.Context, messageName, correlationKey string) ([]model.MessageSubscription, error)

	// Engine Query Helpers
	GetProcessByBpmnProcessID(ctx context.Context, bpmnProcessID string) (*model.Process, error)                                           // Get latest version
	GetProcessInstanceWithState(ctx context.Context, key int64) (*model.ProcessInstance, []model.ElementInstance, []model.Variable, error) // For re-hydrating state
	GetElementInstance(ctx context.Context, key int64) (*model.ElementInstance, error)
	ListActiveElementInstances(ctx context.Context, processInstanceKey int64) ([]model.ElementInstance, error)
	ListActiveProcessInstances(ctx context.Context) ([]*model.ProcessInstance, error)
	ListProcesses(ctx context.Context) ([]*model.Process, error)
}
