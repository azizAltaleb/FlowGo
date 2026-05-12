package tests

import (
	"context"
	"github.com/azizAltaleb/flowgo/backend/libs/model"
	"strconv"
	"testing"
)

func TestServiceTaskExecution(t *testing.T) {
	e := setupTestEngine(t)

	// Define a handler
	executed := false
	e.RegisterHandler("myService", func(ctx context.Context, instance *model.WorkflowInstance, step *model.StepDefinition) error {
		executed = true
		// Modify context
		instance.Context["serviceResult"] = "done"
		return nil
	})

	// Flow: Start -> Service Task -> End
	steps := []model.StepDefinition{
		{ID: "start", Type: model.StepTypeStart, Name: "Start", Outgoing: []model.Transition{{TargetRef: "service1"}}},
		{
			ID:             "service1",
			Type:           model.StepTypeServiceTask,
			Name:           "My Service",
			Implementation: "myService",
			Incoming:       []string{"start"},
			Outgoing:       []model.Transition{{TargetRef: "end"}},
		},
		{ID: "end", Type: model.StepTypeEnd, Name: "End", Incoming: []string{"service1"}},
	}

	wf, err := e.DeployWorkflow(context.Background(), "Service Test", steps)
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}

	// Start Instance
	instance, err := e.StartInstance(context.Background(), strconv.FormatInt(wf.ID, 10), make(map[string]any))
	if err != nil {
		t.Fatalf("StartInstance failed: %v", err)
	}

	// Service task should run automatically.
	if !executed {
		t.Error("Service Task handler was not executed")
	}

	// Verify Context update
	// Reload instance to get latest state from DB (though instance pointer might be updated in memory, best to check DB)
	instance, _ = e.GetInstance(context.Background(), instance.ID)
	if val, ok := instance.Context["serviceResult"]; !ok || val != "done" {
		t.Errorf("Expected context['serviceResult'] = 'done', got %v", val)
	}

	if instance.Status != model.StatusCompleted {
		t.Errorf("Expected instance COMPLETED, got %s", instance.Status)
	}
}
