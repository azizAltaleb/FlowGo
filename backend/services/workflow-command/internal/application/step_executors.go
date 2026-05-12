package application

import (
	"context"
	"errors"
	"fmt"
	pb "github.com/azizAltaleb/flowgo/backend/api/v1/go"
	"github.com/azizAltaleb/flowgo/backend/libs/id"
	"github.com/azizAltaleb/flowgo/backend/libs/logger"
	"github.com/azizAltaleb/flowgo/backend/libs/model"
	"strconv"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

type ServiceTaskExecutor struct{}

func (x *ServiceTaskExecutor) Execute(ctx context.Context, e *Engine, instance *model.WorkflowInstance, step *model.StepDefinition, execID string, wf *model.WorkflowDefinition) error {
	// Find execution to get ElementInstanceKey
	var exec *model.Execution
	for i := range instance.Executions {
		if instance.Executions[i].ID == execID {
			exec = &instance.Executions[i]
			break
		}
	}
	if exec == nil {
		return fmt.Errorf("execution %s not found in ServiceTaskExecutor", execID)
	}

	handlerID := step.Implementation
	if handlerID == "" {
		handlerID = step.ID // Fallback to Step ID
	}

	// Engine: Manage Job
	// Use ElementInstanceKey to ensure uniqueness per activation (important for loops)
	jobKey := generateKey(fmt.Sprintf("%s_%d_job", execID, exec.ElementInstanceKey))
	var job *model.Job

	// Try to get existing job to handle retries correctly
	if existingJob, err := e.repo.GetJob(ctx, jobKey); err == nil {
		job = existingJob
	} else {
		// Parse ProcessInstanceKey
		piKey, _ := strconv.ParseInt(instance.ID, 10, 64)

		// Create new job
		job = &model.Job{
			Key:                  jobKey,
			ID:                   id.GenerateUUIDv7(),
			Type:                 handlerID,
			ProcessInstanceKey:   piKey,
			ElementInstanceKey:   exec.ElementInstanceKey,
			ProcessDefinitionKey: wf.ID,
			ElementID:            step.ID,
			Worker:               "",
			Retries:              3, // Default retries
			State:                "CREATED",
			CreatedAt:            time.Now(),
			UpdatedAt:            time.Now(),
		}
		if err := e.repo.CreateJob(ctx, job); err != nil {
			// log error?
		}
	}

	if handler, ok := e.handlers[handlerID]; ok {
		// Listener: Start
		if err := e.executeTaskListeners(ctx, instance, step, "start"); err != nil {
			return err
		}

		// Engine: Activate Job
		job.State = "ACTIVATED"
		job.Worker = "internal-worker"
		job.UpdatedAt = time.Now()
		e.repo.UpdateJob(ctx, job)

		// Update Handler to accept Context if needed?
		// For now we keep ServiceTaskHandler as is or we update it?
		// User requirement: "Refactor Command Service Repository/Engine for Context Propagation"
		// It's better to update handler too, but let's check definition in engine.go first.
		// Current engine.go: type ServiceTaskHandler func(instance *model.WorkflowInstance, step *model.StepDefinition) error
		// I'll update engine.go later. For now, call as is.
		if err := handler(ctx, instance, step); err != nil {
			// If there are attached boundary events, return the error to let autoAdvance/handleBoundaryError handle it.
			if len(step.BoundaryEventRefs) > 0 {
				return err
			}

			// Engine: Fail Job
			job.State = "FAILED"
			job.Retries--
			job.UpdatedAt = time.Now()
			e.repo.UpdateJob(ctx, job)

			if job.Retries <= 0 {
				// Engine: Raise Incident
				incidentKey := generateKey(fmt.Sprintf("%d_incident", job.Key))
				incident := &model.Incident{
					Key:                incidentKey,
					ID:                 id.GenerateUUIDv7(),
					ProcessInstanceKey: job.ProcessInstanceKey,
					ElementInstanceKey: job.ElementInstanceKey,
					JobKey:             job.Key,
					ErrorType:          "JOB_NO_RETRIES",
					ErrorMessage:       err.Error(),
					State:              "CREATED",
					CreatedAt:          time.Now(),
				}
				if err := e.repo.CreateIncident(ctx, incident); err != nil {
					// log error?
				}
			}

			// Return nil so the engine saves the state at this step (waiting for retry/resolution)
			// instead of rolling back the navigation.
			return nil
		}

		// Engine: Persist Variables (if handler modified context)
		piKey, _ := strconv.ParseInt(instance.ID, 10, 64)
		if err := e.persistVariables(ctx, instance.ID, piKey, instance.Context); err != nil {
			return err
		}

		// Listener: End
		if err := e.executeTaskListeners(ctx, instance, step, "end"); err != nil {
			return err
		}

		// Engine: Complete Job
		job.State = "COMPLETED"
		job.UpdatedAt = time.Now()
		e.repo.UpdateJob(ctx, job)

		return e.proceedToken(ctx, instance, execID, wf)
	} else {
		// No in-process handler found: keep job pending for external worker activation.
		job.State = "CREATED"
		job.Worker = ""
		job.UpdatedAt = time.Now()
		e.repo.UpdateJob(ctx, job)
		return nil
	}
}

type UserTaskExecutor struct{}

func (x *UserTaskExecutor) Execute(ctx context.Context, e *Engine, instance *model.WorkflowInstance, step *model.StepDefinition, execID string, wf *model.WorkflowDefinition) error {
	// Find execution to get ElementInstanceKey
	var exec *model.Execution
	for i := range instance.Executions {
		if instance.Executions[i].ID == execID {
			exec = &instance.Executions[i]
			break
		}
	}
	if exec == nil {
		return fmt.Errorf("execution %s not found in UserTaskExecutor", execID)
	}

	// Create a Job for the User Task
	jobKey := generateKey(fmt.Sprintf("%s_%d_job", execID, exec.ElementInstanceKey))
	var job *model.Job

	// Check if job exists
	if existingJob, err := e.repo.GetJob(ctx, jobKey); err == nil {
		job = existingJob
	} else {
		piKey, _ := strconv.ParseInt(instance.ID, 10, 64)

		// Extract assignment properties
		assignee, _ := step.Properties["assignee"].(string)
		candidateUsers, _ := step.Properties["candidate_users"].(string)
		candidateGroups, _ := step.Properties["candidate_groups"].(string)

		// SLA: Due Date
		var dueDate time.Time
		if dueDateStr, ok := step.Properties["due_date"].(string); ok && dueDateStr != "" {
			// Try parsing as Duration (PT1H)
			if d, err := parseISO8601Duration(dueDateStr); err == nil {
				dueDate = time.Now().Add(d)
			} else {
				// Try parsing as absolute time (ISO8601)
				if t, err := time.Parse(time.RFC3339, dueDateStr); err == nil {
					dueDate = t
				}
			}
		}

		job = &model.Job{
			Key:                  jobKey,
			ID:                   id.GenerateUUIDv7(),
			Type:                 "flowgo:userTask", // Standard type for user tasks
			ProcessInstanceKey:   piKey,
			ElementInstanceKey:   exec.ElementInstanceKey,
			ProcessDefinitionKey: wf.ID,
			ElementID:            step.ID,
			Worker:               "",
			Retries:              1,
			State:                "CREATED",
			Assignee:             assignee,
			CandidateUsers:       candidateUsers,
			CandidateGroups:      candidateGroups,
			DueDate:              dueDate,
			CreatedAt:            time.Now(),
			UpdatedAt:            time.Now(),
		}
		if err := e.repo.CreateJob(ctx, job); err != nil {
			return fmt.Errorf("failed to create job: %w", err)
		}

		// Publish JobCreated
		if err := e.eventPublisher.Publish(ctx, &pb.JobCreated{
			Key:                  job.Key,
			Id:                   job.ID,
			Type:                 job.Type,
			ProcessInstanceKey:   job.ProcessInstanceKey,
			ElementInstanceKey:   job.ElementInstanceKey,
			ProcessDefinitionKey: job.ProcessDefinitionKey,
			ElementId:            job.ElementID,
			CreatedAt:            timestamppb.New(job.CreatedAt),
		}, "JobCreated"); err != nil {
			fmt.Printf("failed to publish JobCreated: %v\n", err)
		}

		// Listener: Create
		if err := e.executeTaskListeners(ctx, instance, step, "create"); err != nil {
			return err
		}

		// Listener: Assignment
		if job.Assignee != "" {
			if err := e.executeTaskListeners(ctx, instance, step, "assignment"); err != nil {
				return err
			}
		}
	}

	return nil
}

type ScriptTaskExecutor struct{}

func (x *ScriptTaskExecutor) Execute(ctx context.Context, e *Engine, instance *model.WorkflowInstance, step *model.StepDefinition, execID string, wf *model.WorkflowDefinition) error {
	script, _ := step.Properties["script"].(string)
	if script == "" {
		// No script? Just proceed.
		return e.proceedToken(ctx, instance, execID, wf)
	}

	// Simple check for format
	format, _ := step.Properties["script_format"].(string)
	if format != "" && format != "javascript" && format != "js" {
		return fmt.Errorf("unsupported script format: %s", format)
	}

	// Setup Timeout
	timeoutDuration := defaultScriptEvaluationTimeout
	if timeoutStr, ok := step.Properties["timeout"].(string); ok && timeoutStr != "" {
		if d, err := parseISO8601Duration(timeoutStr); err == nil {
			timeoutDuration = d
		}
	}

	vm, val, err := evaluateJavaScriptExpression(script, instance.Context, timeoutDuration, expressionKindScript)
	if err != nil {
		// ADR 0007: Safe Execution
		// Create Incident and FAIL the job/execution?
		// Script Tasks are "automatic", so they fail immediately.
		// We should raise an incident and potentially retry?
		// Scripts usually are deterministic, so retrying might not help unless it's a timeout.
		// For now, we raise an Incident and stop.

		// Determine Error Code
		var evalErr *ExpressionEvaluationError
		errorCode := "SCRIPT_ERROR"
		errorMessage := err.Error()

		if errors.As(err, &evalErr) {
			if evalErr.IsTimeout() {
				errorCode = "SCRIPT_TIMEOUT"
			}
		}

		// Parse ProcessInstanceKey
		piKey, _ := strconv.ParseInt(instance.ID, 10, 64)

		incidentKey := generateKey(fmt.Sprintf("%s_incident_script", execID))
		incident := &model.Incident{
			Key:                incidentKey,
			ID:                 id.GenerateUUIDv7(),
			ProcessInstanceKey: piKey,
		}

		// Fetch execution to get keys
		var exec *model.Execution
		for i := range instance.Executions {
			if instance.Executions[i].ID == execID {
				exec = &instance.Executions[i]
				break
			}
		}
		if exec != nil {
			incident.ElementInstanceKey = exec.ElementInstanceKey
		}

		incident.ErrorType = errorCode
		incident.ErrorMessage = errorMessage
		incident.State = "CREATED"
		incident.CreatedAt = time.Now()

		if err := e.repo.CreateIncident(ctx, incident); err != nil {
			// Log failure to create incident
			logger.Default().Error(ctx, "failed to create incident for script error", map[string]any{"error": err.Error()})
		}

		// Update Execution state to failed/incident?
		// Engine doesn't have "Incident" state for execution usually, it stays Active but blocked.
		// But in this simple engine, maybe we just return error?
		// If we return error, proceedToken might handle it.
		// proceedToken calls:
		// if err := executor.Execute(...); err != nil { ... raise incident ... }
		// Wait, let's check `autoAdvance` in navigation.go.
		// `autoAdvance` calls `Execute`.
		// If `Execute` returns error, `autoAdvance` handles incident creation!
		// So we SHOULD just return the error wrapped properly.
		// Let's check `autoAdvance` implementation again.

		return fmt.Errorf("script execution failed: %w", err)
	}

	// Update variables from VM
	// We need to capture both updated existing variables and NEW variables.
	// Iterating over global object keys is one way.
	keys := vm.GlobalObject().Keys()

	for _, k := range keys {
		// Skip internal/standard globals if possible or just overwrite context
		if k == "console" {
			continue
		}

		val := vm.Get(k)
		if val != nil {
			exported := val.Export()
			// Basic filtering of functions/undefined
			if exported != nil {
				instance.Context[k] = exported
			}
		}
	}

	// Handle Result Variable
	if resultVar, ok := step.Properties["result_variable"].(string); ok && resultVar != "" {
		instance.Context[resultVar] = val.Export()
	}

	// Persist changes
	piKey, _ := strconv.ParseInt(instance.ID, 10, 64)
	if err := e.persistVariables(ctx, instance.ID, piKey, instance.Context); err != nil {
		return err
	}

	return e.proceedToken(ctx, instance, execID, wf)
}

type SendTaskExecutor struct{}

func (x *SendTaskExecutor) Execute(ctx context.Context, e *Engine, instance *model.WorkflowInstance, step *model.StepDefinition, execID string, wf *model.WorkflowDefinition) error {
	return e.proceedToken(ctx, instance, execID, wf)
}

type ReceiveTaskExecutor struct{}

func (x *ReceiveTaskExecutor) Execute(ctx context.Context, e *Engine, instance *model.WorkflowInstance, step *model.StepDefinition, execID string, wf *model.WorkflowDefinition) error {
	return nil
}

type BusinessRuleTaskExecutor struct{}

func (x *BusinessRuleTaskExecutor) Execute(ctx context.Context, e *Engine, instance *model.WorkflowInstance, step *model.StepDefinition, execID string, wf *model.WorkflowDefinition) error {
	return e.proceedToken(ctx, instance, execID, wf)
}

type SubProcessExecutor struct{}

func (x *SubProcessExecutor) Execute(ctx context.Context, e *Engine, instance *model.WorkflowInstance, step *model.StepDefinition, execID string, wf *model.WorkflowDefinition) error {
	// 1. Find Start Event inside SubProcess
	var startNode *model.StepDefinition
	for _, subStep := range step.SubSteps {
		if subStep.Type == model.StepTypeStart {
			startNode = &subStep
			break
		}
	}

	if startNode == nil {
		// Empty subprocess or no start event? Just complete it.
		return e.proceedToken(ctx, instance, execID, wf)
	}

	// 2. Create Child Execution
	childExec := model.Execution{
		ID:        generateRuntimeID(),
		StepID:    startNode.ID,
		Status:    "ACTIVE",
		ParentID:  execID, // Link to SubProcess execution
		StartTime: time.Now(),
	}

	// Engine: Create new Element Instance for the SubProcess Start Event
	newKey := generateKey(childExec.ID)
	childExec.ElementInstanceKey = newKey

	// Parse ProcessInstanceKey
	piKey, _ := strconv.ParseInt(instance.ID, 10, 64)

	el := &model.ElementInstance{
		Key:                  newKey,
		ProcessInstanceKey:   piKey,
		ProcessDefinitionKey: wf.ID,
		ElementID:            startNode.ID,
		BpmnElementType:      string(startNode.Type),
		FlowScopeKey:         e.getFlowScopeKey(instance, childExec.ParentID), // Should link to SubProcess ElementInstance
		State:                "ACTIVATED",
		CreatedAt:            time.Now(),
	}
	if err := e.repo.CreateElementInstance(ctx, el); err != nil {
		// log error?
	}

	instance.Executions = append(instance.Executions, childExec)

	// 3. Start Child Execution
	return e.autoAdvance(ctx, instance, childExec.ID, wf)
}

type CallActivityExecutor struct{}

func (x *CallActivityExecutor) Execute(ctx context.Context, e *Engine, instance *model.WorkflowInstance, step *model.StepDefinition, execID string, wf *model.WorkflowDefinition) error {
	// Find execution to get ElementInstanceKey
	var exec *model.Execution
	for i := range instance.Executions {
		if instance.Executions[i].ID == execID {
			exec = &instance.Executions[i]
			break
		}
	}
	if exec == nil {
		return fmt.Errorf("execution %s not found in CallActivityExecutor", execID)
	}

	calledElement, ok := step.Properties["called_element"].(string)
	if !ok || calledElement == "" {
		return fmt.Errorf("call activity %s missing called_element", step.ID)
	}

	// Find latest workflow definition by Process ID
	wfDef, err := e.getWorkflowDefinitionByBpmnProcessID(ctx, calledElement)
	if err != nil {
		return fmt.Errorf("workflow definition not found for key %s: %v", calledElement, err)
	}

	// Start child instance
	// Pass current context (variables) to child
	// Pass ElementInstanceKey as parentExecutionID
	childInstance, err := e.createAndStartInstance(ctx, strconv.FormatInt(wfDef.ID, 10), instance.Context, instance.ID, fmt.Sprintf("%d", exec.ElementInstanceKey))
	if err != nil {
		return err
	}

	// If child completed synchronously, the parent instance in DB has been updated by notifyParentProcess.
	// We must reload it to reflect the changes (e.g. token moved from CallActivity to next step).
	if childInstance.Status == model.StatusCompleted {
		updatedInstance, err := e.GetInstance(ctx, instance.ID)
		if err != nil {
			return fmt.Errorf("failed to reload parent instance after child completion: %v", err)
		}
		// Update the in-memory instance with the latest state from DB
		*instance = *updatedInstance
	}

	return nil
}

type IntermediateThrowEventExecutor struct{}

func (x *IntermediateThrowEventExecutor) Execute(ctx context.Context, e *Engine, instance *model.WorkflowInstance, step *model.StepDefinition, execID string, wf *model.WorkflowDefinition) error {
	var err error
	if signalRef, ok := step.Properties["signal_ref"].(string); ok && signalRef != "" {
		err = e.publishSignal(ctx, signalRef, nil)
	} else if messageRef, ok := step.Properties["message_ref"].(string); ok && messageRef != "" {
		correlationKey := ""
		if ckProp, ok := step.Properties["correlation_key"].(string); ok {
			if val, exists := instance.Context[ckProp]; exists {
				correlationKey = fmt.Sprintf("%v", val)
			} else {
				correlationKey = ckProp
			}
		}
		err = e.publishMessage(ctx, messageRef, correlationKey, nil)
	} else if errorRef, ok := step.Properties["error_ref"].(string); ok && errorRef != "" {
		// Error Throw Event
		errorCode, _ := step.Properties["error_code"].(string)
		errorMessage, _ := step.Properties["error_message"].(string)
		return &BpmnError{
			ErrorCode:    errorCode,
			ErrorMessage: errorMessage,
		}
	} else if evtType, ok := step.Properties["event_definition_type"].(string); ok && evtType == "compensate" {
		// Compensation Throw Event
		activityRef, _ := step.Properties["activity_ref"].(string)
		// Find execution to get ParentID (Scope)
		var exec *model.Execution
		for i := range instance.Executions {
			if instance.Executions[i].ID == execID {
				exec = &instance.Executions[i]
				break
			}
		}
		if exec != nil {
			if err := e.TriggerCompensation(ctx, instance, exec.ParentID, activityRef, wf); err != nil {
				return err
			}
		}
	}

	if err != nil {
		return err
	}
	return e.proceedToken(ctx, instance, execID, wf)
}

type GatewayExecutor struct{}

func (x *GatewayExecutor) Execute(ctx context.Context, e *Engine, instance *model.WorkflowInstance, step *model.StepDefinition, execID string, wf *model.WorkflowDefinition) error {
	// Handle Gateway Join Logic (Converging)
	if len(step.Incoming) > 1 {
		ready, othersToConsume, err := e.checkGatewayJoin(instance, step, wf)
		if err != nil {
			return err
		}

		if !ready {
			return nil // Wait for other tokens
		}
		// Consume other tokens (Merge)
		for _, idx := range othersToConsume {
			if instance.Executions[idx].ID != execID {
				instance.Executions[idx].Status = "COMPLETED"
				// Engine: Complete element instance for consumed token
				if key := instance.Executions[idx].ElementInstanceKey; key != 0 {
					el := &model.ElementInstance{
						Key:     key,
						State:   "COMPLETED",
						EndTime: time.Now(),
					}
					if err := e.repo.UpdateElementInstance(ctx, el); err != nil {
						// log error?
					}
				}
			}
		}
	}

	return e.proceedToken(ctx, instance, execID, wf)
}

type PassthroughExecutor struct{}

func (x *PassthroughExecutor) Execute(ctx context.Context, e *Engine, instance *model.WorkflowInstance, step *model.StepDefinition, execID string, wf *model.WorkflowDefinition) error {
	if step.Type == model.StepTypeEnd {
		if errorRef, ok := step.Properties["error_ref"].(string); ok && errorRef != "" {
			errorCode, _ := step.Properties["error_code"].(string)
			errorMessage, _ := step.Properties["error_message"].(string)
			return &BpmnError{
				ErrorCode:    errorCode,
				ErrorMessage: errorMessage,
			}
		}
		if signalRef, ok := step.Properties["signal_ref"].(string); ok && signalRef != "" {
			// Signal Throw End Event
			if err := e.publishSignal(ctx, signalRef, nil); err != nil {
				return err
			}
		}
		if evtType, ok := step.Properties["event_definition_type"].(string); ok && evtType == "compensate" {
			// Compensation End Event
			activityRef, _ := step.Properties["activity_ref"].(string)
			// Find execution to get ParentID (Scope)
			var exec *model.Execution
			for i := range instance.Executions {
				if instance.Executions[i].ID == execID {
					exec = &instance.Executions[i]
					break
				}
			}
			if exec != nil {
				if err := e.TriggerCompensation(ctx, instance, exec.ParentID, activityRef, wf); err != nil {
					return err
				}
			}
		}
	}
	return e.proceedToken(ctx, instance, execID, wf)
}
