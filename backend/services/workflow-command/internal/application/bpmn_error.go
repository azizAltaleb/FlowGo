package application

import "fmt"

// BpmnError represents a business error thrown by the process (e.g. Error End Event)
type BpmnError struct {
	ErrorCode    string
	ErrorMessage string
}

func (e *BpmnError) Error() string {
	return fmt.Sprintf("BPMN Error [%s]: %s", e.ErrorCode, e.ErrorMessage)
}
