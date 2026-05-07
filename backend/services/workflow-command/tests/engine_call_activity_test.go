package tests

import (
	"context"
	"github.com/azizAltaleb/goflow/backend/libs/model"
	"strconv"
	"testing"
	"time"
)

func TestCallActivityExecution(t *testing.T) {
	e := setupTestEngine(t)

	// 1. Deploy Child Process (to be called)
	// Start -> UserTask (childTask) -> End
	childSteps := []model.StepDefinition{
		{
			ID:   "start",
			Type: model.StepTypeStart,
			Name: "Start",
			Outgoing: []model.Transition{
				{TargetRef: "childTask"},
			},
		},
		{
			ID:   "childTask",
			Type: model.StepTypeUserTask,
			Name: "Child Task",
			Outgoing: []model.Transition{
				{TargetRef: "end"},
			},
		},
		{
			ID:       "end",
			Type:     model.StepTypeEnd,
			Name:     "End",
			Outgoing: []model.Transition{},
		},
	}

	childWf, err := e.DeployWorkflow(context.Background(), "Child Process", childSteps)
	if err != nil {
		t.Fatalf("Failed to deploy child workflow: %v", err)
	}
	// Important: The engine uses FindWorkflowByProcessID which looks up by process_definition_id.
	// Our DeployWorkflow sets process_definition_id to a UUID.
	// We need to know this ID to reference it in the parent.
	childProcessID := childWf.ProcessDefinitionID

	// 2. Deploy Parent Process
	// Start -> CallActivity (calls Child) -> UserTask (parentTask) -> End
	parentSteps := []model.StepDefinition{
		{
			ID:   "p_start",
			Type: model.StepTypeStart,
			Name: "Parent Start",
			Outgoing: []model.Transition{
				{TargetRef: "call_child"},
			},
		},
		{
			ID:   "call_child",
			Type: model.StepTypeCallActivity,
			Name: "Call Child Workflow",
			Properties: map[string]any{
				"called_element": childProcessID,
			},
			Outgoing: []model.Transition{
				{TargetRef: "p_task"},
			},
		},
		{
			ID:   "p_task",
			Type: model.StepTypeUserTask,
			Name: "Parent Task",
			Outgoing: []model.Transition{
				{TargetRef: "p_end"},
			},
		},
		{
			ID:       "p_end",
			Type:     model.StepTypeEnd,
			Name:     "Parent End",
			Outgoing: []model.Transition{},
		},
	}

	parentWf, err := e.DeployWorkflow(context.Background(), "Parent Process", parentSteps)
	if err != nil {
		t.Fatalf("Failed to deploy parent workflow: %v", err)
	}

	// 3. Start Parent Process
	parentInstance, err := e.StartInstance(context.Background(), strconv.FormatInt(parentWf.ID, 10), map[string]any{"var1": "value1"})
	if err != nil {
		t.Fatalf("Failed to start parent instance: %v", err)
	}

	// 4. Verify Parent is waiting at CallActivity
	// Since CallActivity starts the child and returns (child hits user task), parent should be ACTIVE at "call_child"
	parentInstance, err = e.GetInstance(context.Background(), parentInstance.ID)
	if err != nil {
		t.Fatalf("Failed to get parent instance: %v", err)
	}
	if parentInstance.Status != model.StatusRunning {
		t.Errorf("Parent instance status expected RUNNING, got %s", parentInstance.Status)
	}

	currentSteps := getCurrentSteps(parentInstance)
	if !contains(currentSteps, "call_child") {
		t.Errorf("Parent expected to be at 'call_child', got %v", currentSteps)
	}

	// 5. Find Child Instance
	// We can list active instances and find one with ParentInstanceID == parentInstance.ID
	instances, err := e.ListActiveInstances(context.Background())
	if err != nil {
		t.Fatalf("Failed to list instances: %v", err)
	}

	var childSummary *model.WorkflowInstance
	for _, inst := range instances {
		if inst.ParentInstanceID == parentInstance.ID {
			childSummary = inst
			break
		}
	}

	if childSummary == nil {
		t.Fatalf("Child instance not found for parent %s", parentInstance.ID)
	}

	// Reload child instance to get full details (context, executions)
	childInstance, err := e.GetInstance(context.Background(), childSummary.ID)
	if err != nil {
		t.Fatalf("Failed to get child instance: %v", err)
	}

	// Verify Child State
	if childInstance.WorkflowID != strconv.FormatInt(childWf.ID, 10) {
		t.Errorf("Child instance has wrong workflow ID: %s", childInstance.WorkflowID)
	}
	// Verify Context Inheritance
	if val, ok := childInstance.Context["var1"].(string); !ok || val != "value1" {
		t.Errorf("Child instance missing inherited context var1")
	}

	childStepsCurr := getCurrentSteps(childInstance)
	if !contains(childStepsCurr, "childTask") {
		t.Errorf("Child expected to be at 'childTask', got %v", childStepsCurr)
	}

	// 6. Complete Child Task
	if err := e.CompleteTask(context.Background(), childInstance.ID, "childTask"); err != nil {
		t.Fatalf("Failed to complete child task: %v", err)
	}

	// 7. Verify Child is Completed
	childInstance, err = e.GetInstance(context.Background(), childInstance.ID)
	if err != nil {
		t.Fatalf("Failed to get child instance: %v", err)
	}
	if childInstance.Status != model.StatusCompleted {
		t.Errorf("Child instance expected COMPLETED, got %s", childInstance.Status)
	}

	// 8. Verify Parent Moved to Next Step
	// Need to reload parent instance
	// Give a tiny bit of time if there was any async saving (though we use synchronous logic mostly)
	time.Sleep(10 * time.Millisecond)

	parentInstance, err = e.GetInstance(context.Background(), parentInstance.ID)
	if err != nil {
		t.Fatalf("Failed to get parent instance: %v", err)
	}

	currentSteps = getCurrentSteps(parentInstance)
	if contains(currentSteps, "call_child") {
		t.Error("Parent should not be at 'call_child' anymore")
	}
	if !contains(currentSteps, "p_task") {
		t.Errorf("Parent expected to be at 'p_task', got %v", currentSteps)
	}

	// 9. Complete Parent Task
	if err := e.CompleteTask(context.Background(), parentInstance.ID, "p_task"); err != nil {
		t.Fatalf("Failed to complete parent task: %v", err)
	}

	// 10. Verify Parent Completed
	parentInstance, err = e.GetInstance(context.Background(), parentInstance.ID)
	if err != nil {
		t.Fatalf("Failed to get parent instance: %v", err)
	}
	if parentInstance.Status != model.StatusCompleted {
		t.Errorf("Parent instance expected COMPLETED, got %s", parentInstance.Status)
	}
}

func TestCallActivity_ImmediateChild(t *testing.T) {
	e := setupTestEngine(t)

	// 1. Deploy Immediate Child Process (Start -> End)
	childSteps := []model.StepDefinition{
		{
			ID:   "start",
			Type: model.StepTypeStart,
			Name: "Start",
			Outgoing: []model.Transition{
				{TargetRef: "end"},
			},
		},
		{
			ID:       "end",
			Type:     model.StepTypeEnd,
			Name:     "End",
			Outgoing: []model.Transition{},
		},
	}
	childWf, _ := e.DeployWorkflow(context.Background(), "Immediate Child", childSteps)

	// 2. Deploy Parent Process
	parentSteps := []model.StepDefinition{
		{
			ID:   "start",
			Type: model.StepTypeStart,
			Name: "Start",
			Outgoing: []model.Transition{
				{TargetRef: "call_child"},
			},
		},
		{
			ID:   "call_child",
			Type: model.StepTypeCallActivity,
			Name: "Call Child",
			Properties: map[string]any{
				"called_element": childWf.ProcessDefinitionID,
			},
			Outgoing: []model.Transition{
				{TargetRef: "end"},
			},
		},
		{
			ID:       "end",
			Type:     model.StepTypeEnd,
			Name:     "End",
			Outgoing: []model.Transition{},
		},
	}
	parentWf, _ := e.DeployWorkflow(context.Background(), "Parent", parentSteps)

	// 3. Start Parent
	// Should run all the way to completion immediately
	parentInstance, err := e.StartInstance(context.Background(), strconv.FormatInt(parentWf.ID, 10), nil)
	if err != nil {
		t.Fatalf("StartInstance failed: %v", err)
	}

	if parentInstance.Status != model.StatusCompleted {
		t.Errorf("Parent instance expected COMPLETED, got %s. Executions: %+v", parentInstance.Status, parentInstance.Executions)
	}
}
