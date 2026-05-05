package dto

import (
	"fmt"
	"workflow-engine/backend/libs/model"
)

// ToWorkflowResponse maps internal workflow model to API response
func ToWorkflowResponse(w *model.WorkflowDefinition) WorkflowDefinitionResponse {
	return WorkflowDefinitionResponse{
		ID:                  fmt.Sprintf("%d", w.ID),
		ProcessDefinitionID: w.ProcessDefinitionID,
		Name:                w.Name,
		Version:             w.Version,
		ResourceName:        w.ResourceName,
		DeploymentID:        w.DeploymentID,
		TenantID:            w.TenantID,
		ResourceChecksum:    w.ResourceChecksum,
		BPMNXML:             w.BPMNXML,
		Steps:               w.Steps, // Direct copy for now
		CreatedAt:           w.CreatedAt,
	}
}

// ToWorkflowInstanceResponse maps internal instance model to API response
func ToWorkflowInstanceResponse(i *model.WorkflowInstance) WorkflowInstanceResponse {
	return WorkflowInstanceResponse{
		ID:                i.ID,
		WorkflowID:        i.WorkflowID,
		Status:            string(i.Status),
		ParentInstanceID:  i.ParentInstanceID,
		ParentExecutionID: i.ParentExecutionID,
		Context:           i.Context,
		CreatedAt:         i.CreatedAt,
		UpdatedAt:         i.UpdatedAt,
		Executions:        i.Executions, // Direct copy for now
	}
}

// ToJobResponse maps internal job model to API response
func ToJobResponse(j model.Job) JobResponse {
	return JobResponse{
		Key:                  fmt.Sprintf("%d", j.Key),
		Type:                 j.Type,
		ProcessInstanceKey:   fmt.Sprintf("%d", j.ProcessInstanceKey),
		ElementInstanceKey:   fmt.Sprintf("%d", j.ElementInstanceKey),
		ProcessDefinitionKey: fmt.Sprintf("%d", j.ProcessDefinitionKey),
		ElementID:            j.ElementID,
		Worker:               j.Worker,
		Retries:              j.Retries,
		State:                j.State,
		LockExpirationTime:   j.LockExpirationTime,
		CreatedAt:            j.CreatedAt,
		UpdatedAt:            j.UpdatedAt,
	}
}

func ToJobResponses(jobs []model.Job) []JobResponse {
	responses := make([]JobResponse, len(jobs))
	for i, j := range jobs {
		responses[i] = ToJobResponse(j)
	}
	return responses
}
