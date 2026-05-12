package tests

import (
	"context"
	"github.com/azizAltaleb/flowgo/backend/libs/model"
	"strconv"
	"testing"
	"time"
)

func TestBoundaryTimerEventExecution(t *testing.T) {
	e := setupTestEngine(t)

	// Workflow: Start -> UserTask (Boundary Timer -> TimeoutTask) -> End
	//                               |-> End
	steps := []model.StepDefinition{
		{ID: "start", Type: model.StepTypeStart, Outgoing: []model.Transition{{TargetRef: "userTask"}}},
		{
			ID:                "userTask",
			Type:              model.StepTypeUserTask,
			Name:              "User Task",
			BoundaryEventRefs: []string{"boundaryTimer"},
			Outgoing:          []model.Transition{{TargetRef: "normalEnd"}},
		},
		{
			ID:   "boundaryTimer",
			Type: model.StepTypeBoundaryEvent,
			Properties: map[string]any{
				"timer_duration":  "PT1S",
				"cancel_activity": true,
				"attached_to":     "userTask",
			},
			Outgoing: []model.Transition{{TargetRef: "timeoutTask"}},
		},
		{ID: "normalEnd", Type: model.StepTypeEnd, Name: "Normal End"},
		{ID: "timeoutTask", Type: model.StepTypeScriptTask, Name: "Timeout Task", Outgoing: []model.Transition{{TargetRef: "timeoutEnd"}}},
		{ID: "timeoutEnd", Type: model.StepTypeEnd, Name: "Timeout End"},
	}

	wf, err := e.DeployWorkflow(context.Background(), "Boundary Timer Test", steps)
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}

	// Start Instance
	instance, err := e.StartInstance(context.Background(), strconv.FormatInt(wf.ID, 10), nil)
	if err != nil {
		t.Fatalf("StartInstance failed: %v", err)
	}

	// Verify at User Task
	instance, _ = e.GetInstance(context.Background(), instance.ID)
	var userExec *model.Execution
	for i := range instance.Executions {
		if instance.Executions[i].StepID == "userTask" && instance.Executions[i].Status == "ACTIVE" {
			userExec = &instance.Executions[i]
			break
		}
	}
	if userExec == nil {
		t.Fatalf("Expected execution at userTask")
	}

	// Wait for timer duration
	time.Sleep(1100 * time.Millisecond)

	// Trigger CheckTimers
	if err := e.CheckTimers(context.Background()); err != nil {
		t.Fatalf("CheckTimers failed: %v", err)
	}

	// Verify Transition
	instance, _ = e.GetInstance(context.Background(), instance.ID)

	// UserTask execution should be COMPLETED (interrupted)
	// TimeoutTask execution should be COMPLETED (since it's a script task, it runs auto)
	// TimeoutEnd should be COMPLETED

	foundTimeout := false
	userTaskActive := false

	for _, ex := range instance.Executions {
		if ex.StepID == "userTask" && ex.Status == "ACTIVE" {
			userTaskActive = true
		}
		if ex.StepID == "timeoutEnd" && ex.Status == "COMPLETED" {
			foundTimeout = true
		}
	}

	if userTaskActive {
		t.Errorf("User Task should be interrupted (not active)")
	}
	if !foundTimeout {
		t.Errorf("Expected execution to reach timeoutEnd. Executions: %+v", instance.Executions)
	}
}
