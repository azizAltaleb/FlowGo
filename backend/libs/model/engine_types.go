package model

import (
	"time"
)

// Engine Schema Models

// Process represents a deployed workflow definition (BPMN process)
type Process struct {
	Key              int64     `json:"key,string" gorm:"primaryKey"` // Unique key (PK)
	BpmnProcessID    string    `json:"bpmnProcessId"`                // ID from BPMN XML (e.g. "order-process")
	Version          int       `json:"version"`                      // Version of the process
	ResourceName     string    `json:"resourceName"`                 // Name of the BPMN file
	DeploymentKey    int64     `json:"deploymentKey,string"`         // Key of the deployment
	Resource         []byte    `json:"resource"`                     // BPMN XML content
	ResourceChecksum string    `json:"resourceChecksum"`             // Checksum to detect duplicates
	TenantID         string    `json:"tenantId"`                     // Multi-tenancy support
	CreatedAt        time.Time `json:"createdAt"`
}

func (Process) TableName() string {
	return "process"
}

// ProcessInstance represents a running instance of a process
type ProcessInstance struct {
	Key                      int64          `json:"key,string" gorm:"primaryKey"`    // Unique key (PK)
	ID                       string         `json:"id" gorm:"index"`                 // Unique ID (UUIDv7)
	ProcessDefinitionKey     int64          `json:"processDefinitionKey,string"`     // FK to Process.Key
	Version                  int            `json:"version"`                         // Cached
	ParentProcessInstanceKey int64          `json:"parentProcessInstanceKey,string"` // For Call Activity
	ParentElementInstanceKey int64          `json:"parentElementInstanceKey,string"` // For Call Activity
	State                    string         `json:"state"`                           // ACTIVE, COMPLETED, CANCELED
	CreatedAt                time.Time      `json:"createdAt"`
	EndTime                  time.Time      `json:"endTime"`
	Context                  map[string]any `json:"context,omitempty" gorm:"-"` // Denormalized Context (ES only)
}

func (ProcessInstance) TableName() string {
	return "process_instance"
}

// ElementInstance represents a token/step execution state
type ElementInstance struct {
	Key                  int64     `json:"key,string" gorm:"primaryKey"`
	ID                   string    `json:"id" gorm:"index"` // Unique ID (UUIDv7)
	ProcessInstanceKey   int64     `json:"processInstanceKey,string"`
	ProcessDefinitionKey int64     `json:"processDefinitionKey,string"`
	ElementID            string    `json:"elementId"`           // ID of the element in BPMN
	BpmnElementType      string    `json:"bpmnElementType"`     // SERVICE_TASK, START_EVENT, etc.
	FlowScopeKey         int64     `json:"flowScopeKey,string"` // Key of parent scope (e.g. SubProcess)
	State                string    `json:"state"`               // ACTIVATING, ACTIVATED, COMPLETING, COMPLETED, TERMINATED
	CreatedAt            time.Time `json:"createdAt"`
	EndTime              time.Time `json:"endTime"`
}

func (ElementInstance) TableName() string {
	return "element_instance"
}

// Variable represents process variables/context
type Variable struct {
	ScopeKey           int64     `json:"scopeKey,string" gorm:"primaryKey"` // ProcessInstanceKey or ElementInstanceKey
	ProcessInstanceKey int64     `json:"processInstanceKey,string"`
	Name               string    `json:"name"`                    // Kept for backward compatibility or index
	Value              []byte    `json:"value" gorm:"type:jsonb"` // Full JSON context document
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

func (Variable) TableName() string {
	return "variable"
}

// Job represents a task for an external worker (Service Task)
type Job struct {
	Key                  int64      `json:"key,string" gorm:"primaryKey"`
	ID                   string     `json:"id" gorm:"index"` // Unique ID (UUIDv7)
	Type                 string     `json:"type"`            // Task type (e.g. "payment-service")
	ProcessInstanceKey   int64      `json:"processInstanceKey,string"`
	ElementInstanceKey   int64      `json:"elementInstanceKey,string"`
	ProcessDefinitionKey int64      `json:"processDefinitionKey,string"`
	ElementID            string     `json:"elementId"`
	Worker               string     `json:"worker"`  // ID of the worker that locked the job
	Retries              int        `json:"retries"` // Remaining retries
	State                string     `json:"state"`   // CREATED, ACTIVATED, COMPLETED, FAILED
	Assignee             string     `json:"assignee"`
	CandidateUsers       string     `json:"candidateUsers"`
	CandidateGroups      string     `json:"candidateGroups"`
	LockExpirationTime   *time.Time `json:"lockExpirationTime,omitempty"`
	DueDate              time.Time  `json:"dueDate"` // SLA Target
	BreachedAt           *time.Time `json:"breachedAt,omitempty"`
	CreatedAt            time.Time  `json:"createdAt"`
	UpdatedAt            time.Time  `json:"updatedAt"`
}

func (Job) TableName() string {
	return "job"
}

// Timer represents a timer event (intermediate catch or boundary)
type Timer struct {
	Key                int64     `json:"key,string" gorm:"primaryKey"`
	ID                 string    `json:"id" gorm:"index"` // Unique ID (UUIDv7)
	ElementInstanceKey int64     `json:"elementInstanceKey,string"`
	ProcessInstanceKey int64     `json:"processInstanceKey,string"`
	ElementID          string    `json:"elementId"` // ID of the timer element (step ID or boundary event ID)
	DueDate            time.Time `json:"dueDate"`
	State              string    `json:"state"` // CREATED, TRIGGERED, CANCELED
	RepeatCount        int       `json:"repeatCount"`
	CreatedAt          time.Time `json:"createdAt"`
}

func (Timer) TableName() string {
	return "timer"
}

// MessageSubscription represents a subscription for message correlation
type MessageSubscription struct {
	Key                int64     `json:"key,string" gorm:"primaryKey"`
	ID                 string    `json:"id" gorm:"index"` // Unique ID (UUIDv7)
	ElementInstanceKey int64     `json:"elementInstanceKey,string"`
	ProcessInstanceKey int64     `json:"processInstanceKey,string"`
	ElementID          string    `json:"elementId"` // ID of the message element (catch event or boundary event)
	MessageName        string    `json:"messageName"`
	CorrelationKey     string    `json:"correlationKey"`
	State              string    `json:"state"` // OPEN, CORRELATED, CLOSED
	CreatedAt          time.Time `json:"createdAt"`
}

func (MessageSubscription) TableName() string {
	return "message_subscription"
}

// Incident represents an error that halted execution
type Incident struct {
	Key                int64     `json:"key,string" gorm:"primaryKey"`
	ID                 string    `json:"id" gorm:"index"` // Unique ID (UUIDv7)
	ProcessInstanceKey int64     `json:"processInstanceKey,string"`
	ElementInstanceKey int64     `json:"elementInstanceKey,string"`
	JobKey             int64     `json:"jobKey,string"` // Optional
	ErrorType          string    `json:"errorType"`
	ErrorMessage       string    `json:"errorMessage"`
	State              string    `json:"state"` // CREATED, RESOLVED
	CreatedAt          time.Time `json:"createdAt"`
	ResolvedAt         time.Time `json:"resolvedAt"`
}

func (Incident) TableName() string {
	return "incident"
}

// IdempotencyRecord stores completed externally-triggered operations.
// It enables safe retries from clients without reapplying state transitions.
type IdempotencyRecord struct {
	Key       string    `json:"key" gorm:"primaryKey;size:128"`
	Operation string    `json:"operation" gorm:"primaryKey;size:64"`
	CreatedAt time.Time `json:"createdAt"`
}

func (IdempotencyRecord) TableName() string {
	return "idempotency_record"
}

// OutboxMessage stores domain events produced inside transactions.
// Messages are inserted during commit and marked published after dispatch.
type OutboxMessage struct {
	ID          string     `json:"id" gorm:"primaryKey;size:64"`
	EventType   string     `json:"eventType" gorm:"index;size:128"`
	Payload     []byte     `json:"payload" gorm:"type:bytea"`
	Status      string     `json:"status" gorm:"index;size:16"`
	Attempts    int        `json:"attempts"`
	LastError   string     `json:"lastError" gorm:"type:text"`
	NextAttempt *time.Time `json:"nextAttempt,omitempty" gorm:"index"`
	CreatedAt   time.Time  `json:"createdAt"`
	PublishedAt *time.Time `json:"publishedAt,omitempty"`
}

func (OutboxMessage) TableName() string {
	return "outbox_message"
}
