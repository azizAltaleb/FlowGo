package tests

import (
	"context"
	"github.com/azizAltaleb/flowgo/backend/libs/model"
	"strconv"
	"testing"
)

func TestAutoExecutingTasks(t *testing.T) {
	e := setupTestEngine(t)

	// Flow: Start -> Script -> Send -> BusinessRule -> End
	steps := []model.StepDefinition{
		{ID: "start", Type: model.StepTypeStart, Name: "Start", Outgoing: []model.Transition{{TargetRef: "script"}}},
		{ID: "script", Type: model.StepTypeScriptTask, Name: "Script Task", Outgoing: []model.Transition{{TargetRef: "send"}}},
		{ID: "send", Type: model.StepTypeSendTask, Name: "Send Task", Outgoing: []model.Transition{{TargetRef: "rule"}}},
		{ID: "rule", Type: model.StepTypeBusinessRuleTask, Name: "Rule Task", Outgoing: []model.Transition{{TargetRef: "end"}}},
		{ID: "end", Type: model.StepTypeEnd, Name: "End"},
	}

	wf, err := e.DeployWorkflow(context.Background(), "Auto Task Test", steps)
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}

	// Start Instance
	instance, err := e.StartInstance(context.Background(), strconv.FormatInt(wf.ID, 10), nil)
	if err != nil {
		t.Fatalf("StartInstance failed: %v", err)
	}

	// All tasks are automatic, so instance should complete immediately
	// Reload to get latest status
	instance, _ = e.GetInstance(context.Background(), instance.ID)
	if instance.Status != model.StatusCompleted {
		t.Errorf("Expected instance to be COMPLETED, got %s", instance.Status)
	}

	// Verify step progression (optional, if we track history, but here checking completed state is enough for auto-execution)
}
