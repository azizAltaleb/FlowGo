package tests

import (
	"context"
	"strconv"
	"testing"
	"workflow-engine/backend/libs/model"
)

func TestEventBasedGateway_SignalTrigger(t *testing.T) {
	e := setupTestEngine(t)

	// Flow: Start -> Event Gateway -> (Signal Catch / Timer Catch)
	// If Signal triggers, Timer path should be cancelled.

	steps := []model.StepDefinition{
		{ID: "start", Type: model.StepTypeStart, Name: "Start", Outgoing: []model.Transition{{TargetRef: "event-gateway"}}},

		{ID: "event-gateway", Type: model.StepTypeGatewayEventBased, Name: "Event Gateway",
			Incoming: []string{"start"},
			Outgoing: []model.Transition{
				{TargetRef: "signal-catch"},
				{TargetRef: "timer-catch"},
			},
		},

		// Branch 1: Signal
		{ID: "signal-catch", Type: model.StepTypeIntermediateCatchEvent, Name: "Catch Signal",
			Properties: map[string]any{"signal_ref": "MySignal"},
			Incoming:   []string{"event-gateway"},
			Outgoing:   []model.Transition{{TargetRef: "task-signal"}},
		},
		{ID: "task-signal", Type: model.StepTypeUserTask, Name: "Task After Signal",
			Incoming: []string{"signal-catch"},
			Outgoing: []model.Transition{{TargetRef: "end"}},
		},

		// Branch 2: Timer (Long duration to ensure it doesn't fire first)
		{ID: "timer-catch", Type: model.StepTypeIntermediateTimerCatchEvent, Name: "Catch Timer",
			Properties: map[string]any{"duration": "PT1H"}, // 1 Hour
			Incoming:   []string{"event-gateway"},
			Outgoing:   []model.Transition{{TargetRef: "task-timer"}},
		},
		{ID: "task-timer", Type: model.StepTypeUserTask, Name: "Task After Timer",
			Incoming: []string{"timer-catch"},
			Outgoing: []model.Transition{{TargetRef: "end"}},
		},

		{ID: "end", Type: model.StepTypeEnd, Name: "End", Incoming: []string{"task-signal", "task-timer"}},
	}

	wf, err := e.DeployWorkflow(context.Background(), "Event Gateway Test", steps)
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}

	// Start Instance
	instance, err := e.StartInstance(context.Background(), strconv.FormatInt(wf.ID, 10), nil)
	if err != nil {
		t.Fatalf("StartInstance failed: %v", err)
	}

	// Should be waiting at BOTH signal-catch and timer-catch
	// Check Executions
	current := getCurrentSteps(instance)
	if len(current) != 2 {
		t.Errorf("Expected 2 active executions (signal and timer), got %d: %v", len(current), current)
	}
	if !contains(current, "signal-catch") || !contains(current, "timer-catch") {
		t.Errorf("Expected execution at signal-catch and timer-catch, got %v", current)
	}

	// Trigger Signal
	err = e.PublishSignal(context.Background(), "MySignal", nil)
	if err != nil {
		t.Fatalf("PublishSignal failed: %v", err)
	}

	// Refresh instance
	instance, _ = e.GetInstance(context.Background(), instance.ID)

	// Now:
	// 1. "signal-catch" should have completed and moved to "task-signal"
	// 2. "timer-catch" should be TERMINATED (cancelled)

	currentActive := getCurrentSteps(instance)
	if len(currentActive) != 1 {
		t.Errorf("Expected 1 active execution, got %d: %v", len(currentActive), currentActive)
	}
	if !contains(currentActive, "task-signal") {
		t.Errorf("Expected active execution at 'task-signal', got %v", currentActive)
	}

	// Check that timer-catch execution is TERMINATED
	var timerExec *model.Execution
	for _, ex := range instance.Executions {
		if ex.StepID == "timer-catch" {
			timerExec = &ex
			break
		}
	}

	if timerExec == nil {
		t.Fatal("Timer execution not found (it should exist but be TERMINATED)")
	}
	if timerExec.Status != "TERMINATED" {
		t.Errorf("Expected Timer execution to be TERMINATED, got %s", timerExec.Status)
	}
}

func TestEventBasedGateway_TimerTrigger(t *testing.T) {
	e := setupTestEngine(t)

	// Flow: Start -> Event Gateway -> (Signal Catch / Timer Catch)
	// If Timer triggers, Signal path should be cancelled.

	steps := []model.StepDefinition{
		{ID: "start", Type: model.StepTypeStart, Name: "Start", Outgoing: []model.Transition{{TargetRef: "event-gateway"}}},

		{ID: "event-gateway", Type: model.StepTypeGatewayEventBased, Name: "Event Gateway",
			Incoming: []string{"start"},
			Outgoing: []model.Transition{
				{TargetRef: "signal-catch"},
				{TargetRef: "timer-catch"},
			},
		},

		// Branch 1: Signal
		{ID: "signal-catch", Type: model.StepTypeIntermediateCatchEvent, Name: "Catch Signal",
			Properties: map[string]any{"signal_ref": "MySignal"},
			Incoming:   []string{"event-gateway"},
			Outgoing:   []model.Transition{{TargetRef: "task-signal"}},
		},
		{ID: "task-signal", Type: model.StepTypeUserTask, Name: "Task After Signal",
			Incoming: []string{"signal-catch"},
			Outgoing: []model.Transition{{TargetRef: "end"}},
		},

		// Branch 2: Timer (Short duration)
		{ID: "timer-catch", Type: model.StepTypeIntermediateTimerCatchEvent, Name: "Catch Timer",
			Properties: map[string]any{"timer_duration": "PT0S"}, // Immediate
			Incoming:   []string{"event-gateway"},
			Outgoing:   []model.Transition{{TargetRef: "task-timer"}},
		},
		{ID: "task-timer", Type: model.StepTypeUserTask, Name: "Task After Timer",
			Incoming: []string{"timer-catch"},
			Outgoing: []model.Transition{{TargetRef: "end"}},
		},

		{ID: "end", Type: model.StepTypeEnd, Name: "End", Incoming: []string{"task-signal", "task-timer"}},
	}

	wf, err := e.DeployWorkflow(context.Background(), "Event Gateway Timer Test", steps)
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}

	// Start Instance
	instance, err := e.StartInstance(context.Background(), strconv.FormatInt(wf.ID, 10), nil)
	if err != nil {
		t.Fatalf("StartInstance failed: %v", err)
	}

	// Wait for timer to check (since we used PT0S, it might need a CheckTimers call)
	// CheckTimers uses time.Now(), so it should pick it up.
	err = e.CheckTimers(context.Background())
	if err != nil {
		t.Fatalf("CheckTimers failed: %v", err)
	}

	// Refresh instance
	instance, _ = e.GetInstance(context.Background(), instance.ID)

	// Now:
	// 1. "timer-catch" should have completed and moved to "task-timer"
	// 2. "signal-catch" should be TERMINATED

	currentActive := getCurrentSteps(instance)
	if len(currentActive) != 1 {
		// It might be possible that CheckTimers hasn't picked it up if duration calc puts it slightly in future?
		// But PT0S should be immediate.
		t.Errorf("Expected 1 active execution, got %d: %v", len(currentActive), currentActive)
	}
	if !contains(currentActive, "task-timer") {
		t.Errorf("Expected active execution at 'task-timer', got %v", currentActive)
	}

	// Check that signal-catch execution is TERMINATED
	var signalExec *model.Execution
	for _, ex := range instance.Executions {
		if ex.StepID == "signal-catch" {
			signalExec = &ex
			break
		}
	}

	if signalExec == nil {
		t.Fatal("Signal execution not found (it should exist but be TERMINATED)")
	}
	if signalExec.Status != "TERMINATED" {
		t.Errorf("Expected Signal execution to be TERMINATED, got %s", signalExec.Status)
	}
}
