package application

import (
	"context"
	"fmt"
	"github.com/azizAltaleb/flowgo/backend/libs/logger"
	"github.com/azizAltaleb/flowgo/backend/libs/model"
)

// executeTaskListeners runs registered listeners for a specific lifecycle event (create, complete, etc.)
func (e *Engine) executeTaskListeners(ctx context.Context, instance *model.WorkflowInstance, step *model.StepDefinition, eventType string) error {
	if len(step.TaskListeners) == 0 {
		return nil
	}

	log := logger.New("task-listener")

	for _, listener := range step.TaskListeners {
		if listener.Event == eventType {
			handlerID := listener.Implementation
			if handler, ok := e.handlers[handlerID]; ok {
				log.Info(ctx, "executing task listener", map[string]any{
					"instance_id": instance.ID,
					"step_id":     step.ID,
					"event":       eventType,
					"handler":     handlerID,
				})

				// Execute the handler
				// Note: Listeners typically shouldn't block or fail the task if they fail,
				// but in some engines they do. We'll log error and proceed for now,
				// or fail if critical? Let's propagate error for now to allow strict validation.
				if err := handler(ctx, instance, step); err != nil {
					log.Error(ctx, "task listener failed", map[string]any{
						"error":   err.Error(),
						"handler": handlerID,
					})
					return fmt.Errorf("task listener %s failed: %v", handlerID, err)
				}
			} else {
				log.Warn(ctx, "task listener handler not found", map[string]any{
					"handler": handlerID,
				})
			}
		}
	}
	return nil
}
