package application

import (
	"context"
	"workflow-engine/backend/libs/model"
)

// StepExecutor defines the strategy interface for executing BPMN steps
type StepExecutor interface {
	Execute(ctx context.Context, e *Engine, instance *model.WorkflowInstance, step *model.StepDefinition, execID string, wf *model.WorkflowDefinition) error
}
