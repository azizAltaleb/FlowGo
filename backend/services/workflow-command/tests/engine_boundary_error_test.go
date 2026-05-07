package tests

import (
	"context"
	"github.com/azizAltaleb/goflow/backend/libs/model"
	"github.com/azizAltaleb/goflow/backend/services/workflow-command/internal/application"
	"strconv"
	"testing"
)

func TestBoundaryErrorEventExecution(t *testing.T) {
	e := setupTestEngine(t)

	// Register a failing service handler that throws a BPMN Error
	e.RegisterHandler("failingService", func(ctx context.Context, instance *model.WorkflowInstance, step *model.StepDefinition) error {
		return &application.BpmnError{
			ErrorCode:    "MyError",
			ErrorMessage: "Something went wrong",
		}
	})

	// Workflow: Start -> ServiceTask (Boundary Error -> ErrorHandler) -> End
	//                                |-> NormalEnd
	steps := []model.StepDefinition{
		{ID: "start", Type: model.StepTypeStart, Outgoing: []model.Transition{{TargetRef: "serviceTask"}}},
		{
			ID:                "serviceTask",
			Type:              model.StepTypeServiceTask,
			Name:              "Failing Service",
			Implementation:    "failingService",
			BoundaryEventRefs: []string{"boundaryError"},
			Outgoing:          []model.Transition{{TargetRef: "normalEnd"}},
		},
		{
			ID:   "boundaryError",
			Type: model.StepTypeBoundaryEvent,
			Properties: map[string]any{
				"error_ref":       "MyError", // For now we might just catch all, or match string
				"cancel_activity": true,
				"attached_to":     "serviceTask",
			},
			Outgoing: []model.Transition{{TargetRef: "errorHandler"}},
		},
		{ID: "normalEnd", Type: model.StepTypeEnd, Name: "Normal End"},
		{ID: "errorHandler", Type: model.StepTypeScriptTask, Name: "Error Handler", Outgoing: []model.Transition{{TargetRef: "errorEnd"}}},
		{ID: "errorEnd", Type: model.StepTypeEnd, Name: "Error End"},
	}

	wf, err := e.DeployWorkflow(context.Background(), "Boundary Error Test", steps)
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}

	// Start Instance
	instance, err := e.StartInstance(context.Background(), strconv.FormatInt(wf.ID, 10), nil)
	if err != nil {
		t.Fatalf("StartInstance failed: %v", err)
	}

	// Verify Transition
	instance, _ = e.GetInstance(context.Background(), instance.ID)

	// ServiceTask execution should be COMPLETED (interrupted/failed but moved to error path)
	// Actually, if it fails, the token at ServiceTask might be updated to COMPLETED or TERMINATED by the error handler
	// ErrorHandler execution should be COMPLETED (auto script)
	// ErrorEnd should be COMPLETED

	foundErrorEnd := false
	normalEndReached := false

	for _, ex := range instance.Executions {
		if ex.StepID == "errorEnd" && ex.Status == "COMPLETED" {
			foundErrorEnd = true
		}
		if ex.StepID == "normalEnd" {
			normalEndReached = true
		}
	}

	if normalEndReached {
		t.Errorf("Normal End should not be reached")
	}
	if !foundErrorEnd {
		t.Errorf("Expected execution to reach errorEnd. Executions: %+v", instance.Executions)
	}
}
