package application

import (
	"context"
	"testing"
	"time"
)

func TestEvaluateConditionReturnsTrueForValidExpression(t *testing.T) {
	e := &Engine{}
	if !e.evaluateCondition(context.Background(), "x > 1", map[string]any{"x": 2}) {
		t.Fatalf("expected condition to evaluate to true")
	}
}

func TestEvaluateConditionReturnsTrueForBooleanComparison(t *testing.T) {
	e := &Engine{}
	if !e.evaluateCondition(context.Background(), "x == true", map[string]any{"x": true}) {
		t.Fatalf("expected boolean condition to evaluate to true")
	}
}

func TestEvaluateConditionNormalizesSingleEqualsComparison(t *testing.T) {
	e := &Engine{}
	if !e.evaluateCondition(context.Background(), "x = true", map[string]any{"x": true}) {
		t.Fatalf("expected normalized single-equals condition to evaluate to true")
	}
}

func TestEvaluateConditionTimeoutReturnsFalse(t *testing.T) {
	e := &Engine{}
	start := time.Now()

	ok := e.evaluateCondition(context.Background(), "for(;;){}", map[string]any{})
	if ok {
		t.Fatalf("expected timed-out condition to evaluate to false")
	}

	elapsed := time.Since(start)
	if elapsed > defaultGatewayConditionTimeout+500*time.Millisecond {
		t.Fatalf("condition evaluation timeout took too long: %v", elapsed)
	}
}
