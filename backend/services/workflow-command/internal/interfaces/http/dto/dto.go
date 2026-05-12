package dto

import (
	"github.com/azizAltaleb/flowgo/backend/libs/model"
	"time"
)

// --- Workflow Definitions ---

type WorkflowDefinitionResponse struct {
	ID                  string    `json:"id"` // Converted to string for JSON API consistency (even if int64 internally)
	ProcessDefinitionID string    `json:"process_definition_id"`
	Name                string    `json:"name"`
	Version             int       `json:"version"`
	ResourceName        string    `json:"resource_name"`
	DeploymentID        string    `json:"deployment_id"`
	TenantID            string    `json:"tenant_id"`
	ResourceChecksum    string    `json:"resource_checksum"`
	BPMNXML             string    `json:"bpmn_xml,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
	// Steps are usually not returned in full detail in list/get summaries unless requested,
	// but to mirror model exactly for now:
	Steps []model.StepDefinition `json:"steps"`
}

// --- Workflow Instances ---

type WorkflowInstanceResponse struct {
	ID                string         `json:"id"`
	WorkflowID        string         `json:"workflow_id"`
	Status            string         `json:"status"` // string representation of WorkflowStatus
	ParentInstanceID  string         `json:"parent_instance_id,omitempty"`
	ParentExecutionID string         `json:"parent_execution_id,omitempty"`
	Context           map[string]any `json:"context"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	// Executions might be detailed or distinct DTOs too, but let's decouple top level first.
	// For deep structures, we might still use model types if we don't want to duplicate everything yet.
	// ADR goal is boundaries.
	Executions []model.Execution `json:"executions"`
}

// --- Requests (Moved from handler.go) ---

type StartInstanceRequest struct {
	WorkflowID string         `json:"workflow_id"`
	Context    map[string]any `json:"context"`
}

type UpdateVariablesRequest struct {
	Variables map[string]any `json:"variables"`
}

type CompleteTaskRequest struct {
	StepID string `json:"step_id"`
}

type ActivateJobsRequest struct {
	Type           string `json:"type"`
	Worker         string `json:"worker"`
	MaxJobs        int    `json:"maxJobs"`
	TimeoutMs      int    `json:"timeoutMs"`
	LockDurationMs int    `json:"lockDurationMs"`
}

type ActivateJobsResponse struct {
	Jobs []JobResponse `json:"jobs"`
}

type WorkerCapabilitiesResponse struct {
	ProtocolVersion string   `json:"protocolVersion"`
	Capabilities    []string `json:"capabilities"`
}

type EngineMetricsResponse struct {
	OutboxPending        int64 `json:"outboxPending"`
	OutboxPublishSuccess int64 `json:"outboxPublishSuccess"`
	OutboxPublishFailure int64 `json:"outboxPublishFailure"`
	OutboxPublishLagSec  int64 `json:"outboxPublishLagSec"`
	OutboxMaxAttempts    int   `json:"outboxMaxAttempts"`
	IdempotencyHit       int64 `json:"idempotencyHit"`
	IdempotencyMiss      int64 `json:"idempotencyMiss"`
}

type CompleteJobRequest struct {
	Worker    string         `json:"worker"`
	Variables map[string]any `json:"variables"`
}

type FailJobRequest struct {
	Worker       string `json:"worker"`
	ErrorMessage string `json:"errorMessage"`
	Retries      *int   `json:"retries"`
}

type ExtendJobLockRequest struct {
	Worker         string `json:"worker"`
	LockDurationMs int    `json:"lockDurationMs"`
}

type PublishSignalRequest struct {
	SignalName string         `json:"signal_name"`
	Payload    map[string]any `json:"payload"`
}

type PublishMessageRequest struct {
	MessageName    string         `json:"message_name"`
	CorrelationKey string         `json:"correlation_key"`
	Payload        map[string]any `json:"payload"`
}

// --- Job Response ---

type JobResponse struct {
	Key                  string     `json:"key"` // String for API
	Type                 string     `json:"type"`
	ProcessInstanceKey   string     `json:"processInstanceKey"`
	ElementInstanceKey   string     `json:"elementInstanceKey"`
	ProcessDefinitionKey string     `json:"processDefinitionKey"`
	ElementID            string     `json:"elementId"`
	Worker               string     `json:"worker"`
	Retries              int        `json:"retries"`
	State                string     `json:"state"`
	LockExpirationTime   *time.Time `json:"lockExpirationTime,omitempty"`
	CreatedAt            time.Time  `json:"createdAt"`
	UpdatedAt            time.Time  `json:"updatedAt"`
}
