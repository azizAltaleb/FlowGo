package tests

import (
	"context"
	"github.com/azizAltaleb/goflow/backend/libs/model"
	"strconv"
	"testing"
)

func TestSignalEventExecution(t *testing.T) {
	e := setupTestEngine(t)

	// Flow: Start -> Wait for Signal (MySignal) -> End
	steps := []model.StepDefinition{
		{ID: "start", Type: model.StepTypeStart, Name: "Start", Outgoing: []model.Transition{{TargetRef: "catch"}}},
		{
			ID:   "catch",
			Type: model.StepTypeIntermediateCatchEvent,
			Name: "Catch Signal",
			Properties: map[string]any{
				"signal_ref": "MySignal",
			},
			Incoming: []string{"start"},
			Outgoing: []model.Transition{{TargetRef: "end"}},
		},
		{ID: "end", Type: model.StepTypeEnd, Name: "End", Incoming: []string{"catch"}},
	}

	wf, err := e.DeployWorkflow(context.Background(), "Signal Test", steps)
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}

	// Start Instance
	instance, err := e.StartInstance(context.Background(), strconv.FormatInt(wf.ID, 10), nil)
	if err != nil {
		t.Fatalf("StartInstance failed: %v", err)
	}

	// 1. Verify waiting at Catch
	current := getCurrentSteps(instance)
	if len(current) != 1 || !contains(current, "catch") {
		t.Errorf("Expected wait at 'catch', got %v", current)
	}

	// 2. Publish Wrong Signal
	if err := e.PublishSignal(context.Background(), "WrongSignal", nil); err != nil {
		t.Fatalf("PublishSignal failed: %v", err)
	}

	// Reload - should still be waiting
	instance, _ = e.GetInstance(context.Background(), instance.ID)
	current = getCurrentSteps(instance)
	if len(current) != 1 || !contains(current, "catch") {
		t.Errorf("Should still wait at 'catch' after wrong signal, got %v", current)
	}

	// 3. Publish Correct Signal with Payload
	payload := map[string]any{"myVar": "triggered"}
	if err := e.PublishSignal(context.Background(), "MySignal", payload); err != nil {
		t.Fatalf("PublishSignal failed: %v", err)
	}

	// Reload
	instance, _ = e.GetInstance(context.Background(), instance.ID)
	if instance.Status != model.StatusCompleted {
		t.Errorf("Expected COMPLETED, got %s. Executions: %+v", instance.Status, instance.Executions)
	}

	// Verify payload merge
	if val, ok := instance.Context["myVar"]; !ok || val != "triggered" {
		t.Errorf("Expected payload 'myVar'='triggered', got %v", val)
	}
}
