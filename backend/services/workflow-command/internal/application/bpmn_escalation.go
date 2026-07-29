package application

import "fmt"

// BpmnEscalation is thrown by escalation throw/end events and matched by catch/boundary.
type BpmnEscalation struct {
	EscalationCode string
	Message        string
}

func (e *BpmnEscalation) Error() string {
	return fmt.Sprintf("BPMN Escalation [%s]: %s", e.EscalationCode, e.Message)
}
