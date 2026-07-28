package application

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/artificialflow/artificialflow/backend/libs/model"
)

// PublishEscalation delivers an escalation to waiting catches/boundaries and event sub-processes.
func (e *Engine) PublishEscalation(ctx context.Context, escalationCode string, payload map[string]any) error {
	escalationCode = strings.TrimSpace(escalationCode)
	if escalationCode == "" {
		return fmt.Errorf("escalation code is required")
	}
	return e.withTx(ctx, func(tx *Engine) error {
		if err := tx.deliverEscalationToWaiters(ctx, escalationCode, payload); err != nil {
			return err
		}
		return tx.startInstancesForEscalationStart(ctx, escalationCode, payload)
	})
}

func (e *Engine) publishEscalation(ctx context.Context, escalationCode string, payload map[string]any) error {
	if err := e.deliverEscalationToWaiters(ctx, escalationCode, payload); err != nil {
		return err
	}
	return e.startInstancesForEscalationStart(ctx, escalationCode, payload)
}

func escalationCodeMatches(step *model.StepDefinition, escalationCode string) bool {
	if step == nil || step.Properties == nil {
		return false
	}
	if step.Properties["event_definition_type"] != "escalation" {
		// Boundary may only have escalation_ref set
		_, hasRef := step.Properties["escalation_ref"]
		_, hasCode := step.Properties["escalation_code"]
		if !hasRef && !hasCode {
			return false
		}
	}
	code, _ := step.Properties["escalation_code"].(string)
	if code == "" {
		code, _ = step.Properties["escalation_ref"].(string)
	}
	return code == "" || code == escalationCode
}

func (e *Engine) deliverEscalationToWaiters(ctx context.Context, escalationCode string, payload map[string]any) error {
	active, err := e.repo.ListActiveElementInstances(ctx, 0)
	if err != nil {
		return err
	}
	defs := map[int64]*model.WorkflowDefinition{}
	for _, el := range active {
		wf, err := e.getWorkflowDefinitionFromMemo(ctx, el.ProcessDefinitionKey, defs)
		if err != nil || wf == nil {
			continue
		}
		step := findStepDeep(wf.Steps, el.ElementID)
		if step == nil {
			continue
		}
		instance, err := e.GetInstance(ctx, fmt.Sprintf("%d", el.ProcessInstanceKey))
		if err != nil {
			continue
		}
		if payload != nil {
			if instance.Context == nil {
				instance.Context = map[string]any{}
			}
			for k, v := range payload {
				instance.Context[k] = v
			}
			_ = e.persistVariables(ctx, instance.ID, el.ProcessInstanceKey, payload)
		}

		// Intermediate escalation catch waiting on this element.
		if step.Type == model.StepTypeIntermediateCatchEvent && escalationCodeMatches(step, escalationCode) {
			for _, ex := range instance.Executions {
				if ex.ElementInstanceKey == el.Key && (ex.Status == "ACTIVE" || ex.Status == "") {
					if err := e.proceedToken(ctx, instance, ex.ID, wf); err != nil {
						return err
					}
					break
				}
			}
			_ = e.tryStartEventSubProcesses(ctx, instance, wf, "escalation", escalationCode, payload)
			continue
		}

		// Activity with attached escalation boundary.
		for _, ref := range step.BoundaryEventRefs {
			boundary := findStepDeep(wf.Steps, ref)
			if !escalationCodeMatches(boundary, escalationCode) {
				continue
			}
			if err := e.triggerBoundaryPath(ctx, instance, boundary, step.ID, wf); err != nil {
				return err
			}
			_ = e.tryStartEventSubProcesses(ctx, instance, wf, "escalation", escalationCode, payload)
			break
		}
	}
	return nil
}

// CheckConditionals evaluates waiting conditional catches/boundaries (and event sub-process starts).
func (e *Engine) CheckConditionals(ctx context.Context) error {
	return e.withTx(ctx, func(tx *Engine) error {
		active, err := tx.repo.ListActiveElementInstancesByTypes(ctx, 0, []string{
			string(model.StepTypeIntermediateCatchEvent),
			string(model.StepTypeBoundaryEvent),
		})
		if err != nil {
			return err
		}
		defs := map[int64]*model.WorkflowDefinition{}
		for _, el := range active {
			wf, err := tx.getWorkflowDefinitionFromMemo(ctx, el.ProcessDefinitionKey, defs)
			if err != nil || wf == nil {
				continue
			}
			step := findStepDeep(wf.Steps, el.ElementID)
			if step == nil || step.Properties == nil {
				continue
			}
			if step.Properties["event_definition_type"] != "conditional" {
				continue
			}
			cond, _ := step.Properties["condition"].(string)
			if strings.TrimSpace(cond) == "" {
				continue
			}
			instance, err := tx.GetInstance(ctx, fmt.Sprintf("%d", el.ProcessInstanceKey))
			if err != nil {
				continue
			}
			if !tx.evaluateCondition(ctx, cond, instance.Context) {
				continue
			}
			execID := ""
			for _, ex := range instance.Executions {
				if ex.ElementInstanceKey == el.Key && (ex.Status == "ACTIVE" || ex.Status == "") {
					execID = ex.ID
					break
				}
			}
			if execID == "" {
				continue
			}
			if step.Type == model.StepTypeBoundaryEvent {
				attachedID, _ := step.Properties["attached_to"].(string)
				if err := tx.triggerBoundaryPath(ctx, instance, step, attachedID, wf); err != nil {
					return err
				}
			} else {
				if err := tx.proceedToken(ctx, instance, execID, wf); err != nil {
					return err
				}
			}
		}
		// Advance waiting conditional start events (token parked on START).
		activeStarts, err := tx.repo.ListActiveElementInstancesByTypes(ctx, 0, []string{string(model.StepTypeStart)})
		if err != nil {
			return err
		}
		for _, el := range activeStarts {
			wf, err := tx.getWorkflowDefinitionFromMemo(ctx, el.ProcessDefinitionKey, defs)
			if err != nil || wf == nil {
				continue
			}
			step := findStepDeep(wf.Steps, el.ElementID)
			if step == nil || step.Properties["event_definition_type"] != "conditional" {
				continue
			}
			cond, _ := step.Properties["condition"].(string)
			instance, err := tx.GetInstance(ctx, fmt.Sprintf("%d", el.ProcessInstanceKey))
			if err != nil {
				continue
			}
			if !tx.evaluateCondition(ctx, cond, instance.Context) {
				continue
			}
			for _, ex := range instance.Executions {
				if ex.ElementInstanceKey == el.Key && (ex.Status == "ACTIVE" || ex.Status == "") {
					if err := tx.proceedToken(ctx, instance, ex.ID, wf); err != nil {
						return err
					}
					break
				}
			}
		}
		return nil
	})
}

func (e *Engine) triggerBoundaryPath(ctx context.Context, instance *model.WorkflowInstance, boundary *model.StepDefinition, attachedStepID string, wf *model.WorkflowDefinition) error {
	cancelActivity := true
	if c, ok := boundary.Properties["cancel_activity"].(bool); ok {
		cancelActivity = c
	}
	var parentID string
	for i := range instance.Executions {
		ex := &instance.Executions[i]
		if ex.StepID != attachedStepID {
			continue
		}
		if ex.Status != "ACTIVE" && ex.Status != "" {
			continue
		}
		parentID = ex.ParentID
		if cancelActivity {
			ex.Status = "TERMINATED"
			if key := ex.ElementInstanceKey; key != 0 {
				_ = e.repo.UpdateElementInstance(ctx, &model.ElementInstance{
					Key: key, State: "TERMINATED", EndTime: time.Now(),
				})
			}
		}
		break
	}
	piKey, _ := strconv.ParseInt(instance.ID, 10, 64)
	for _, t := range boundary.Outgoing {
		newExec := model.Execution{
			ID:        generateRuntimeID(),
			StepID:    t.TargetRef,
			Status:    "ACTIVE",
			ParentID:  parentID,
			StartTime: time.Now(),
		}
		newKey := generateKey(newExec.ID)
		newExec.ElementInstanceKey = newKey
		nextType := "TASK"
		if s := findStepDeep(wf.Steps, t.TargetRef); s != nil {
			nextType = string(s.Type)
		}
		_ = e.repo.CreateElementInstance(ctx, &model.ElementInstance{
			Key: newKey, ProcessInstanceKey: piKey, ProcessDefinitionKey: wf.ID,
			ElementID: t.TargetRef, BpmnElementType: nextType,
			FlowScopeKey: e.getFlowScopeKey(instance, parentID),
			State:        "ACTIVATED", CreatedAt: time.Now(),
		})
		instance.Executions = append(instance.Executions, newExec)
		if err := e.autoAdvance(ctx, instance, newExec.ID, wf); err != nil {
			return err
		}
	}
	return nil
}

func findStepDeep(steps []model.StepDefinition, id string) *model.StepDefinition {
	for i := range steps {
		if steps[i].ID == id {
			return &steps[i]
		}
		if found := findStepDeep(steps[i].SubSteps, id); found != nil {
			return found
		}
	}
	return nil
}

func (e *Engine) tryStartEventSubProcesses(ctx context.Context, instance *model.WorkflowInstance, wf *model.WorkflowDefinition, eventKind, eventRef string, payload map[string]any) error {
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
		if !eventSubProcessStartMatches(start, eventKind, eventRef) {
			continue
		}
		interrupting := true
		if c, ok := start.Properties["cancel_activity"].(bool); ok {
			interrupting = c
		}
		if interrupting {
			for k := range instance.Executions {
				st := strings.ToUpper(instance.Executions[k].Status)
				if st == "ACTIVE" || st == "WAITING" || st == "" {
					// Keep tokens inside this event sub-process if already running.
					if instance.Executions[k].ParentID == sp.ID {
						continue
					}
					instance.Executions[k].Status = "TERMINATED"
				}
			}
		}
		if payload != nil {
			if instance.Context == nil {
				instance.Context = map[string]any{}
			}
			for k, v := range payload {
				instance.Context[k] = v
			}
		}
		child := model.Execution{
			ID:        generateRuntimeID(),
			StepID:    start.ID,
			Status:    "ACTIVE",
			ParentID:  sp.ID,
			StartTime: time.Now(),
		}
		newKey := generateKey(child.ID)
		child.ElementInstanceKey = newKey
		piKey, _ := strconv.ParseInt(instance.ID, 10, 64)
		_ = e.repo.CreateElementInstance(ctx, &model.ElementInstance{
			Key: newKey, ProcessInstanceKey: piKey, ProcessDefinitionKey: wf.ID,
			ElementID: start.ID, BpmnElementType: string(start.Type),
			FlowScopeKey: e.getFlowScopeKey(instance, sp.ID),
			State:        "ACTIVATED", CreatedAt: time.Now(),
		})
		instance.Executions = append(instance.Executions, child)
		if err := e.autoAdvance(ctx, instance, child.ID, wf); err != nil {
			return err
		}
	}
	return nil
}

func eventSubProcessStartMatches(start *model.StepDefinition, eventKind, eventRef string) bool {
	switch eventKind {
	case "message":
		ref, _ := start.Properties["message_ref"].(string)
		return start.Properties["event_definition_type"] == "message" && (eventRef == "" || ref == eventRef)
	case "signal":
		ref, _ := start.Properties["signal_ref"].(string)
		return start.Properties["event_definition_type"] == "signal" && (eventRef == "" || ref == eventRef)
	case "escalation":
		ref, _ := start.Properties["escalation_code"].(string)
		if ref == "" {
			ref, _ = start.Properties["escalation_ref"].(string)
		}
		return start.Properties["event_definition_type"] == "escalation" && (eventRef == "" || ref == eventRef || ref == "")
	case "error":
		ref, _ := start.Properties["error_code"].(string)
		return start.Properties["event_definition_type"] == "error" || ref != ""
	case "timer":
		return start.Properties["event_definition_type"] == "timer"
	case "conditional":
		return start.Properties["event_definition_type"] == "conditional"
	default:
		return false
	}
}

// startInstancesForMessageStart creates instances for process definitions whose none-start is a message start.
func (e *Engine) startInstancesForMessageStart(ctx context.Context, messageName, correlationKey string, payload map[string]any) error {
	defs, err := e.listDeployedWorkflowDefinitions(ctx)
	if err != nil {
		return err
	}
	for _, wf := range defs {
		start := firstStartStep(wf.Steps)
		if start == nil || start.Properties == nil {
			continue
		}
		if start.Properties["event_definition_type"] != "message" {
			continue
		}
		ref, _ := start.Properties["message_ref"].(string)
		if ref != messageName {
			continue
		}
		vars := map[string]any{}
		for k, v := range payload {
			vars[k] = v
		}
		if correlationKey != "" {
			if ck, ok := start.Properties["correlation_key"].(string); ok && ck != "" {
				vars[ck] = correlationKey
			} else {
				vars["correlationKey"] = correlationKey
			}
		}
		if _, err := e.StartInstance(ctx, strconv.FormatInt(wf.ID, 10), vars); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) startInstancesForSignalStart(ctx context.Context, signalName string, payload map[string]any) error {
	defs, err := e.listDeployedWorkflowDefinitions(ctx)
	if err != nil {
		return err
	}
	for _, wf := range defs {
		start := firstStartStep(wf.Steps)
		if start == nil || start.Properties == nil {
			continue
		}
		if start.Properties["event_definition_type"] != "signal" {
			continue
		}
		ref, _ := start.Properties["signal_ref"].(string)
		if ref != signalName {
			continue
		}
		if _, err := e.StartInstance(ctx, strconv.FormatInt(wf.ID, 10), payload); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) startInstancesForEscalationStart(ctx context.Context, escalationCode string, payload map[string]any) error {
	defs, err := e.listDeployedWorkflowDefinitions(ctx)
	if err != nil {
		return err
	}
	for _, wf := range defs {
		start := firstStartStep(wf.Steps)
		if start == nil || start.Properties == nil {
			continue
		}
		if start.Properties["event_definition_type"] != "escalation" {
			continue
		}
		ref, _ := start.Properties["escalation_code"].(string)
		if ref == "" {
			ref, _ = start.Properties["escalation_ref"].(string)
		}
		if ref != "" && ref != escalationCode {
			continue
		}
		if _, err := e.StartInstance(ctx, strconv.FormatInt(wf.ID, 10), payload); err != nil {
			return err
		}
	}
	return nil
}

func firstStartStep(steps []model.StepDefinition) *model.StepDefinition {
	for i := range steps {
		if steps[i].Type == model.StepTypeStart {
			return &steps[i]
		}
	}
	return nil
}

func (e *Engine) listDeployedWorkflowDefinitions(ctx context.Context) ([]*model.WorkflowDefinition, error) {
	// Narrow interface so incomplete test mocks without a real process catalog are skipped.
	type startEventCatalog interface {
		ListProcessesForEventStarts(ctx context.Context) ([]*model.Process, error)
	}
	l, ok := e.repo.(startEventCatalog)
	if !ok {
		return nil, nil
	}
	procs, err := l.ListProcessesForEventStarts(ctx)
	if err != nil {
		return nil, nil
	}
	// Keep highest version per BPMN process id.
	latest := map[string]*model.Process{}
	for _, p := range procs {
		if p == nil {
			continue
		}
		prev, ok := latest[p.BpmnProcessID]
		if !ok || p.Version >= prev.Version {
			latest[p.BpmnProcessID] = p
		}
	}
	out := make([]*model.WorkflowDefinition, 0, len(latest))
	for _, p := range latest {
		wf, err := e.getWorkflowDefinition(ctx, p.Key)
		if err != nil {
			continue
		}
		out = append(out, wf)
	}
	return out, nil
}

// cancelTransactionScope cancels active work under a transaction sub-process and fires cancel boundaries.
func (e *Engine) cancelTransactionScope(ctx context.Context, instance *model.WorkflowInstance, transactionExecID string, wf *model.WorkflowDefinition) error {
	var txStepID string
	var parentID string
	for i := range instance.Executions {
		if instance.Executions[i].ID == transactionExecID {
			txStepID = instance.Executions[i].StepID
			parentID = instance.Executions[i].ParentID
			instance.Executions[i].Status = "TERMINATED"
			break
		}
	}
	txStep := findStepDeep(wf.Steps, txStepID)
	if txStep == nil {
		return fmt.Errorf("transaction step %s not found", txStepID)
	}
	for i := range instance.Executions {
		ex := &instance.Executions[i]
		if ex.ParentID == transactionExecID || ex.ParentID == txStepID {
			if ex.Status == "ACTIVE" || ex.Status == "" {
				ex.Status = "CANCELLED"
			}
		}
	}
	// Trigger cancel boundary events attached to the transaction.
	for _, ref := range txStep.BoundaryEventRefs {
		boundary := findStepDeep(wf.Steps, ref)
		if boundary == nil || boundary.Properties["event_definition_type"] != "cancel" {
			continue
		}
		if err := e.triggerBoundaryPath(ctx, instance, boundary, txStepID, wf); err != nil {
			return err
		}
	}
	_ = parentID
	return e.completeInstanceIfNoActiveExecutions(ctx, instance, wf)
}
