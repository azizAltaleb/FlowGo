package application

import (
	"context"
	"github.com/azizAltaleb/flowgo/backend/libs/model"
	"strings"
)

// applyInputMappings maps global/parent context variables to local task variables
func (e *Engine) applyInputMappings(ctx context.Context, instance *model.WorkflowInstance, step *model.StepDefinition, execID string) error {
	if len(step.InputParameters) == 0 {
		return nil
	}

	updates := make(map[string]any)

	for target, source := range step.InputParameters {
		// 1. Check if Source is a Variable
		if val, ok := instance.Context[source]; ok {
			updates[target] = val
			continue
		}

		// 2. Check for Expression (Simple ${var})
		if strings.HasPrefix(source, "${") && strings.HasSuffix(source, "}") {
			varName := source[2 : len(source)-1]
			if val, ok := instance.Context[varName]; ok {
				updates[target] = val
				continue
			}
			// If var not found, skip or set nil?
			// Let's set nil for now to indicate missing value
			updates[target] = nil
			continue
		}

		// 3. Treat as Literal Value (String)
		updates[target] = source
	}

	if len(updates) > 0 {
		// Apply to instance context
		for k, v := range updates {
			instance.Context[k] = v
		}

		// Persist mapped variables
		// We use the execution's ElementInstanceKey as the scope if possible,
		// but since we don't strictly enforce local scopes for all tasks yet,
		// we might be writing to global scope if we use instance.ID.
		// However, to support "local" inputs properly, we should scope them to the execution.

		// Find execution to get scope key
		var scopeKey int64
		for _, ex := range instance.Executions {
			if ex.ID == execID {
				scopeKey = ex.ElementInstanceKey
				break
			}
		}

		// Fallback to process instance key if execution not found (shouldn't happen)
		if scopeKey == 0 {
			// parse instance ID
			// piKey, _ := strconv.ParseInt(instance.ID, 10, 64)
			// scopeKey = piKey
			// Actually let's just allow persistVariables to handle it.
			// But wait, persistVariables takes a scopeKey.
		}

		// Since persistVariables is attached to Engine, we need to import or assume it exists.
		// It is in engine.go.

		// Let's rely on instance.Context update for immediate execution usage.
		// And persist so they survive restarts.
		// For now, mapping inputs implies creating them in the current scope.
		if err := e.persistVariables(ctx, instance.ID, scopeKey, updates); err != nil {
			return err
		}
	}

	return nil
}

// applyOutputMappings maps local task variables back to global/parent context
func (e *Engine) applyOutputMappings(ctx context.Context, instance *model.WorkflowInstance, step *model.StepDefinition, execID string) error {
	if len(step.OutputParameters) == 0 {
		return nil
	}

	updates := make(map[string]any)

	for target, source := range step.OutputParameters {
		// Source is the local variable name
		if val, ok := instance.Context[source]; ok {
			updates[target] = val
		}
	}

	if len(updates) > 0 {
		// Apply updates
		for k, v := range updates {
			instance.Context[k] = v
		}

		// Persist to Parent Scope?
		// Usually Output Mapping propagates UP.
		// So we should find the Parent Scope Key.
		// For a simple task, parent is the Flow Scope (Process Instance or SubProcess).

		// Find execution
		var parentScopeKey int64
		for _, ex := range instance.Executions {
			if ex.ID == execID {
				// We need the FlowScopeKey, not the ElementInstanceKey of the task itself.
				// But Execution struct doesn't have FlowScopeKey directly, it has ParentID.
				// We can look up the parent execution.
				if ex.ParentID != "" {
					// Find parent exec
					parentScopeKey = e.getFlowScopeKey(instance, ex.ParentID)
				} else {
					// Root scope
					parentScopeKey = e.getFlowScopeKey(instance, "")
				}
				break
			}
		}

		if err := e.persistVariables(ctx, instance.ID, parentScopeKey, updates); err != nil {
			return err
		}
	}

	return nil
}
