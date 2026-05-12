package tests

import (
	"context"
	"github.com/azizAltaleb/flowgo/backend/libs/model"
	"strconv"
	"testing"
)

func TestBPMNEventsAndReceiveTask(t *testing.T) {
	e := setupTestEngine(t)

	// Flow: Start -> Throw Event -> Receive Task -> Catch Event -> End
	steps := []model.StepDefinition{
		{ID: "start", Type: model.StepTypeStart, Name: "Start", Outgoing: []model.Transition{{TargetRef: "throw"}}},
		// Throw event should be automatic
		{ID: "throw", Type: model.StepTypeIntermediateThrowEvent, Name: "Throw Signal", Outgoing: []model.Transition{{TargetRef: "receive"}}},
		// Receive task should wait
		{ID: "receive", Type: model.StepTypeReceiveTask, Name: "Wait for Message", Outgoing: []model.Transition{{TargetRef: "catch"}}},
		// Catch event should wait
		{ID: "catch", Type: model.StepTypeIntermediateCatchEvent, Name: "Wait for Signal", Outgoing: []model.Transition{{TargetRef: "end"}}},
		{ID: "end", Type: model.StepTypeEnd, Name: "End"},
	}

	wf, err := e.DeployWorkflow(context.Background(), "BPMN Events Test", steps)
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}

	// Start Instance
	instance, err := e.StartInstance(context.Background(), strconv.FormatInt(wf.ID, 10), nil)
	if err != nil {
		t.Fatalf("StartInstance failed: %v", err)
	}

	// 1. Should auto-advance past 'throw' and wait at 'receive'
	current := getCurrentSteps(instance)
	if len(current) != 1 || !contains(current, "receive") {
		t.Errorf("Expected wait at 'receive', got %v", current)
	}

	// 2. Complete 'receive'
	if err := e.CompleteTask(context.Background(), instance.ID, "receive"); err != nil {
		t.Fatalf("Failed to complete receive task: %v", err)
	}

	// Reload
	instance, _ = e.GetInstance(context.Background(), instance.ID)

	// 3. Should now be waiting at 'catch'
	current = getCurrentSteps(instance)
	if len(current) != 1 || !contains(current, "catch") {
		t.Errorf("Expected wait at 'catch', got %v", current)
	}

	// 4. Complete 'catch' (Trigger event)
	if err := e.CompleteTask(context.Background(), instance.ID, "catch"); err != nil {
		t.Fatalf("Failed to trigger catch event: %v", err)
	}

	// Reload
	instance, _ = e.GetInstance(context.Background(), instance.ID)

	// 5. Should be End/Completed
	if instance.Status != model.StatusCompleted {
		t.Errorf("Expected COMPLETED, got %s", instance.Status)
	}
}
