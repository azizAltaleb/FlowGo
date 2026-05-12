package tests

import (
	"context"
	"github.com/azizAltaleb/flowgo/backend/libs/model"
	"strconv"
	"testing"
)

func TestParallelGatewayExecution(t *testing.T) {
	e := setupTestEngine(t)

	// Flow: Start -> Parallel Split -> Task A & Task B -> Parallel Join -> End
	steps := []model.StepDefinition{
		{ID: "start", Type: model.StepTypeStart, Name: "Start", Outgoing: []model.Transition{{TargetRef: "split"}}},

		// Split
		{ID: "split", Type: model.StepTypeGatewayParallel, Name: "Parallel Split",
			Incoming: []string{"start"},
			Outgoing: []model.Transition{
				{TargetRef: "taskA"},
				{TargetRef: "taskB"},
			},
		},

		// Branch A
		{ID: "taskA", Type: model.StepTypeUserTask, Name: "Task A", Incoming: []string{"split"}, Outgoing: []model.Transition{{TargetRef: "join"}}},

		// Branch B
		{ID: "taskB", Type: model.StepTypeUserTask, Name: "Task B", Incoming: []string{"split"}, Outgoing: []model.Transition{{TargetRef: "join"}}},

		// Join
		{ID: "join", Type: model.StepTypeGatewayParallel, Name: "Parallel Join",
			Incoming: []string{"taskA", "taskB"},
			Outgoing: []model.Transition{{TargetRef: "end"}},
		},

		{ID: "end", Type: model.StepTypeEnd, Name: "End", Incoming: []string{"join"}},
	}

	wf, err := e.DeployWorkflow(context.Background(), "Parallel Test", steps)
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}

	// Start
	instance, err := e.StartInstance(context.Background(), strconv.FormatInt(wf.ID, 10), nil)
	if err != nil {
		t.Fatalf("StartInstance failed: %v", err)
	}

	// Should be at Task A and Task B now
	current := getCurrentSteps(instance)
	if len(current) != 2 || !contains(current, "taskA") || !contains(current, "taskB") {
		t.Errorf("Expected tokens at taskA and taskB, got: %v", current)
	}

	// Complete Task A
	// We need to complete the specific execution for Task A.
	// The current CompleteTask implementation just picks the first active one?
	// Let's rely on finding the execution ID.
	var execA, execB string
	for _, ex := range instance.Executions {
		if ex.StepID == "taskA" && ex.Status == "ACTIVE" {
			execA = ex.ID
		}
		if ex.StepID == "taskB" && ex.Status == "ACTIVE" {
			execB = ex.ID
		}
	}

	// Use new CompleteExecution method
	if err := e.CompleteExecution(context.Background(), instance.ID, execA); err != nil {
		t.Fatalf("Failed to complete Task A: %v", err)
	}

	// Reload instance
	instance, _ = e.GetInstance(context.Background(), instance.ID)

	// Check status: Task A should be completed. Token should be waiting at "join".
	// But Parallel Join waits for ALL incoming. So checking status at join.
	// Actually, my implementation of `checkGatewayJoin` returns false if not ready.
	// So the token for Task A -> Join will be consumed?
	// Wait, in my code:
	// `if !ready { return nil }`
	// It returns nil, meaning the token REMAINS ACTIVE at the Gateway step?
	// Let's check `proceedToken` logic.
	// It calls `autoAdvance`.
	// `autoAdvance` calls `checkGatewayJoin`.
	// If `!ready`, it returns `nil`.
	// So the execution is now at `join` step and status is `ACTIVE` (because proceedToken updates StepID then calls autoAdvance).

	// Let's verify.
	current = getCurrentSteps(instance)
	if !contains(current, "join") || !contains(current, "taskB") {
		t.Errorf("Expected tokens at join and taskB, got: %v", current)
	}

	// Complete Task B
	if err := e.CompleteExecution(context.Background(), instance.ID, execB); err != nil {
		t.Fatalf("Failed to complete Task B: %v", err)
	}

	// Now Join should trigger.
	// Reload
	instance, _ = e.GetInstance(context.Background(), instance.ID)

	if instance.Status != model.StatusCompleted {
		t.Errorf("Expected instance to be COMPLETED, got %s", instance.Status)
	}
}

func TestInclusiveGatewayExecution(t *testing.T) {
	e := setupTestEngine(t)

	// Flow: Start -> Inclusive Split (cond1, cond2) -> Task A / Task B -> Inclusive Join -> End
	steps := []model.StepDefinition{
		{ID: "start", Type: model.StepTypeStart, Name: "Start", Outgoing: []model.Transition{{TargetRef: "split"}}},

		// Split
		{ID: "split", Type: model.StepTypeGatewayInclusive, Name: "Inclusive Split",
			Incoming: []string{"start"},
			Outgoing: []model.Transition{
				{TargetRef: "taskA", Condition: "varA"}, // executes if varA is true
				{TargetRef: "taskB", Condition: "varB"}, // executes if varB is true
			},
		},

		{ID: "taskA", Type: model.StepTypeUserTask, Name: "Task A", Incoming: []string{"split"}, Outgoing: []model.Transition{{TargetRef: "join"}}},
		{ID: "taskB", Type: model.StepTypeUserTask, Name: "Task B", Incoming: []string{"split"}, Outgoing: []model.Transition{{TargetRef: "join"}}},

		// Join
		{ID: "join", Type: model.StepTypeGatewayInclusive, Name: "Inclusive Join",
			Incoming: []string{"taskA", "taskB"},
			Outgoing: []model.Transition{{TargetRef: "end"}},
		},

		{ID: "end", Type: model.StepTypeEnd, Name: "End", Incoming: []string{"join"}},
	}

	wf, err := e.DeployWorkflow(context.Background(), "Inclusive Test", steps)
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}

	// Case 1: Only varA is true. Should go Task A -> Join -> End. Task B should not be active.
	ctx := map[string]any{"varA": true, "varB": false}
	instance, err := e.StartInstance(context.Background(), strconv.FormatInt(wf.ID, 10), ctx)
	if err != nil {
		t.Fatalf("StartInstance failed: %v", err)
	}

	current := getCurrentSteps(instance)
	if len(current) != 1 || !contains(current, "taskA") {
		t.Errorf("Expected token only at taskA, got: %v", current)
	}

	// Complete Task A
	// Find exec
	var execA string
	for _, ex := range instance.Executions {
		if ex.StepID == "taskA" {
			execA = ex.ID
			break
		}
	}
	if err := e.CompleteExecution(context.Background(), instance.ID, execA); err != nil {
		t.Fatalf("Failed to complete Task A: %v", err)
	}

	instance, _ = e.GetInstance(context.Background(), instance.ID)
	// Join should happen immediately because Task B is not active and not reachable (path not taken).
	if instance.Status != model.StatusCompleted {
		t.Errorf("Expected instance COMPLETED, got %s. Executions: %v", instance.Status, instance.Executions)
	}
}
