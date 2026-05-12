package application

import (
	"context"
	"github.com/azizAltaleb/flowgo/backend/libs/model"
)

// StepExecutor defines the strategy interface for executing BPMN steps
type StepExecutor interface {
	Execute(ctx context.Context, e *Engine, instance *model.WorkflowInstance, step *model.StepDefinition, execID string, wf *model.WorkflowDefinition) error
}
