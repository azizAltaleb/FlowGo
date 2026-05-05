package model

import (
	"time"
)

type WorkflowStatus string

const (
	StatusPending   WorkflowStatus = "PENDING"
	StatusRunning   WorkflowStatus = "RUNNING"
	StatusCompleted WorkflowStatus = "COMPLETED"
	StatusFailed    WorkflowStatus = "FAILED"
)

// WorkflowDefinition represents the blueprint of a workflow
type WorkflowDefinition struct {
	// Technical Key (Unique Database ID)
	ID int64 `json:"id,string"`

	// Logical Process ID (e.g. "order-process" from BPMN XML)
	ProcessDefinitionID string `json:"process_definition_id"`

	// Name of the process (from BPMN)
	Name string `json:"name"`

	// Version of the process definition
	Version int `json:"version"`

	// XML Resource Name (e.g. "order.bpmn")
	ResourceName string `json:"resource_name"`

	// Deployment ID (linking to a deployment event)
	DeploymentID string `json:"deployment_id"`

	// Tenant ID for multi-tenancy
	TenantID string `json:"tenant_id"`

	// Checksum of the BPMN XML to detect duplicates
	ResourceChecksum string `json:"resource_checksum"`

	// BPMNXML stores the raw BPMN 2.0 XML content
	BPMNXML string `json:"bpmn_xml,omitempty"`

	// Store steps as JSON
	Steps []StepDefinition `json:"steps"`

	CreatedAt time.Time `json:"created_at"`
}

type StepType string

const (
	StepTypeStart StepType = "START"
	StepTypeEnd   StepType = "END"

	// Gateways
	StepTypeGatewayExclusive  StepType = "EXCLUSIVE_GATEWAY"
	StepTypeGatewayInclusive  StepType = "INCLUSIVE_GATEWAY"
	StepTypeGatewayParallel   StepType = "PARALLEL_GATEWAY"
	StepTypeGatewayComplex    StepType = "COMPLEX_GATEWAY"
	StepTypeGatewayEventBased StepType = "EVENT_BASED_GATEWAY"

	// Tasks
	StepTypeTask             StepType = "TASK" // Generic
	StepTypeUserTask         StepType = "USER_TASK"
	StepTypeServiceTask      StepType = "SERVICE_TASK"
	StepTypeScriptTask       StepType = "SCRIPT_TASK"
	StepTypeSendTask         StepType = "SEND_TASK"
	StepTypeReceiveTask      StepType = "RECEIVE_TASK"
	StepTypeManualTask       StepType = "MANUAL_TASK"
	StepTypeBusinessRuleTask StepType = "BUSINESS_RULE_TASK"

	// Sub-Processes
	StepTypeSubProcess      StepType = "SUB_PROCESS"
	StepTypeCallActivity    StepType = "CALL_ACTIVITY"
	StepTypeEventSubProcess StepType = "EVENT_SUB_PROCESS"

	// Events (Catch/Throw)
	StepTypeIntermediateCatchEvent      StepType = "INTERMEDIATE_CATCH_EVENT"
	StepTypeIntermediateTimerCatchEvent StepType = "INTERMEDIATE_TIMER_CATCH_EVENT"
	StepTypeIntermediateThrowEvent      StepType = "INTERMEDIATE_THROW_EVENT"
	StepTypeBoundaryEvent               StepType = "BOUNDARY_EVENT"
)

type StepDefinition struct {
	ID       string       `json:"id"`
	Type     StepType     `json:"type"`
	Name     string       `json:"name"`
	Incoming []string     `json:"incoming"` // List of source step IDs
	Outgoing []Transition `json:"outgoing"`

	// For Service Tasks
	Implementation string `json:"implementation,omitempty"`

	// For Sub-Processes
	SubSteps []StepDefinition `json:"sub_steps,omitempty"`

	// Boundary Events attached to this step
	BoundaryEventRefs []string `json:"boundary_event_refs,omitempty"`

	// Default Sequence Flow (for Gateways)
	DefaultFlow string `json:"default_flow,omitempty"`

	// Specific Properties
	Properties map[string]any `json:"properties,omitempty"`

	// Multi-Instance / Loop Characteristics
	LoopType       string `json:"loop_type,omitempty"`       // "PARALLEL", "SEQUENTIAL", or empty/null
	LoopCollection string `json:"loop_collection,omitempty"` // Context variable name for the collection
	LoopElement    string `json:"loop_element,omitempty"`    // Variable name for the current item

	// Lifecycle Hooks
	TaskListeners []TaskListener `json:"task_listeners,omitempty"`

	// Data Handling
	InputParameters  map[string]string `json:"input_parameters,omitempty"`  // Map local var <- global var/expression
	OutputParameters map[string]string `json:"output_parameters,omitempty"` // Map global var <- local var/expression
}

type TaskListener struct {
	Event          string `json:"event"`          // "create", "assignment", "complete", "delete"
	Implementation string `json:"implementation"` // Handler ID (like Service Task)
}

type Transition struct {
	ID        string `json:"id"`
	TargetRef string `json:"target_ref"`
	Condition string `json:"condition,omitempty"` // BPMN Condition Expression
}

// WorkflowInstance represents a running execution
type WorkflowInstance struct {
	ID         string         `json:"id"`
	WorkflowID string         `json:"workflow_id"`
	Status     WorkflowStatus `json:"status"`

	// Hierarchy for Call Activities
	ParentInstanceID  string `json:"parent_instance_id,omitempty"`
	ParentExecutionID string `json:"parent_execution_id,omitempty"`

	// Token-based execution state
	Executions []Execution `json:"executions"`

	// Store context as JSON in Postgres
	Context map[string]any `json:"context"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Execution struct {
	ID        string    `json:"id"`
	StepID    string    `json:"step_id"`
	Status    string    `json:"status"` // ACTIVE, WAITING, COMPLETED
	ParentID  string    `json:"parent_id,omitempty"`
	StartTime time.Time `json:"start_time"`
	// Engine mapping
	ElementInstanceKey int64 `json:"element_instance_key,omitempty,string"`
}
