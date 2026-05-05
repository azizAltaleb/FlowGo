package tests

import (
	"context"
	"strconv"
	"testing"
	"time"
	"workflow-engine/backend/libs/model"
)

func TestTimerEventExecution(t *testing.T) {
	e := setupTestEngine(t)

	// Flow: Start -> Timer (1s) -> End
	steps := []model.StepDefinition{
		{ID: "start", Type: model.StepTypeStart, Name: "Start", Outgoing: []model.Transition{{TargetRef: "timer"}}},
		{
			ID:   "timer",
			Type: model.StepTypeIntermediateTimerCatchEvent,
			Name: "Wait 1s",
			Properties: map[string]any{
				"timer_duration": "PT1S",
			},
			Incoming: []string{"start"},
			Outgoing: []model.Transition{{TargetRef: "end"}},
		},
		{ID: "end", Type: model.StepTypeEnd, Name: "End", Incoming: []string{"timer"}},
	}

	wf, err := e.DeployWorkflow(context.Background(), "Timer Test", steps)
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}

	// Start Instance
	instance, err := e.StartInstance(context.Background(), strconv.FormatInt(wf.ID, 10), nil)
	if err != nil {
		t.Fatalf("StartInstance failed: %v", err)
	}

	// 1. Verify waiting at Timer
	current := getCurrentSteps(instance)
	if len(current) != 1 || !contains(current, "timer") {
		t.Errorf("Expected wait at 'timer', got %v", current)
	}

	// 2. Check timers immediately (should not trigger)
	if err := e.CheckTimers(context.Background()); err != nil {
		t.Fatalf("CheckTimers failed: %v", err)
	}

	// Reload
	instance, _ = e.GetInstance(context.Background(), instance.ID)
	if instance.Status == model.StatusCompleted {
		t.Fatal("Instance completed too early")
	}

	// 3. Wait for duration + buffer
	time.Sleep(1200 * time.Millisecond)

	// 4. Check timers again (should trigger)
	if err := e.CheckTimers(context.Background()); err != nil {
		t.Fatalf("CheckTimers failed: %v", err)
	}

	// Reload
	instance, _ = e.GetInstance(context.Background(), instance.ID)
	// Status should be COMPLETED because Timer -> End (Automatic)
	if instance.Status != model.StatusCompleted {
		t.Errorf("Expected COMPLETED, got %s. Executions: %+v", instance.Status, instance.Executions)
	}
}
