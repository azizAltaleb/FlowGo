package application

import (
	"errors"
	"fmt"
	"time"

	"github.com/dop251/goja"
)

const (
	defaultGatewayConditionTimeout = 1 * time.Second
	defaultScriptEvaluationTimeout = 5 * time.Second
)

type expressionKind string

const (
	expressionKindCondition expressionKind = "condition"
	expressionKindScript    expressionKind = "script"
)

type expressionErrorCode string

const (
	expressionErrorCodeTimeout expressionErrorCode = "timeout"
	expressionErrorCodeRuntime expressionErrorCode = "runtime"
)

// ExpressionEvaluationError describes expression execution failures with stable error codes.
type ExpressionEvaluationError struct {
	Kind  expressionKind
	Code  expressionErrorCode
	Cause error
}

func (e *ExpressionEvaluationError) Error() string {
	if e == nil {
		return ""
	}
	if e.Cause == nil {
		return fmt.Sprintf("%s evaluation %s", e.Kind, e.Code)
	}
	return fmt.Sprintf("%s evaluation %s: %v", e.Kind, e.Code, e.Cause)
}

func (e *ExpressionEvaluationError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func (e *ExpressionEvaluationError) IsTimeout() bool {
	return e != nil && e.Code == expressionErrorCodeTimeout
}

func evaluateJavaScriptExpression(expression string, contextVars map[string]any, timeout time.Duration, kind expressionKind) (*goja.Runtime, goja.Value, error) {
	if timeout <= 0 {
		switch kind {
		case expressionKindCondition:
			timeout = defaultGatewayConditionTimeout
		default:
			timeout = defaultScriptEvaluationTimeout
		}
	}

	vm := goja.New()

	// Enforce strict mode by default
	// We wrap the expression in a function to ensure 'use strict' applies correctly if needed,
	// or just prepend it. For simple expressions, prepending might be enough, but for scripts it's safer.
	// However, simple expressions like "x > 5" won't work if just prepended with "use strict";
	// because "use strict"; x > 5 is a valid statement list.
	// But let's be careful not to break return values.
	// If it's a script (multi-line), we might want to wrap.
	// For now, let's keep it simple and just run it. Strict mode in Goja is enabled by default for modules, but here we run string.
	// We can enable it by adding "use strict"; at the beginning.
	script := "\"use strict\";\n" + expression

	for k, v := range contextVars {
		if err := vm.Set(k, v); err != nil {
			return vm, nil, &ExpressionEvaluationError{
				Kind:  kind,
				Code:  expressionErrorCodeRuntime,
				Cause: fmt.Errorf("set variable %q: %w", k, err),
			}
		}
	}

	// Setup Interrupt for Timeout
	timer := time.AfterFunc(timeout, func() {
		vm.Interrupt(fmt.Sprintf("%s evaluation timeout", kind))
	})
	defer timer.Stop()

	val, err := vm.RunString(script)
	if err != nil {
		code := expressionErrorCodeRuntime
		var interruptedErr *goja.InterruptedError
		// Goja returns *InterruptedError if Interrupt() was called
		if errors.As(err, &interruptedErr) {
			code = expressionErrorCodeTimeout
		}

		return vm, nil, &ExpressionEvaluationError{
			Kind:  kind,
			Code:  code,
			Cause: err,
		}
	}

	return vm, val, nil
}
