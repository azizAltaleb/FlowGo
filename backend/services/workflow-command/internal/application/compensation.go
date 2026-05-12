package application

import (
	"context"
	"github.com/azizAltaleb/flowgo/backend/libs/model"
	"sort"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// TriggerCompensation executes compensation handlers for completed activities within a scope
func (e *Engine) TriggerCompensation(ctx context.Context, instance *model.WorkflowInstance, scopeID string, activityRef string, wf *model.WorkflowDefinition) error {
	// 1. Find the parent scope key
	// If scopeID is empty, use the process instance key (global scope)
	var scopeKey int64
	if scopeID == "" {
		k, _ := strconv.ParseInt(instance.ID, 10, 64)
		scopeKey = k
	} else {
		// Find execution for scopeID
		for _, ex := range instance.Executions {
			if ex.ID == scopeID {
				scopeKey = ex.ElementInstanceKey
				break
			}
		}
	}

	// 2. Identify completed executions in this scope
	// We need to inspect all executions in the instance.
	type completedActivity struct {
		Exec    model.Execution
		EndTime time.Time
		Step    *model.StepDefinition
	}
	var completed []completedActivity

	for _, ex := range instance.Executions {
		if ex.Status != "COMPLETED" {
			continue
		}

		// Check Scope
		// ex.ParentID matches scopeID?
		// If scopeID is root (""), ex.ParentID should be "" or top level?
		// My Execution.ParentID maps to parent execution ID.
		if ex.ParentID != scopeID {
			continue
		}

		// Check if we are targeting a specific activity
		if activityRef != "" && ex.StepID != activityRef {
			continue
		}

		// Find Step Definition
		step := findStep(wf.Steps, ex.StepID)
		if step == nil {
			continue
		}

		// We assume EndTime is approx StartTime if not tracked explicitly in Execution struct (it's not).
		// Wait, Execution struct has StartTime. It doesn't have EndTime.
		// ElementInstance in DB has EndTime.
		// For sorting, StartTime is a decent proxy for sequential flows, but EndTime is better.
		// For MVP, using StartTime for reverse order.
		completed = append(completed, completedActivity{
			Exec:    ex,
			EndTime: ex.StartTime, // Using StartTime as proxy
			Step:    step,
		})
	}

	// 3. Sort by time (Reverse Order for Compensation)
	sort.Slice(completed, func(i, j int) bool {
		return completed[i].EndTime.After(completed[j].EndTime)
	})

	// 4. Execute Compensation Handlers
	for _, activity := range completed {
		// Check for Compensation Boundary Event
		for _, refID := range activity.Step.BoundaryEventRefs {
			boundaryStep := findStep(wf.Steps, refID)
			if boundaryStep == nil || boundaryStep.Type != model.StepTypeBoundaryEvent {
				continue
			}

			// Check if it is a Compensation Event
			// We identify it by presence of "compensate_event_definition" or similar property?
			// Or check if "cancel_activity" is false (compensation usually doesn't cancel, it compensates AFTER completion).
			// Let's assume property "event_type" = "compensation" or existing "is_compensation" flag.
			// Using "event_definition_type" property.
			evtType, _ := boundaryStep.Properties["event_definition_type"].(string)
			if evtType != "compensate" {
				continue
			}

			// Execute the handler linked to this boundary event
			// The handler is the target of the Outgoing transition of the boundary event
			for _, t := range boundaryStep.Outgoing {
				handlerID := t.TargetRef

				// Execute Handler
				// We create a new execution for the compensation handler.
				// It usually runs synchronously.

				handlerExec := model.Execution{
					ID:        uuid.New().String(),
					StepID:    handlerID,
					Status:    "ACTIVE",
					ParentID:  scopeID, // Runs in the same scope
					StartTime: time.Now(),
				}
				handlerKey := generateKey(handlerExec.ID)
				handlerExec.ElementInstanceKey = handlerKey

				// Parse PI Key
				piKey, _ := strconv.ParseInt(instance.ID, 10, 64)

				// Find handler step type
				handlerStep := findStep(wf.Steps, handlerID)
				handlerType := "TASK"
				if handlerStep != nil {
					handlerType = string(handlerStep.Type)
				}

				el := &model.ElementInstance{
					Key:                  handlerKey,
					ProcessInstanceKey:   piKey,
					ProcessDefinitionKey: wf.ID,
					ElementID:            handlerID,
					BpmnElementType:      handlerType,
					FlowScopeKey:         scopeKey,
					State:                "ACTIVATED",
					CreatedAt:            time.Now(),
				}
				e.repo.CreateElementInstance(ctx, el)

				instance.Executions = append(instance.Executions, handlerExec)

				// Run it
				if err := e.autoAdvance(ctx, instance, handlerExec.ID, wf); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
