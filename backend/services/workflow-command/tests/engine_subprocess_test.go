package tests

import (
	"context"
	"github.com/azizAltaleb/goflow/backend/libs/model"
	"strconv"
	"testing"
)

func TestSubProcessExecution(t *testing.T) {
	e := setupTestEngine(t)

	// Define SubSteps
	subSteps := []model.StepDefinition{
		{ID: "subStart", Type: model.StepTypeStart, Name: "Sub Start", Outgoing: []model.Transition{{TargetRef: "subUser"}}},
		{ID: "subUser", Type: model.StepTypeUserTask, Name: "Sub User Task", Incoming: []string{"subStart"}, Outgoing: []model.Transition{{TargetRef: "subEnd"}}},
		{ID: "subEnd", Type: model.StepTypeEnd, Name: "Sub End", Incoming: []string{"subUser"}},
	}

	// Main Process Steps
	steps := []model.StepDefinition{
		{ID: "start", Type: model.StepTypeStart, Name: "Start", Outgoing: []model.Transition{{TargetRef: "sub"}}},
		{
			ID:       "sub",
			Type:     model.StepTypeSubProcess,
			Name:     "SubProcess",
			SubSteps: subSteps,
			Incoming: []string{"start"},
			Outgoing: []model.Transition{{TargetRef: "end"}},
		},
		{ID: "end", Type: model.StepTypeEnd, Name: "End", Incoming: []string{"sub"}},
	}
	// Note: In the parser, we flatten the steps. So we should verify that behavior here by appending subSteps to steps manually
	// because the engine expects all steps to be in the main list for ID lookup.
	// The parser does this automatically. Here we are constructing manually.
	allSteps := append(steps, subSteps...)

	wf, err := e.DeployWorkflow(context.Background(), "SubProcess Test", allSteps)
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}

	// Start Instance
	instance, err := e.StartInstance(context.Background(), strconv.FormatInt(wf.ID, 10), nil)
	if err != nil {
		t.Fatalf("StartInstance failed: %v", err)
	}

	// 1. Verify Root is at SubProcess (Active) and Child is at Sub User Task
	// Get instance to refresh executions
	instance, _ = e.GetInstance(context.Background(), instance.ID)

	var subProcExec, childExec *model.Execution
	for i := range instance.Executions {
		ex := &instance.Executions[i]
		if ex.StepID == "sub" && ex.Status == "ACTIVE" {
			subProcExec = ex
		}
		if ex.StepID == "subUser" && ex.Status == "ACTIVE" {
			childExec = ex
		}
	}

	if subProcExec == nil {
		t.Errorf("Expected active execution at 'sub'. Executions: %+v", instance.Executions)
	}
	if childExec == nil {
		t.Errorf("Expected active execution at 'subUser'. Executions: %+v", instance.Executions)
	}
	if childExec != nil && subProcExec != nil && childExec.ParentID != subProcExec.ID {
		t.Errorf("Expected child parentID=%s, got %s", subProcExec.ID, childExec.ParentID)
	}

	if childExec == nil {
		return
	}

	// 2. Complete Sub User Task
	if err := e.CompleteTask(context.Background(), instance.ID, "subUser"); err != nil {
		t.Fatalf("CompleteTask subUser failed: %v", err)
	}

	// 3. Verify Completion
	instance, _ = e.GetInstance(context.Background(), instance.ID)
	if instance.Status != model.StatusCompleted {
		t.Errorf("Expected instance completed, got %s. Executions: %+v", instance.Status, instance.Executions)
	}

	// Verify path: Start -> Sub(Active) -> SubStart -> SubUser -> SubEnd -> Sub(Complete) -> End
	// Check if subEnd was reached
	foundSubEnd := false
	for _, ex := range instance.Executions {
		if ex.StepID == "subEnd" && ex.Status == "COMPLETED" {
			foundSubEnd = true
			break
		}
	}
	if !foundSubEnd {
		t.Errorf("Expected execution at 'subEnd'")
	}
}
