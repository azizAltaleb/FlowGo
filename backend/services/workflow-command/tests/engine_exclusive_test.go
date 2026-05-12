package tests

import (
	"context"
	"github.com/azizAltaleb/flowgo/backend/libs/model"
	"strconv"
	"testing"
)

func TestExclusiveGatewayExecution(t *testing.T) {
	e := setupTestEngine(t)

	// Flow: Start -> XOR Split -> Task A / Task B -> End
	steps := []model.StepDefinition{
		{ID: "start", Type: model.StepTypeStart, Name: "Start", Outgoing: []model.Transition{{TargetRef: "xor"}}},
		{
			ID:       "xor",
			Type:     model.StepTypeGatewayExclusive,
			Name:     "XOR Split",
			Incoming: []string{"start"},
			Outgoing: []model.Transition{
				{TargetRef: "taskA", Condition: "amount < 100"},
				{TargetRef: "taskB", Condition: "amount >= 100"},
			},
		},
		{ID: "taskA", Type: model.StepTypeUserTask, Name: "Task A (<100)", Incoming: []string{"xor"}, Outgoing: []model.Transition{{TargetRef: "end"}}},
		{ID: "taskB", Type: model.StepTypeUserTask, Name: "Task B (>=100)", Incoming: []string{"xor"}, Outgoing: []model.Transition{{TargetRef: "end"}}},
		{ID: "end", Type: model.StepTypeEnd, Name: "End", Incoming: []string{"taskA", "taskB"}},
	}

	_, err := e.DeployWorkflow(context.Background(), "Exclusive Test", steps)
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}

	// Redefine steps for boolean logic
	stepsBool := []model.StepDefinition{
		{ID: "start", Type: model.StepTypeStart, Name: "Start", Outgoing: []model.Transition{{TargetRef: "xor"}}},
		{
			ID:       "xor",
			Type:     model.StepTypeGatewayExclusive,
			Name:     "XOR Split",
			Incoming: []string{"start"},
			Outgoing: []model.Transition{
				{TargetRef: "taskA", Condition: "isSmall"},
				{TargetRef: "taskB", Condition: "isLarge"},
			},
		},
		{ID: "taskA", Type: model.StepTypeUserTask, Name: "Task A", Incoming: []string{"xor"}, Outgoing: []model.Transition{{TargetRef: "end"}}},
		{ID: "taskB", Type: model.StepTypeUserTask, Name: "Task B", Incoming: []string{"xor"}, Outgoing: []model.Transition{{TargetRef: "end"}}},
		{ID: "end", Type: model.StepTypeEnd, Name: "End", Incoming: []string{"taskA", "taskB"}},
	}

	wfBool, _ := e.DeployWorkflow(context.Background(), "Exclusive Bool Test", stepsBool)

	// Test Path A
	ctxA := map[string]any{"isSmall": true, "isLarge": false}
	instA, err := e.StartInstance(context.Background(), strconv.FormatInt(wfBool.ID, 10), ctxA)
	if err != nil {
		t.Fatalf("StartInstance A failed: %v", err)
	}

	currA := getCurrentSteps(instA)
	if len(currA) != 1 || !contains(currA, "taskA") {
		t.Errorf("Expected Task A, got %v", currA)
	}

	// Test Path B
	ctxB := map[string]any{"isSmall": false, "isLarge": true}
	instB, err := e.StartInstance(context.Background(), strconv.FormatInt(wfBool.ID, 10), ctxB)
	if err != nil {
		t.Fatalf("StartInstance B failed: %v", err)
	}

	currB := getCurrentSteps(instB)
	if len(currB) != 1 || !contains(currB, "taskB") {
		t.Errorf("Expected Task B, got %v", currB)
	}
}

func TestExclusiveGatewayJSExpression(t *testing.T) {
	e := setupTestEngine(t)

	// Flow: Start -> XOR -> Task A / Task B -> End
	// Condition: JS Expression
	steps := []model.StepDefinition{
		{ID: "start", Type: model.StepTypeStart, Name: "Start", Outgoing: []model.Transition{{TargetRef: "xor"}}},
		{
			ID:       "xor",
			Type:     model.StepTypeGatewayExclusive,
			Name:     "XOR Split",
			Incoming: []string{"start"},
			Outgoing: []model.Transition{
				{TargetRef: "taskA", Condition: "x * 2 == 10"}, // JS Expression
				{TargetRef: "taskB", Condition: "x * 2 != 10"},
			},
		},
		{ID: "taskA", Type: model.StepTypeUserTask, Name: "Task A", Incoming: []string{"xor"}, Outgoing: []model.Transition{{TargetRef: "end"}}},
		{ID: "taskB", Type: model.StepTypeUserTask, Name: "Task B", Incoming: []string{"xor"}, Outgoing: []model.Transition{{TargetRef: "end"}}},
		{ID: "end", Type: model.StepTypeEnd, Name: "End", Incoming: []string{"taskA", "taskB"}},
	}

	wf, err := e.DeployWorkflow(context.Background(), "JS Expr Test", steps)
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}

	// Case 1: x = 5 -> x*2 == 10 -> Task A
	ctx := map[string]any{"x": 5}
	inst, err := e.StartInstance(context.Background(), strconv.FormatInt(wf.ID, 10), ctx)
	if err != nil {
		t.Fatalf("StartInstance failed: %v", err)
	}

	curr := getCurrentSteps(inst)
	if len(curr) != 1 || !contains(curr, "taskA") {
		t.Errorf("Expected Task A for x=5, got %v", curr)
	}
}

func TestExclusiveGatewaySingleEqualsCompatibility(t *testing.T) {
	e := setupTestEngine(t)

	steps := []model.StepDefinition{
		{ID: "start", Type: model.StepTypeStart, Name: "Start", Outgoing: []model.Transition{{TargetRef: "xor"}}},
		{
			ID:       "xor",
			Type:     model.StepTypeGatewayExclusive,
			Name:     "XOR Split",
			Incoming: []string{"start"},
			Outgoing: []model.Transition{
				{TargetRef: "taskTrue", Condition: "x = true"},
				{TargetRef: "taskFalse", Condition: "x = false"},
			},
		},
		{ID: "taskTrue", Type: model.StepTypeUserTask, Name: "Task True", Incoming: []string{"xor"}, Outgoing: []model.Transition{{TargetRef: "end"}}},
		{ID: "taskFalse", Type: model.StepTypeUserTask, Name: "Task False", Incoming: []string{"xor"}, Outgoing: []model.Transition{{TargetRef: "end"}}},
		{ID: "end", Type: model.StepTypeEnd, Name: "End", Incoming: []string{"taskTrue", "taskFalse"}},
	}

	wf, err := e.DeployWorkflow(context.Background(), "Single Equals Compatibility", steps)
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}

	instTrue, err := e.StartInstance(context.Background(), strconv.FormatInt(wf.ID, 10), map[string]any{"x": true})
	if err != nil {
		t.Fatalf("StartInstance true failed: %v", err)
	}
	if curr := getCurrentSteps(instTrue); len(curr) != 1 || !contains(curr, "taskTrue") {
		t.Fatalf("Expected taskTrue for x=true, got %v", curr)
	}

	instFalse, err := e.StartInstance(context.Background(), strconv.FormatInt(wf.ID, 10), map[string]any{"x": false})
	if err != nil {
		t.Fatalf("StartInstance false failed: %v", err)
	}
	if curr := getCurrentSteps(instFalse); len(curr) != 1 || !contains(curr, "taskFalse") {
		t.Fatalf("Expected taskFalse for x=false, got %v", curr)
	}
}
