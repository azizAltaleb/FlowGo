package application

import (
	"context"
	"errors"
	"fmt"
	"github.com/artificialflow/artificialflow/backend/libs/id"
	"github.com/artificialflow/artificialflow/backend/libs/model"
	"strconv"
	"time"
)

// CheckTimers scans all active instances for timer events that are due
// It processes each timer in its own transaction to ensure isolation.
func (e *Engine) CheckTimers(ctx context.Context) error {
	// Schedule definition-level timer start events (ISO-8601 PT… durations) before firing due timers.
	if err := e.ensureTimerStartSchedules(ctx); err != nil {
		return err
	}

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
	// Also evaluate conditional catches/starts on the same scheduler tick.
	if err := e.CheckConditionals(ctx); err != nil {
		return err
	}
	return nil
}

// ensureTimerStartSchedules creates CREATED timers for process definitions
// whose start event is a timer (duration PT…, absolute timeDate, or timeCycle R[n]/PT…).
// Convention: ProcessInstanceKey=0, ElementInstanceKey=process definition key.
func (e *Engine) ensureTimerStartSchedules(ctx context.Context) error {
	defs, err := e.listDeployedWorkflowDefinitions(ctx)
	if err != nil || len(defs) == 0 {
		return err
	}
	now := time.Now()
	for _, wf := range defs {
		start := firstStartStep(wf.Steps)
		if start == nil || start.Properties == nil {
			continue
		}
		if start.Properties["event_definition_type"] != "timer" {
			continue
		}
		sched, err := resolveTimerSchedule(start.Properties, now)
		if err != nil {
			continue
		}
		key := generateDeterministicKey(fmt.Sprintf("timer_start_%d_%s", wf.ID, start.ID))
		if existing, err := e.repo.GetTimer(ctx, key); err == nil && existing != nil {
			// Already scheduled or previously completed (one-shot / exhausted cycle).
			continue
		}
		timer := &model.Timer{
			Key:                key,
			ID:                 id.GenerateUUIDv7(),
			ElementInstanceKey: wf.ID,
			ProcessInstanceKey: 0,
			ElementID:          start.ID,
			DueDate:            sched.DueDate,
			RepeatCount:        sched.RepeatCount,
			State:              "CREATED",
			CreatedAt:          now,
		}
		if err := e.repo.CreateTimer(ctx, timer); err != nil {
			return err
		}
	}
	return nil
}

// finalizeTimerAfterFire marks a timer TRIGGERED or re-arms it for timeCycle.
func (e *Engine) finalizeTimerAfterFire(ctx context.Context, timer *model.Timer, props map[string]any) error {
	interval := cycleIntervalFromProps(props)
	if timer.RepeatCount < 0 {
		// Infinite cycle
		if interval <= 0 {
			timer.State = "TRIGGERED"
			return e.repo.UpdateTimer(ctx, timer)
		}
		timer.DueDate = time.Now().Add(interval)
		timer.State = "CREATED"
		return e.repo.UpdateTimer(ctx, timer)
	}
	if timer.RepeatCount > 1 && interval > 0 {
		timer.RepeatCount--
		timer.DueDate = time.Now().Add(interval)
		timer.State = "CREATED"
		return e.repo.UpdateTimer(ctx, timer)
	}
	timer.State = "TRIGGERED"
	timer.RepeatCount = 0
	return e.repo.UpdateTimer(ctx, timer)
}

func (e *Engine) cancelCreatedTimersForElementInstance(ctx context.Context, elementInstanceKey int64) {
	if elementInstanceKey == 0 {
		return
	}
	timers, err := e.repo.ListCreatedTimersByElementInstanceKey(ctx, elementInstanceKey)
	if err != nil {
		return
	}
	for i := range timers {
		timers[i].State = "CANCELED"
		_ = e.repo.UpdateTimer(ctx, &timers[i])
	}
}

func (e *Engine) processTimer(ctx context.Context, timer model.Timer) error {
	// Definition-level timer start (no process instance yet).
	if timer.ProcessInstanceKey == 0 {
		return e.processTimerStartDefinition(ctx, timer)
	}

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

	// Find the step definition for the timer (including nested event sub-process starts).
	step := findStep(wf.Steps, timer.ElementID)
	if step == nil {
		return nil // Step not found
	}

	piKey := timer.ProcessInstanceKey

	// Timer event sub-process start (triggeredByEvent).
	if step.Type == model.StepTypeStart && step.Properties != nil && step.Properties["event_definition_type"] == "timer" {
		if err := e.tryStartEventSubProcesses(ctx, instance, wf, "timer", "", nil); err != nil {
			return err
		}
		if err := e.finalizeTimerAfterFire(ctx, &timer, step.Properties); err != nil {
			return err
		}
		if instance.Status == model.StatusCompleted {
			pi := &model.ProcessInstance{
				Key:     piKey,
				State:   "COMPLETED",
				EndTime: time.Now(),
			}
			_ = e.repo.UpdateProcessInstance(ctx, pi)
		}
		return nil
	}

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

				// Intermediate catch consumes the token — one-shot even if cycle was modeled.
				timer.State = "TRIGGERED"
				_ = e.repo.UpdateTimer(ctx, &timer)

				// Engine: Check Completion
				if instance.Status == model.StatusCompleted {
					pi := &model.ProcessInstance{
						Key:     piKey,
						State:   "COMPLETED",
						EndTime: time.Now(),
					}
					e.repo.UpdateProcessInstance(ctx, pi)
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

			if cancelActivity {
				timer.State = "TRIGGERED"
				_ = e.repo.UpdateTimer(ctx, &timer)
				// Cancel other boundary timers attached to the same activity.
				e.cancelCreatedTimersForElementInstance(ctx, timer.ElementInstanceKey)
			} else if err := e.finalizeTimerAfterFire(ctx, &timer, step.Properties); err != nil {
				return err
			}

			// Engine: Check Completion
			if instance.Status == model.StatusCompleted {
				pi := &model.ProcessInstance{
					Key:     piKey,
					State:   "COMPLETED",
					EndTime: time.Now(),
				}
				e.repo.UpdateProcessInstance(ctx, pi)
			}

		}
	}
	return nil
}

// processTimerStartDefinition starts a process instance when a definition-level
// timer start is due. ElementInstanceKey holds the process definition key.
func (e *Engine) processTimerStartDefinition(ctx context.Context, timer model.Timer) error {
	defKey := timer.ElementInstanceKey
	if defKey == 0 {
		timer.State = "TRIGGERED"
		_ = e.repo.UpdateTimer(ctx, &timer)
		return nil
	}
	wf, err := e.getWorkflowDefinition(ctx, defKey)
	if err != nil || wf == nil {
		timer.State = "TRIGGERED"
		_ = e.repo.UpdateTimer(ctx, &timer)
		return nil
	}
	start := firstStartStep(wf.Steps)
	if start == nil || start.ID != timer.ElementID {
		timer.State = "TRIGGERED"
		_ = e.repo.UpdateTimer(ctx, &timer)
		return nil
	}
	if start.Properties == nil || start.Properties["event_definition_type"] != "timer" {
		timer.State = "TRIGGERED"
		_ = e.repo.UpdateTimer(ctx, &timer)
		return nil
	}
	if _, err := e.createAndStartInstance(ctx, strconv.FormatInt(defKey, 10), nil, "", "", nil); err != nil {
		return err
	}
	return e.finalizeTimerAfterFire(ctx, &timer, start.Properties)
}

// scheduleEventSubProcessTimers creates timers for timer starts
// inside triggeredByEvent event sub-processes on a newly started instance.
func (e *Engine) scheduleEventSubProcessTimers(ctx context.Context, instance *model.WorkflowInstance, wf *model.WorkflowDefinition) error {
	if instance == nil || wf == nil {
		return nil
	}
	piKey, err := strconv.ParseInt(instance.ID, 10, 64)
	if err != nil || piKey == 0 {
		return nil
	}
	now := time.Now()
	for i := range wf.Steps {
		sp := &wf.Steps[i]
		if sp.Type != model.StepTypeEventSubProcess {
			continue
		}
		var start *model.StepDefinition
		for j := range sp.SubSteps {
			if sp.SubSteps[j].Type == model.StepTypeStart {
				start = &sp.SubSteps[j]
				break
			}
		}
		if start == nil || start.Properties == nil {
			continue
		}
		if start.Properties["event_definition_type"] != "timer" {
			continue
		}
		sched, err := resolveTimerSchedule(start.Properties, now)
		if err != nil {
			continue
		}
		key := generateDeterministicKey(fmt.Sprintf("esp_timer_%d_%s", piKey, start.ID))
		if existing, err := e.repo.GetTimer(ctx, key); err == nil && existing != nil {
			continue
		}
		timer := &model.Timer{
			Key:                key,
			ID:                 id.GenerateUUIDv7(),
			ElementInstanceKey: 0,
			ProcessInstanceKey: piKey,
			ElementID:          start.ID,
			DueDate:            sched.DueDate,
			RepeatCount:        sched.RepeatCount,
			State:              "CREATED",
			CreatedAt:          now,
		}
		if err := e.repo.CreateTimer(ctx, timer); err != nil {
			return err
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
		jobCopy := job
		if err := e.withTx(ctx, func(txEngine *Engine) error {
			// Mark as breached (ListOverdueJobs only returns jobs with BreachedAt unset).
			jobCopy.BreachedAt = &now
			jobCopy.UpdatedAt = now

			if err := txEngine.repo.UpdateJob(ctx, &jobCopy); err != nil {
				return err
			}

			incident := &model.Incident{
				Key:                generateDeterministicKey(fmt.Sprintf("sla_breach_%d", jobCopy.Key)),
				ID:                 id.GenerateUUIDv7(),
				ProcessInstanceKey: jobCopy.ProcessInstanceKey,
				ElementInstanceKey: jobCopy.ElementInstanceKey,
				JobKey:             jobCopy.Key,
				ErrorType:          "SLA_DUE_DATE_BREACHED",
				ErrorMessage:       fmt.Sprintf("user-task due date breached for element %s", jobCopy.ElementID),
				State:              "CREATED",
				CreatedAt:          now,
			}
			if err := txEngine.repo.CreateIncident(ctx, incident); err != nil {
				return err
			}

			payload := map[string]any{
				"processInstanceKey": jobCopy.ProcessInstanceKey,
				"jobKey":             jobCopy.Key,
				"elementId":          jobCopy.ElementID,
				"incidentKey":        incident.Key,
			}
			// Use non-wrapping publish so we stay in the current transaction.
			return txEngine.publishEscalation(ctx, "user-task.due-date.breached", payload)
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
	activeElements, err := e.repo.ListActiveElementInstancesByTypes(ctx, 0, signalCandidateElementTypes()) // 0 means all instances
	if err != nil {
		return err
	}

	definitions := make(map[int64]*model.WorkflowDefinition)
	var errs []error

	for _, el := range activeElements {
		if !e.signalElementCanMatch(ctx, el, signalName, definitions) {
			continue
		}
		if err := e.withTx(ctx, func(txEngine *Engine) error {
			return txEngine.processSignalForElement(ctx, el, signalName, payload, definitions)
		}); err != nil {
			// We only care if it failed *after* matching.
			// The processSignalForElement function should return nil if no match or not relevant.
			// If it returns error, it means a real failure during processing.
			errs = append(errs, fmt.Errorf("failed signal for element %d: %w", el.Key, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors during signal publish: %v", errs)
	}
	// Also start process definitions with matching signal start events.
	if err := e.startInstancesForSignalStart(ctx, signalName, payload); err != nil {
		return err
	}
	// Interrupt/start event sub-processes waiting for this signal on active instances.
	if err := e.triggerEventSubProcessesOnActiveInstances(ctx, "signal", signalName, payload); err != nil {
		return err
	}
	return nil
}

func (e *Engine) processSignalForElement(ctx context.Context, el model.ElementInstance, signalName string, payload map[string]any, definitions map[int64]*model.WorkflowDefinition) error {
	if !isSignalCandidateElementType(el.BpmnElementType) {
		return nil
	}

	var instance *model.WorkflowInstance
	wf, step, matched := e.matchSignalStep(ctx, el, signalName, definitions)
	if !matched {
		wf, activity, boundary, boundaryMatched := e.matchSignalBoundaryOnActivity(ctx, el, signalName, definitions)
		if boundaryMatched {
			var err error
			instance, err = e.GetInstance(ctx, fmt.Sprintf("%d", el.ProcessInstanceKey))
			if err != nil {
				return nil
			}
			if payload != nil {
				if instance.Context == nil {
					instance.Context = make(map[string]any)
				}
				for k, v := range payload {
					instance.Context[k] = v
				}
			}
			if err := e.triggerBoundaryPath(ctx, instance, boundary, activity.ID, wf); err != nil {
				return err
			}
			piKey, _ := strconv.ParseInt(instance.ID, 10, 64)
			if instance.Status == model.StatusCompleted {
				pi := &model.ProcessInstance{
					Key:     piKey,
					State:   "COMPLETED",
					EndTime: time.Now(),
				}
				_ = e.repo.UpdateProcessInstance(ctx, pi)
			}
			if len(payload) > 0 {
				if err := e.persistVariables(ctx, instance.ID, piKey, payload); err != nil {
					return err
				}
			}
			return nil
		}

		if el.ProcessDefinitionKey != 0 {
			return nil
		}

		var err error
		instance, err = e.GetInstance(ctx, fmt.Sprintf("%d", el.ProcessInstanceKey))
		if err != nil {
			return nil // Skip if instance not found
		}

		wfID, err := strconv.ParseInt(instance.WorkflowID, 10, 64)
		if err != nil {
			return nil
		}
		wf, step, matched = e.matchSignalStepForProcess(ctx, wfID, el.ElementID, signalName, definitions)
		if !matched {
			return nil
		}
	}

	if instance == nil {
		var err error
		instance, err = e.GetInstance(ctx, fmt.Sprintf("%d", el.ProcessInstanceKey))
		if err != nil {
			return nil // Skip if instance not found
		}
	}

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

		// Engine: Persist only the correlated signal payload.
		if len(payload) > 0 {
			if err := e.persistVariables(ctx, instance.ID, piKey, payload); err != nil {
				return err
			}
		}

		return nil // Triggered
	}
	return nil // Not triggered / Not matching
}

func signalCandidateElementTypes() []string {
	types := []string{
		string(model.StepTypeIntermediateCatchEvent),
		string(model.StepTypeBoundaryEvent),
		"", // Preserve older rows created before element type was consistently populated.
	}
	return append(types, signalActivityElementTypes()...)
}

func signalActivityElementTypes() []string {
	return []string{
		string(model.StepTypeUserTask),
		string(model.StepTypeServiceTask),
		string(model.StepTypeScriptTask),
		string(model.StepTypeReceiveTask),
		string(model.StepTypeManualTask),
		string(model.StepTypeBusinessRuleTask),
		string(model.StepTypeSendTask),
		string(model.StepTypeCallActivity),
		string(model.StepTypeSubProcess),
	}
}

func isSignalCandidateElementType(elementType string) bool {
	if elementType == "" {
		return true
	}
	if elementType == string(model.StepTypeIntermediateCatchEvent) || elementType == string(model.StepTypeBoundaryEvent) {
		return true
	}
	return isSignalActivityElementType(elementType)
}

func isSignalActivityElementType(elementType string) bool {
	for _, t := range signalActivityElementTypes() {
		if elementType == t {
			return true
		}
	}
	return false
}

func signalStepMatches(step *model.StepDefinition, signalName string) bool {
	if step == nil || step.Properties == nil {
		return false
	}
	ref, _ := step.Properties["signal_ref"].(string)
	if ref == "" || ref != signalName {
		return false
	}
	if edt, ok := step.Properties["event_definition_type"].(string); ok && edt != "" && edt != "signal" {
		return false
	}
	return true
}

func (e *Engine) signalElementCanMatch(ctx context.Context, el model.ElementInstance, signalName string, definitions map[int64]*model.WorkflowDefinition) bool {
	if !isSignalCandidateElementType(el.BpmnElementType) {
		return false
	}
	if el.ProcessDefinitionKey == 0 {
		return true
	}
	if _, _, matched := e.matchSignalStep(ctx, el, signalName, definitions); matched {
		return true
	}
	_, _, _, matched := e.matchSignalBoundaryOnActivity(ctx, el, signalName, definitions)
	return matched
}

func (e *Engine) matchSignalStep(ctx context.Context, el model.ElementInstance, signalName string, definitions map[int64]*model.WorkflowDefinition) (*model.WorkflowDefinition, *model.StepDefinition, bool) {
	if el.ProcessDefinitionKey == 0 {
		return nil, nil, false
	}
	return e.matchSignalStepForProcess(ctx, el.ProcessDefinitionKey, el.ElementID, signalName, definitions)
}

func (e *Engine) matchSignalStepForProcess(ctx context.Context, processDefinitionKey int64, elementID, signalName string, definitions map[int64]*model.WorkflowDefinition) (*model.WorkflowDefinition, *model.StepDefinition, bool) {
	wf, err := e.getWorkflowDefinitionFromMemo(ctx, processDefinitionKey, definitions)
	if err != nil {
		return nil, nil, false
	}

	step := findStepDeep(wf.Steps, elementID)
	if step == nil {
		return nil, nil, false
	}
	if step.Type != model.StepTypeIntermediateCatchEvent && step.Type != model.StepTypeBoundaryEvent {
		return nil, nil, false
	}
	if !signalStepMatches(step, signalName) {
		return nil, nil, false
	}
	return wf, step, true
}

// matchSignalBoundaryOnActivity finds a signal boundary attached to an active activity instance.
// Boundary events are not activated as their own element instances — they attach to activities.
func (e *Engine) matchSignalBoundaryOnActivity(ctx context.Context, el model.ElementInstance, signalName string, definitions map[int64]*model.WorkflowDefinition) (*model.WorkflowDefinition, *model.StepDefinition, *model.StepDefinition, bool) {
	if el.ProcessDefinitionKey == 0 {
		return nil, nil, nil, false
	}
	if el.BpmnElementType != "" && !isSignalActivityElementType(el.BpmnElementType) {
		return nil, nil, nil, false
	}
	wf, err := e.getWorkflowDefinitionFromMemo(ctx, el.ProcessDefinitionKey, definitions)
	if err != nil {
		return nil, nil, nil, false
	}
	activity := findStepDeep(wf.Steps, el.ElementID)
	if activity == nil || len(activity.BoundaryEventRefs) == 0 {
		return nil, nil, nil, false
	}
	for _, ref := range activity.BoundaryEventRefs {
		boundary := findStepDeep(wf.Steps, ref)
		if boundary == nil || boundary.Type != model.StepTypeBoundaryEvent {
			continue
		}
		if signalStepMatches(boundary, signalName) {
			return wf, activity, boundary, true
		}
	}
	return nil, nil, nil, false
}

func (e *Engine) getWorkflowDefinitionFromMemo(ctx context.Context, processDefinitionKey int64, definitions map[int64]*model.WorkflowDefinition) (*model.WorkflowDefinition, error) {
	if definitions != nil {
		if wf, ok := definitions[processDefinitionKey]; ok {
			return wf, nil
		}
	}

	wf, err := e.getWorkflowDefinition(ctx, processDefinitionKey)
	if err != nil {
		return nil, err
	}
	if definitions != nil {
		definitions[processDefinitionKey] = wf
	}
	return wf, nil
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
	// Start definitions whose start event matches this message (message start).
	if err := e.startInstancesForMessageStart(ctx, messageName, correlationKey, payload); err != nil {
		return err
	}
	// Interrupt/start event sub-processes waiting for this message on active instances.
	if err := e.triggerEventSubProcessesOnActiveInstances(ctx, "message", messageName, payload); err != nil {
		return err
	}
	return nil
}

func (e *Engine) triggerEventSubProcessesOnActiveInstances(ctx context.Context, eventKind, eventRef string, payload map[string]any) error {
	pis, err := e.repo.ListActiveProcessInstances(ctx)
	if err != nil || len(pis) == 0 {
		return nil
	}
	for _, pi := range pis {
		if pi == nil {
			continue
		}
		if err := e.withTx(ctx, func(tx *Engine) error {
			inst, err := tx.GetInstance(ctx, fmt.Sprintf("%d", pi.Key))
			if err != nil {
				return nil
			}
			wfID, _ := strconv.ParseInt(inst.WorkflowID, 10, 64)
			wf, err := tx.getWorkflowDefinition(ctx, wfID)
			if err != nil {
				return nil
			}
			return tx.tryStartEventSubProcesses(ctx, inst, wf, eventKind, eventRef, payload)
		}); err != nil {
			return err
		}
	}
	return nil
}

// publishSignal is the internal helper for StepExecutors (uses current transaction)
func (e *Engine) publishSignal(ctx context.Context, signalName string, payload map[string]any) error {
	activeElements, err := e.repo.ListActiveElementInstancesByTypes(ctx, 0, signalCandidateElementTypes())
	if err != nil {
		return err
	}
	definitions := make(map[int64]*model.WorkflowDefinition)
	for _, el := range activeElements {
		if err := e.processSignalForElement(ctx, el, signalName, payload, definitions); err != nil {
			return err
		}
	}
	if err := e.startInstancesForSignalStart(ctx, signalName, payload); err != nil {
		return err
	}
	return e.triggerEventSubProcessesOnActiveInstances(ctx, "signal", signalName, payload)
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
				// Engine: Persist only the correlated message payload.
				if len(payload) > 0 {
					if err := e.persistVariables(ctx, instance.ID, piKey, payload); err != nil {
						return err
					}
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

			// Engine: Persist only the correlated message payload.
			if len(payload) > 0 {
				if err := e.persistVariables(ctx, instance.ID, piKey, payload); err != nil {
					return err
				}
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
