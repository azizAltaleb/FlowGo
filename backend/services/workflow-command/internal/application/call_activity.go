package application

import (
	"context"
	"fmt"
	"github.com/azizAltaleb/goflow/backend/libs/model"
	"strconv"
	"time"
)

func (e *Engine) notifyParentProcess(ctx context.Context, childInstance *model.WorkflowInstance) error {
	if childInstance.ParentInstanceID == "" || childInstance.ParentExecutionID == "" {
		return nil
	}

	// Use e.GetInstance to support Engine schema
	parentInstance, err := e.GetInstance(ctx, childInstance.ParentInstanceID)
	if err != nil {
		return fmt.Errorf("failed to load parent instance %s: %v", childInstance.ParentInstanceID, err)
	}

	// Verify parent execution is still active
	var parentExec *model.Execution
	for i := range parentInstance.Executions {
		if parentInstance.Executions[i].ID == childInstance.ParentExecutionID {
			parentExec = &parentInstance.Executions[i]
			break
		}
	}

	if parentExec == nil {
		return fmt.Errorf("parent execution %s not found in instance %s", childInstance.ParentExecutionID, parentInstance.ID)
	}

	if parentExec.Status != "ACTIVE" {
		// Already completed?
		return nil
	}

	// Load parent workflow definition
	parentWfID, _ := strconv.ParseInt(parentInstance.WorkflowID, 10, 64)
	parentWf, err := e.getWorkflowDefinition(ctx, parentWfID)
	if err != nil {
		return err
	}

	// Update Parent Execution Status?
	// Actually, the parent execution was waiting in "ACTIVE" status at the Call Activity step.
	// Now that child is done, we proceed the token.

	if err := e.proceedToken(ctx, parentInstance, parentExec.ID, parentWf); err != nil {
		return err
	}

	// Engine: Check Completion of Parent Instance
	if parentInstance.Status == model.StatusCompleted {
		if piKey, err := strconv.ParseInt(parentInstance.ID, 10, 64); err == nil {
			pi := &model.ProcessInstance{
				Key:     piKey,
				State:   "COMPLETED",
				EndTime: time.Now(),
			}
			if err := e.repo.UpdateProcessInstance(ctx, pi); err != nil {
				// log error?
			}
		}
	}

	return nil
}
