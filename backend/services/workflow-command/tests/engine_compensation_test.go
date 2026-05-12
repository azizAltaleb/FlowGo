package tests

import (
	"context"
	"github.com/azizAltaleb/flowgo/backend/libs/model"
	"strconv"
	"testing"
)

func TestCompensationExecution(t *testing.T) {
	e := setupTestEngine(t)

	// Handlers
	e.RegisterHandler("serviceA", func(ctx context.Context, instance *model.WorkflowInstance, step *model.StepDefinition) error {
		instance.Context["serviceA_done"] = true
		return nil
	})
	e.RegisterHandler("undoServiceA", func(ctx context.Context, instance *model.WorkflowInstance, step *model.StepDefinition) error {
		instance.Context["serviceA_undone"] = true
		return nil
	})

	// Workflow:
	// Start -> Service A (Boundary Compensation -> Undo Service A) -> Intermediate Compensation Throw -> End
	steps := []model.StepDefinition{
		{ID: "start", Type: model.StepTypeStart, Outgoing: []model.Transition{{TargetRef: "serviceA"}}},
		{
			ID:                "serviceA",
			Type:              model.StepTypeServiceTask,
			Name:              "Service A",
			Implementation:    "serviceA",
			BoundaryEventRefs: []string{"compBoundary"},
			Outgoing:          []model.Transition{{TargetRef: "compThrow"}},
		},
		{
			ID:   "compBoundary",
			Type: model.StepTypeBoundaryEvent,
			Properties: map[string]any{
				"event_definition_type": "compensate",
				"attached_to":           "serviceA",
			},
			Outgoing: []model.Transition{{TargetRef: "undoServiceA"}},
		},
		{
			ID:             "undoServiceA",
			Type:           model.StepTypeServiceTask,
			Name:           "Undo Service A",
			Implementation: "undoServiceA",
			// Compensation handlers usually don't have outgoing flows in BPMN, they just complete.
			// But our engine requires outgoing or it completes. If outgoing is empty, it completes.
			Outgoing: []model.Transition{},
		},
		{
			ID:   "compThrow",
			Type: model.StepTypeIntermediateThrowEvent,
			Name: "Throw Compensation",
			Properties: map[string]any{
				"event_definition_type": "compensate",
				// activity_ref is optional. If empty, compensates all in scope.
			},
			Outgoing: []model.Transition{{TargetRef: "end"}},
		},
		{ID: "end", Type: model.StepTypeEnd, Name: "End"},
	}

	wf, err := e.DeployWorkflow(context.Background(), "Compensation Test", steps)
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}

	// Start
	instance, err := e.StartInstance(context.Background(), strconv.FormatInt(wf.ID, 10), make(map[string]any))
	if err != nil {
		t.Fatalf("StartInstance failed: %v", err)
	}

	// Verify Service A ran
	if val, ok := instance.Context["serviceA_done"]; !ok || val != true {
		t.Error("Service A should have run")
	}

	// Verify Compensation ran
	// The compensation happens synchronously in TriggerCompensation call within IntermediateThrowEventExecutor
	if val, ok := instance.Context["serviceA_undone"]; !ok || val != true {
		t.Error("Undo Service A should have run")
	}

	// Verify completion
	if instance.Status != model.StatusCompleted {
		t.Errorf("Instance should be completed, got %s", instance.Status)
	}
}
