package application

import (
	"context"
	"workflow-engine/backend/libs/model"
)

func (e *Engine) checkSubProcessCompletion(ctx context.Context, instance *model.WorkflowInstance, completedExecID string, wf *model.WorkflowDefinition) error {
	// Find the completed execution
	var completedExec *model.Execution
	for i := range instance.Executions {
		if instance.Executions[i].ID == completedExecID {
			completedExec = &instance.Executions[i]
			break
		}
	}
	if completedExec == nil || completedExec.ParentID == "" {
		return nil // No parent to notify
	}

	parentID := completedExec.ParentID

	// Check if ALL siblings (same ParentID) are COMPLETED
	allChildrenCompleted := true
	for _, ex := range instance.Executions {
		if ex.ParentID == parentID && ex.Status != "COMPLETED" {
			allChildrenCompleted = false
			break
		}
	}

	if allChildrenCompleted {
		// Find Parent Execution
		var parentExec *model.Execution
		for i := range instance.Executions {
			if instance.Executions[i].ID == parentID {
				parentExec = &instance.Executions[i]
				break
			}
		}

		if parentExec != nil && parentExec.Status == "ACTIVE" {
			// Complete Parent SubProcess Step

			// Determine if this is actually a SubProcess step
			// (It could be a parallel gateway split, but those don't wait in ACTIVE state usually,
			// unless we change how parallel gateway works. Currently ParallelGateway completes immediately
			// and spawns children with parentID=gatewayExecID? No, currently Parallel Gateway logic
			// spawns children with parentID=gatewayExecID's parent. Wait, let's check proceedToken.)

			// In proceedToken:
			// Parallel Gateway: instance.Executions[execIdx].Status = "COMPLETED"; newExec.ParentID = execID (Wait, checks above: ParentID:  execID)
			// Actually, in proceedToken for ParallelGateway:
			// instance.Executions[execIdx].Status = "COMPLETED"
			// newExec.ParentID = execID

			// So ParallelGateway execution IS completed. So this check (parentExec.Status == "ACTIVE")
			// effectively filters out ParallelGateway parents (which are already COMPLETED).
			// It only picks up SubProcess parents which stay ACTIVE while children run.

			// Proceed the SubProcess token
			return e.proceedToken(ctx, instance, parentExec.ID, wf)
		}
	}
	return nil
}
