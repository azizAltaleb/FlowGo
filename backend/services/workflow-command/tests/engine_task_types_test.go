package tests

import (
	"context"
	"github.com/artificialflow/artificialflow/backend/libs/model"
	"strconv"
	"testing"
)

func TestAutoExecutingTasks(t *testing.T) {
	e := setupTestEngine(t)

	if _, err := e.DeployDecision(context.Background(), "auto_decision", "auto_decision", []byte(`{
		"id": "auto_decision",
		"hitPolicy": "FIRST",
		"rules": [{"when": {}, "then": {"ok": true}}]
	}`)); err != nil {
		t.Fatalf("DeployDecision failed: %v", err)
	}

	// Flow: Start -> Script -> Send (external job) -> BusinessRule -> End
	steps := []model.StepDefinition{
		{ID: "start", Type: model.StepTypeStart, Name: "Start", Outgoing: []model.Transition{{TargetRef: "script"}}},
		{ID: "script", Type: model.StepTypeScriptTask, Name: "Script Task", Outgoing: []model.Transition{{TargetRef: "send"}}},
		{ID: "send", Type: model.StepTypeSendTask, Name: "Send Task", Implementation: "auto-send", Outgoing: []model.Transition{{TargetRef: "rule"}}},
		{ID: "rule", Type: model.StepTypeBusinessRuleTask, Name: "Rule Task", Properties: map[string]any{"decision_ref": "auto_decision"}, Outgoing: []model.Transition{{TargetRef: "end"}}},
		{ID: "end", Type: model.StepTypeEnd, Name: "End"},
	}

	wf, err := e.DeployWorkflow(context.Background(), "Auto Task Test", steps)
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}

	instance, err := e.StartInstance(context.Background(), strconv.FormatInt(wf.ID, 10), nil)
	if err != nil {
		t.Fatalf("StartInstance failed: %v", err)
	}

	jobs, err := e.ActivateJobs(context.Background(), "auto-send", "worker-1", 1, 0, 0)
	if err != nil || len(jobs) != 1 {
		t.Fatalf("expected send job, got %d err=%v", len(jobs), err)
	}
	if err := e.CompleteJob(context.Background(), jobs[0].Key, "worker-1", nil); err != nil {
		t.Fatalf("CompleteJob failed: %v", err)
	}

	instance, _ = e.GetInstance(context.Background(), instance.ID)
	if instance.Status != model.StatusCompleted {
		t.Errorf("Expected instance to be COMPLETED, got %s", instance.Status)
	}
}
