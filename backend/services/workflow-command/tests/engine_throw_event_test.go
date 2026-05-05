package tests

import (
	"context"
	"strconv"
	"testing"
	"workflow-engine/backend/libs/model"
)

func TestIntermediateThrowEventExecution(t *testing.T) {
	e := setupTestEngine(t)

	// Workflow 1: Receiver (Waits for Signal)
	receiverSteps := []model.StepDefinition{
		{ID: "start", Type: model.StepTypeStart, Outgoing: []model.Transition{{TargetRef: "catch"}}},
		{
			ID:   "catch",
			Type: model.StepTypeIntermediateCatchEvent,
			Properties: map[string]any{
				"signal_ref": "GlobalSignal",
			},
			Incoming: []string{"start"},
			Outgoing: []model.Transition{{TargetRef: "end"}},
		},
		{ID: "end", Type: model.StepTypeEnd, Incoming: []string{"catch"}},
	}
	wfReceiver, err := e.DeployWorkflow(context.Background(), "Receiver", receiverSteps)
	if err != nil {
		t.Fatalf("Deploy Receiver failed: %v", err)
	}

	// Start Receiver Instance
	instReceiver, err := e.StartInstance(context.Background(), strconv.FormatInt(wfReceiver.ID, 10), nil)
	if err != nil {
		t.Fatalf("Start Receiver failed: %v", err)
	}

	// Verify Receiver is waiting
	instReceiver, _ = e.GetInstance(context.Background(), instReceiver.ID)
	if instReceiver.Status != model.StatusRunning {
		t.Errorf("Receiver should be RUNNING, got %s", instReceiver.Status)
	}

	// Workflow 2: Sender (Throws Signal)
	senderSteps := []model.StepDefinition{
		{ID: "start", Type: model.StepTypeStart, Outgoing: []model.Transition{{TargetRef: "throw"}}},
		{
			ID:   "throw",
			Type: model.StepTypeIntermediateThrowEvent,
			Properties: map[string]any{
				"signal_ref": "GlobalSignal",
			},
			Incoming: []string{"start"},
			Outgoing: []model.Transition{{TargetRef: "end"}},
		},
		{ID: "end", Type: model.StepTypeEnd, Incoming: []string{"throw"}},
	}
	wfSender, err := e.DeployWorkflow(context.Background(), "Sender", senderSteps)
	if err != nil {
		t.Fatalf("Deploy Sender failed: %v", err)
	}

	// Start Sender Instance
	instSender, err := e.StartInstance(context.Background(), strconv.FormatInt(wfSender.ID, 10), nil)
	if err != nil {
		t.Fatalf("Start Sender failed: %v", err)
	}

	// Verify Sender Completed (it should pass through throw event)
	instSender, _ = e.GetInstance(context.Background(), instSender.ID)
	if instSender.Status != model.StatusCompleted {
		t.Errorf("Sender should be COMPLETED, got %s", instSender.Status)
	}

	// Verify Receiver Completed (signal triggered)
	// Give it a tiny bit of time if async, but our engine is mostly sync for this
	instReceiver, _ = e.GetInstance(context.Background(), instReceiver.ID)
	if instReceiver.Status != model.StatusCompleted {
		t.Errorf("Receiver should be COMPLETED, got %s. Executions: %+v", instReceiver.Status, instReceiver.Executions)
	}
}

func TestIntermediateThrowMessageExecution(t *testing.T) {
	e := setupTestEngine(t)

	// Workflow 1: Receiver (Waits for Message)
	receiverSteps := []model.StepDefinition{
		{ID: "start", Type: model.StepTypeStart, Outgoing: []model.Transition{{TargetRef: "catch"}}},
		{
			ID:   "catch",
			Type: model.StepTypeIntermediateCatchEvent,
			Properties: map[string]any{
				"message_ref": "GlobalMessage",
			},
			Incoming: []string{"start"},
			Outgoing: []model.Transition{{TargetRef: "end"}},
		},
		{ID: "end", Type: model.StepTypeEnd, Incoming: []string{"catch"}},
	}
	wfReceiver, err := e.DeployWorkflow(context.Background(), "Receiver", receiverSteps)
	if err != nil {
		t.Fatalf("Deploy Receiver failed: %v", err)
	}

	instReceiver, err := e.StartInstance(context.Background(), strconv.FormatInt(wfReceiver.ID, 10), nil)
	if err != nil {
		t.Fatalf("Start Receiver failed: %v", err)
	}

	// Workflow 2: Sender (Throws Message)
	senderSteps := []model.StepDefinition{
		{ID: "start", Type: model.StepTypeStart, Outgoing: []model.Transition{{TargetRef: "throw"}}},
		{
			ID:   "throw",
			Type: model.StepTypeIntermediateThrowEvent,
			Properties: map[string]any{
				"message_ref": "GlobalMessage",
				"payload": map[string]any{ // Optional payload simulation if we support it in properties later
					"key": "value",
				},
			},
			Incoming: []string{"start"},
			Outgoing: []model.Transition{{TargetRef: "end"}},
		},
		{ID: "end", Type: model.StepTypeEnd, Incoming: []string{"throw"}},
	}
	wfSender, err := e.DeployWorkflow(context.Background(), "Sender", senderSteps)
	if err != nil {
		t.Fatalf("Deploy Sender failed: %v", err)
	}

	instSender, _ := e.StartInstance(context.Background(), strconv.FormatInt(wfSender.ID, 10), nil)

	// Verify Sender Completed
	instSender, _ = e.GetInstance(context.Background(), instSender.ID)
	if instSender.Status != model.StatusCompleted {
		t.Errorf("Sender should be COMPLETED, got %s", instSender.Status)
	}

	// Verify Receiver Completed
	instReceiver, _ = e.GetInstance(context.Background(), instReceiver.ID)
	if instReceiver.Status != model.StatusCompleted {
		t.Errorf("Receiver should be COMPLETED, got %s", instReceiver.Status)
	}
}
