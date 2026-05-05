package tests

import (
	"context"
	"strconv"
	"testing"
	"workflow-engine/backend/libs/model"
)

func TestMessageEventExecution(t *testing.T) {
	e := setupTestEngine(t)

	// Flow: Start -> Wait for Message (MsgPaymentReceived) -> End
	steps := []model.StepDefinition{
		{ID: "start", Type: model.StepTypeStart, Name: "Start", Outgoing: []model.Transition{{TargetRef: "catch"}}},
		{
			ID:   "catch",
			Type: model.StepTypeIntermediateCatchEvent,
			Name: "Catch Message",
			Properties: map[string]any{
				"message_ref": "MsgPaymentReceived",
			},
			Incoming: []string{"start"},
			Outgoing: []model.Transition{{TargetRef: "end"}},
		},
		{ID: "end", Type: model.StepTypeEnd, Name: "End", Incoming: []string{"catch"}},
	}

	wf, err := e.DeployWorkflow(context.Background(), "Message Test", steps)
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}

	// Start Instance
	instance, err := e.StartInstance(context.Background(), strconv.FormatInt(wf.ID, 10), nil)
	if err != nil {
		t.Fatalf("StartInstance failed: %v", err)
	}

	// 1. Verify waiting at Catch
	current := getCurrentSteps(instance)
	if len(current) != 1 || !contains(current, "catch") {
		t.Errorf("Expected wait at 'catch', got %v", current)
	}

	// 2. Publish Message with Payload
	payload := map[string]any{"paymentId": "12345"}
	if err := e.PublishMessage(context.Background(), "MsgPaymentReceived", "", payload); err != nil {
		t.Fatalf("PublishMessage failed: %v", err)
	}

	// Reload
	instance, _ = e.GetInstance(context.Background(), instance.ID)
	if instance.Status != model.StatusCompleted {
		t.Errorf("Expected COMPLETED, got %s. Executions: %+v", instance.Status, instance.Executions)
	}

	// Verify payload merge
	if val, ok := instance.Context["paymentId"]; !ok || val != "12345" {
		t.Errorf("Expected payload 'paymentId'='12345', got %v", val)
	}
}
