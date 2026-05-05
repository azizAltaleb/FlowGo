package application

import (
	"errors"
	"testing"
	"time"

	"github.com/dop251/goja"
)

func TestEvaluateJavaScriptExpressionSuccess(t *testing.T) {
	vm, val, err := evaluateJavaScriptExpression("x + 1", map[string]any{"x": 2}, 100*time.Millisecond, expressionKindScript)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if vm == nil {
		t.Fatalf("expected runtime to be returned")
	}
	if val == nil {
		t.Fatalf("expected value to be returned")
	}
	if got := val.ToInteger(); got != 3 {
		t.Fatalf("expected result 3, got %d", got)
	}
}

func TestEvaluateJavaScriptExpressionReturnsRuntimeErrorCode(t *testing.T) {
	_, _, err := evaluateJavaScriptExpression("throw new Error('boom')", nil, 100*time.Millisecond, expressionKindScript)
	if err == nil {
		t.Fatalf("expected runtime error")
	}

	var evalErr *ExpressionEvaluationError
	if !errors.As(err, &evalErr) {
		t.Fatalf("expected ExpressionEvaluationError, got %T", err)
	}
	if evalErr.Kind != expressionKindScript {
		t.Fatalf("expected kind %q, got %q", expressionKindScript, evalErr.Kind)
	}
	if evalErr.Code != expressionErrorCodeRuntime {
		t.Fatalf("expected code %q, got %q", expressionErrorCodeRuntime, evalErr.Code)
	}
	if evalErr.Cause == nil {
		t.Fatalf("expected wrapped cause")
	}
}

func TestEvaluateJavaScriptExpressionReturnsTimeoutErrorCode(t *testing.T) {
	start := time.Now()
	_, _, err := evaluateJavaScriptExpression("for(;;){}", nil, 20*time.Millisecond, expressionKindCondition)
	if err == nil {
		t.Fatalf("expected timeout error")
	}

	var evalErr *ExpressionEvaluationError
	if !errors.As(err, &evalErr) {
		t.Fatalf("expected ExpressionEvaluationError, got %T", err)
	}
	if evalErr.Kind != expressionKindCondition {
		t.Fatalf("expected kind %q, got %q", expressionKindCondition, evalErr.Kind)
	}
	if evalErr.Code != expressionErrorCodeTimeout {
		t.Fatalf("expected code %q, got %q", expressionErrorCodeTimeout, evalErr.Code)
	}
	if !evalErr.IsTimeout() {
		t.Fatalf("expected IsTimeout helper to return true")
	}

	var interruptedErr *goja.InterruptedError
	if !errors.As(err, &interruptedErr) {
		t.Fatalf("expected wrapped interrupted error, got %T", err)
	}

	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("timeout evaluation took too long: %v", elapsed)
	}
}
