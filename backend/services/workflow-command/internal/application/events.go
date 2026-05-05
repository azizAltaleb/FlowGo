package application

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"
	"workflow-engine/backend/libs/id"
	"workflow-engine/backend/libs/model"
)

// CheckTimers scans all active instances for timer events that are due
// It processes each timer in its own transaction to ensure isolation.
func (e *Engine) CheckTimers(ctx context.Context) error {
	now := time.Now()
	dueTimers, err := e.repo.ListDueTimers(ctx, now)
	if err != nil {
		return err
	}

	var errs []error
	for _, timer := range dueTimers {
		// Process each timer in its own transaction
		if err := e.withTx(ctx, func(txEngine *Engine) error {
			return txEngine.processTimer(ctx, timer)
		}); err != nil {
			// Log error and continue with next timer
			errs = append(errs, fmt.Errorf("failed to process timer %d: %w", timer.Key, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("encountered errors processing timers: %v", errs)
	}
	return nil
}

func (e *Engine) processTimer(ctx context.Context, timer model.Timer) error {
	// Load Process Instance
	instance, err := e.GetInstance(ctx, fmt.Sprintf("%d", timer.ProcessInstanceKey))
	if err != nil {
		return nil // Skip if instance not found (or log warning)
	}

	// Load Workflow Definition
	wfID, _ := strconv.ParseInt(instance.WorkflowID, 10, 64)
	wf, err := e.getWorkflowDefinition(ctx, wfID)
	if err != nil {
		return nil // Skip if workflow def not found
	}

	// Find the step definition for the timer
	var step *model.StepDefinition
	for _, s := range wf.Steps {
		if s.ID == timer.ElementID {
			step = &s
			break
		}
	}
	if step == nil {
		return nil // Step not found
	}

	piKey := timer.ProcessInstanceKey

	// 1. Intermediate Timer Catch Event
	if step.Type == model.StepTypeIntermediateTimerCatchEvent {
		// Find execution
		var exec *model.Execution
		for i := range instance.Executions {
			if instance.Executions[i].ElementInstanceKey == timer.ElementInstanceKey {
				exec = &instance.Executions[i]
				break
			}
		}

		if exec != nil && exec.Status == "ACTIVE" {
			if err := e.proceedToken(ctx, instance, exec.ID, wf); err == nil {
				// Cancel siblings if this was an Event-Based Gateway
				e.cancelEventGatewaySiblings(ctx, instance, step, exec.ParentID, wf)

				// Mark Timer as Triggered
				timer.State = "TRIGGERED"
				e.repo.UpdateTimer(ctx, &timer)

				// Engine: Check Completion
				if instance.Status == model.StatusCompleted {
					pi := &model.ProcessInstance{
						Key:     piKey,
						State:   "COMPLETED",
						EndTime: time.Now(),
					}
					e.repo.UpdateProcessInstance(ctx, pi)
				}
				// Engine: Persist Variables
				if err := e.persistVariables(ctx, instance.ID, piKey, instance.Context); err != nil {
					return err
				}
			}
		}
	} else if step.Type == model.StepTypeBoundaryEvent {
		// 2. Boundary Timer Event
		// timer.ElementInstanceKey points to the attached activity instance

		// Find the execution for the attached activity
		var exec *model.Execution
		var execIdx int
		found := false
		for i := range instance.Executions {
			if instance.Executions[i].ElementInstanceKey == timer.ElementInstanceKey {
				exec = &instance.Executions[i]
				execIdx = i
				found = true
				break
			}
		}

		if found && exec.Status == "ACTIVE" {
			// Determine interruption
			cancelActivity := true
			if c, ok := step.Properties["cancel_activity"].(bool); ok {
				cancelActivity = c
			}

			if cancelActivity {
				instance.Executions[execIdx].Status = "TERMINATED"
				// Engine: Terminate element instance
				if key := instance.Executions[execIdx].ElementInstanceKey; key != 0 {
					el := &model.ElementInstance{
						Key:     key,
						State:   "TERMINATED",
						EndTime: time.Now(),
					}
					e.repo.UpdateElementInstance(ctx, el)
				}
			}

			// Spawn new token for the boundary outgoing flow
			for _, t := range step.Outgoing {
				newExec := model.Execution{
					ID:        generateRuntimeID(),
					StepID:    t.TargetRef,
					Status:    "ACTIVE",
					ParentID:  exec.ParentID, // Sibling to the task
					StartTime: time.Now(),
				}

				// Engine: Create new Element Instance
				newKey := generateKey(newExec.ID)
				newExec.ElementInstanceKey = newKey

				nextStepType := "TASK"
				for _, s := range wf.Steps {
					if s.ID == t.TargetRef {
						nextStepType = string(s.Type)
						break
					}
				}

				el := &model.ElementInstance{
					Key:                  newKey,
					ID:                   id.GenerateUUIDv7(),
					ProcessInstanceKey:   piKey,
					ProcessDefinitionKey: wf.ID,
					ElementID:            t.TargetRef,
					BpmnElementType:      nextStepType,
					FlowScopeKey:         e.getFlowScopeKey(instance, newExec.ParentID),
					State:                "ACTIVATED",
					CreatedAt:            time.Now(),
				}
				e.repo.CreateElementInstance(ctx, el)

				instance.Executions = append(instance.Executions, newExec)

				// Auto advance the new token
				e.autoAdvance(ctx, instance, newExec.ID, wf)
			}

			// Mark Timer as Triggered
			timer.State = "TRIGGERED"
			e.repo.UpdateTimer(ctx, &timer)

			// Engine: Check Completion
			if instance.Status == model.StatusCompleted {
				pi := &model.ProcessInstance{
					Key:     piKey,
					State:   "COMPLETED",
					EndTime: time.Now(),
				}
				e.repo.UpdateProcessInstance(ctx, pi)
			}

			// Engine: Persist Variables
			if err := e.persistVariables(ctx, instance.ID, piKey, instance.Context); err != nil {
				return err
			}
		}
	}
	return nil
}

// CheckSLAs scans all active jobs for SLA breaches
// Performs updates in individual transactions.
func (e *Engine) CheckSLAs(ctx context.Context) error {
	now := time.Now()
	overdueJobs, err := e.repo.ListOverdueJobs(ctx, now)
	if err != nil {
		return err
	}

	var errs []error
	for _, job := range overdueJobs {
		if err := e.withTx(ctx, func(txEngine *Engine) error {
			// Mark as breached
			job.BreachedAt = &now
			job.UpdatedAt = now

			if err := txEngine.repo.UpdateJob(ctx, &job); err != nil {
				return err
			}

			// TODO: Trigger Escalation (e.g. Email, Notification, Incident)
			// For now, we assume the job simply carries the "Breached" flag.
			// We could also publish an internal event if we had an event bus here.
			return nil
		}); err != nil {
			errs = append(errs, fmt.Errorf("failed to process sla for job %d: %w", job.Key, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors processing SLAs: %v", errs)
	}
	return nil
}

// PublishSignal triggers a signal event across all active instances waiting for it.
func (e *Engine) PublishSignal(ctx context.Context, signalName string, payload map[string]any) error {
	// Signal events don't have persistent subscriptions in this implementation (unlike Messages).
	activeElements, err := e.repo.ListActiveElementInstances(ctx, 0) // 0 means all instances
	if err != nil {
		return err
	}

	triggeredCount := 0
	var errs []error

	for _, el := range activeElements {
		if err := e.withTx(ctx, func(txEngine *Engine) error {
			return txEngine.processSignalForElement(ctx, el, signalName, payload)
		}); err != nil {
			// We only care if it failed *after* matching.
			// The processSignalForElement function should return nil if no match or not relevant.
			// If it returns error, it means a real failure during processing.
			errs = append(errs, fmt.Errorf("failed signal for element %d: %w", el.Key, err))
		} else {
			triggeredCount++
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors during signal publish: %v", errs)
	}
	return nil
}

func (e *Engine) processSignalForElement(ctx context.Context, el model.ElementInstance, signalName string, payload map[string]any) error {
	// Load Process Instance to get Context and WorkflowID
	// We need the full instance to proceedToken.
	instance, err := e.GetInstance(ctx, fmt.Sprintf("%d", el.ProcessInstanceKey))
	if err != nil {
		return nil // Skip if instance not found
	}

	// Load Workflow Definition
	wfID, _ := strconv.ParseInt(instance.WorkflowID, 10, 64)
	wf, err := e.getWorkflowDefinition(ctx, wfID)
	if err != nil {
		return nil // Skip
	}

	// Find Step Definition
	step := findStep(wf.Steps, el.ElementID)
	if step == nil {
		return nil // Skip
	}

	// Check if it's a Signal Catch Event (Intermediate, Boundary, or Start - though Start is handled differently)
	// We focus on Intermediate and Boundary for now.
	if step.Type == model.StepTypeIntermediateCatchEvent || step.Type == model.StepTypeBoundaryEvent {
		if ref, ok := step.Properties["signal_ref"].(string); ok && ref == signalName {
			// Merge payload into context
			if payload != nil {
				if instance.Context == nil {
					instance.Context = make(map[string]any)
				}
				for k, v := range payload {
					instance.Context[k] = v
				}
			}

			// Find the execution corresponding to this element
			// The instance loaded via GetInstance has populated executions.
			var execID string
			for _, ex := range instance.Executions {
				if ex.ElementInstanceKey == el.Key && ex.Status == "ACTIVE" {
					execID = ex.ID
					break
				}
			}

			if execID != "" {
				// Proceed
				if err := e.proceedToken(ctx, instance, execID, wf); err != nil {
					return err
				}

				// Cancel siblings if this was an Event-Based Gateway (handled in checkEventGateway? No, proceedToken advances)
				// If it was attached to an Event Gateway, the gateway logic should handle it?
				// Wait, IntermediateCatchEvent following an EventGateway is just a normal catch.
				// But proceedToken creates NEW tokens for outgoing.
				var exec *model.Execution
				for i := range instance.Executions {
					if instance.Executions[i].ID == execID {
						exec = &instance.Executions[i]
						break
					}
				}
				if exec != nil {
					e.cancelEventGatewaySiblings(ctx, instance, step, exec.ParentID, wf)
				}

				// Engine: Check Completion
				piKey, _ := strconv.ParseInt(instance.ID, 10, 64)
				if instance.Status == model.StatusCompleted {
					pi := &model.ProcessInstance{
						Key:     piKey,
						State:   "COMPLETED",
						EndTime: time.Now(),
					}
					e.repo.UpdateProcessInstance(ctx, pi)
				}

				// Engine: Persist Variables
				if err := e.persistVariables(ctx, instance.ID, piKey, instance.Context); err != nil {
					return err
				}

				return nil // Triggered
			}
		}
	}
	return nil // Not triggered / Not matching
}

// PublishMessage triggers a message event.
func (e *Engine) PublishMessage(ctx context.Context, messageName, correlationKey string, payload map[string]any) error {
	subs, err := e.repo.ListMessageSubscriptions(ctx, messageName, correlationKey)
	if err != nil {
		return err
	}

	var errs []error
	for _, sub := range subs {
		if err := e.withTx(ctx, func(txEngine *Engine) error {
			return txEngine.processMessageSubscription(ctx, sub, payload)
		}); err != nil {
			errs = append(errs, fmt.Errorf("failed message correlation for sub %d: %w", sub.Key, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors during message publish: %v", errs)
	}
	return nil
}

// publishSignal is the internal helper for StepExecutors (uses current transaction)
func (e *Engine) publishSignal(ctx context.Context, signalName string, payload map[string]any) error {
	activeElements, err := e.repo.ListActiveElementInstances(ctx, 0)
	if err != nil {
		return err
	}
	for _, el := range activeElements {
		if err := e.processSignalForElement(ctx, el, signalName, payload); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) processMessageSubscription(ctx context.Context, sub model.MessageSubscription, payload map[string]any) error {
	// Load Process Instance
	instance, err := e.GetInstance(ctx, fmt.Sprintf("%d", sub.ProcessInstanceKey))
	if err != nil {
		return nil // Skip
	}

	// Load Workflow Definition
	wfID, _ := strconv.ParseInt(instance.WorkflowID, 10, 64)
	wf, err := e.getWorkflowDefinition(ctx, wfID)
	if err != nil {
		return nil // Skip
	}

	// Find the step definition
	var step *model.StepDefinition
	for _, s := range wf.Steps {
		if s.ID == sub.ElementID {
			step = &s
			break
		}
	}
	if step == nil {
		return nil // Skip
	}

	piKey := sub.ProcessInstanceKey

	// Merge payload
	if payload != nil {
		if instance.Context == nil {
			instance.Context = make(map[string]any)
		}
		for k, v := range payload {
			instance.Context[k] = v
		}
	}

	// 1. Intermediate Message Catch Event or Receive Task
	if step.Type == model.StepTypeIntermediateCatchEvent || step.Type == model.StepTypeReceiveTask {
		// Find execution
		var exec *model.Execution
		for i := range instance.Executions {
			if instance.Executions[i].ElementInstanceKey == sub.ElementInstanceKey {
				exec = &instance.Executions[i]
				break
			}
		}

		if exec != nil && exec.Status == "ACTIVE" {
			if err := e.proceedToken(ctx, instance, exec.ID, wf); err == nil {
				// Cancel siblings if this was an Event-Based Gateway
				e.cancelEventGatewaySiblings(ctx, instance, step, exec.ParentID, wf)

				// Mark Subscription as Correlated
				sub.State = "CORRELATED"
				e.repo.UpdateMessageSubscription(ctx, &sub)

				// Engine: Check Completion
				if instance.Status == model.StatusCompleted {
					pi := &model.ProcessInstance{
						Key:     piKey,
						State:   "COMPLETED",
						EndTime: time.Now(),
					}
					e.repo.UpdateProcessInstance(ctx, pi)
				}
				// Engine: Persist Variables
				if err := e.persistVariables(ctx, instance.ID, piKey, instance.Context); err != nil {
					return err
				}
			} else {
				return err
			}
		}
	} else if step.Type == model.StepTypeBoundaryEvent {
		// 2. Boundary Message Event
		// sub.ElementInstanceKey points to the attached activity instance

		// Find the execution for the attached activity
		var exec *model.Execution
		var execIdx int
		found := false
		for i := range instance.Executions {
			if instance.Executions[i].ElementInstanceKey == sub.ElementInstanceKey {
				exec = &instance.Executions[i]
				execIdx = i
				found = true
				break
			}
		}

		if found && exec.Status == "ACTIVE" {
			// Determine interruption
			cancelActivity := true
			if c, ok := step.Properties["cancel_activity"].(bool); ok {
				cancelActivity = c
			}

			if cancelActivity {
				instance.Executions[execIdx].Status = "TERMINATED"
				// Engine: Terminate element instance
				if key := instance.Executions[execIdx].ElementInstanceKey; key != 0 {
					el := &model.ElementInstance{
						Key:     key,
						State:   "TERMINATED",
						EndTime: time.Now(),
					}
					e.repo.UpdateElementInstance(ctx, el)
				}
			}

			// Spawn new token for the boundary outgoing flow
			for _, t := range step.Outgoing {
				newExec := model.Execution{
					ID:        generateRuntimeID(),
					StepID:    t.TargetRef,
					Status:    "ACTIVE",
					ParentID:  exec.ParentID, // Sibling to the task
					StartTime: time.Now(),
				}

				// Engine: Create new Element Instance
				newKey := generateKey(newExec.ID)
				newExec.ElementInstanceKey = newKey

				nextStepType := "TASK"
				for _, s := range wf.Steps {
					if s.ID == t.TargetRef {
						nextStepType = string(s.Type)
						break
					}
				}

				el := &model.ElementInstance{
					Key:                  newKey,
					ID:                   id.GenerateUUIDv7(),
					ProcessInstanceKey:   piKey,
					ProcessDefinitionKey: wf.ID,
					ElementID:            t.TargetRef,
					BpmnElementType:      nextStepType,
					FlowScopeKey:         e.getFlowScopeKey(instance, newExec.ParentID),
					State:                "ACTIVATED",
					CreatedAt:            time.Now(),
				}
				e.repo.CreateElementInstance(ctx, el)

				instance.Executions = append(instance.Executions, newExec)

				// Auto advance the new token
				if err := e.autoAdvance(ctx, instance, newExec.ID, wf); err != nil {
					return err
				}
			}

			// Mark Subscription as Correlated
			sub.State = "CORRELATED"
			e.repo.UpdateMessageSubscription(ctx, &sub)

			// Engine: Check Completion
			if instance.Status == model.StatusCompleted {
				pi := &model.ProcessInstance{
					Key:     piKey,
					State:   "COMPLETED",
					EndTime: time.Now(),
				}
				e.repo.UpdateProcessInstance(ctx, pi)
			}

			// Engine: Persist Variables
			if err := e.persistVariables(ctx, instance.ID, piKey, instance.Context); err != nil {
				return err
			}
		}
	}
	return nil
}

// publishMessage is the internal helper for StepExecutors (uses current transaction)
func (e *Engine) publishMessage(ctx context.Context, messageName, correlationKey string, payload map[string]any) error {
	subs, err := e.repo.ListMessageSubscriptions(ctx, messageName, correlationKey)
	if err != nil {
		return err
	}
	for _, sub := range subs {
		if err := e.processMessageSubscription(ctx, sub, payload); err != nil {
			return err
		}
	}
	return nil
}

// Simple ISO8601 Duration Parser (PT#H#M#S)
func parseISO8601Duration(iso string) (time.Duration, error) {
	// Very basic implementation: Convert PT1H2M3S -> 1h2m3s for time.ParseDuration
	// This is a hack for MVP. A proper parser would handle P1Y2M...
	if len(iso) < 2 || iso[:2] != "PT" {
		return 0, fmt.Errorf("unsupported duration format (must start with PT): %s", iso)
	}

	s := iso[2:]
	// Go's time.ParseDuration is compatible with the time part of ISO8601 (H, M, S)
	// assuming lower case for parseDuration? No, ParseDuration expects "1h2m3s".
	// ISO uses upper case.
	// We can simply try to lower case it?
	// 1H -> 1h
	// But ParseDuration doesn't support Y, W, D (which are in the Date part P...T...)
	// Since we strip PT, we are left with e.g. 5M.

	// Lowercase the string
	lower := ""
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			lower += string(r + 32)
		} else {
			lower += string(r)
		}
	}
	return time.ParseDuration(lower)
}

// handleBoundaryError checks if an error can be handled by an attached Boundary Error Event
func (e *Engine) handleBoundaryError(ctx context.Context, instance *model.WorkflowInstance, step *model.StepDefinition, execID string, wf *model.WorkflowDefinition, err error) (bool, error) {
	if len(step.BoundaryEventRefs) == 0 {
		return false, err
	}

	for _, refID := range step.BoundaryEventRefs {
		// Find boundary step def
		var boundaryStep *model.StepDefinition
		for _, s := range wf.Steps {
			if s.ID == refID {
				boundaryStep = &s
				break
			}
		}

		if boundaryStep == nil {
			continue
		}
		if boundaryStep.Type != model.StepTypeBoundaryEvent {
			continue
		}

		// Check if it is an Error Event
		if _, ok := boundaryStep.Properties["error_ref"]; ok {
			// Check Error Code Matching
			// We only handle BpmnError here. System errors usually raise incidents unless we want a "catch all" for system errors too?
			// Standard BPMN: Error Events catch "Business Errors".
			var bpmnErr *BpmnError
			if errors.As(err, &bpmnErr) {
				errorCode, _ := boundaryStep.Properties["error_code"].(string)
				// If boundary event has no error code, it catches ALL errors.
				// If it has a code, it must match.
				if errorCode != "" && errorCode != bpmnErr.ErrorCode {
					continue // No match
				}
			} else {
				// Error is not a BpmnError (it's a system error).
				// Should we catch it?
				// If the boundary event has NO error code, maybe it catches everything?
				// Usually "error_code" is required for specific errors.
				// For now, let's say we ONLY catch BpmnError.
				continue
			}

			// 1. Determine Interruption
			cancelActivity := true // Default for Error Boundary Events
			if c, ok := boundaryStep.Properties["cancel_activity"].(bool); ok {
				cancelActivity = c
			}

			var currentParentID string

			// 2. Update status of the failed execution
			for i := range instance.Executions {
				if instance.Executions[i].ID == execID {
					currentParentID = instance.Executions[i].ParentID
					if cancelActivity {
						instance.Executions[i].Status = "TERMINATED" // Or FAILED/COMPLETED? TERMINATED implies interruption.
						// Engine: Terminate element instance
						if key := instance.Executions[i].ElementInstanceKey; key != 0 {
							el := &model.ElementInstance{
								Key:     key,
								State:   "TERMINATED",
								EndTime: time.Now(),
							}
							if err := e.repo.UpdateElementInstance(ctx, el); err != nil {
								// log error?
							}
						}
					}
					break
				}
			}

			// Parse ProcessInstanceKey
			piKey, _ := strconv.ParseInt(instance.ID, 10, 64)

			// 3. Spawn new token for the boundary outgoing flow
			for _, t := range boundaryStep.Outgoing {
				newExec := model.Execution{
					ID:        generateRuntimeID(),
					StepID:    t.TargetRef,
					Status:    "ACTIVE",
					ParentID:  currentParentID, // Sibling to the failed task
					StartTime: time.Now(),
				}

				// Engine: Create new Element Instance
				newKey := generateKey(newExec.ID)
				newExec.ElementInstanceKey = newKey

				nextStepType := "TASK"
				for _, s := range wf.Steps {
					if s.ID == t.TargetRef {
						nextStepType = string(s.Type)
						break
					}
				}

				el := &model.ElementInstance{
					Key:                  newKey,
					ProcessInstanceKey:   piKey,
					ProcessDefinitionKey: wf.ID,
					ElementID:            t.TargetRef,
					BpmnElementType:      nextStepType,
					FlowScopeKey:         e.getFlowScopeKey(instance, newExec.ParentID),
					State:                "ACTIVATED",
					CreatedAt:            time.Now(),
				}
				if err := e.repo.CreateElementInstance(ctx, el); err != nil {
					// log error?
				}

				instance.Executions = append(instance.Executions, newExec)

				// Auto advance the new token
				if err := e.autoAdvance(ctx, instance, newExec.ID, wf); err != nil {
					return true, err
				}
			}

			// Engine: Check Completion
			if instance.Status == model.StatusCompleted {
				if piKey, err := strconv.ParseInt(instance.ID, 10, 64); err == nil {
					pi := &model.ProcessInstance{
						Key:     piKey,
						State:   "COMPLETED",
						EndTime: time.Now(),
					}
					if err := e.repo.UpdateProcessInstance(ctx, pi); err != nil {
						// log error?
					}
				}
			}

			return true, nil
		}
	}

	return false, err
}
