package repository

import (
	"context"
	"errors"

	"github.com/azizAltaleb/goflow/backend/libs/model"
)

var (
	ErrWorkflowNotFound = errors.New("workflow not found")
	ErrInstanceNotFound = errors.New("instance not found")
)

type QueryRepository interface {
	SearchInstances(ctx context.Context, filter InstanceFilter) (*InstanceSearchResult, error)
	SearchWorkflows(ctx context.Context, filter WorkflowFilter) (*WorkflowSearchResult, error)
	GetInstance(ctx context.Context, id string) (*model.ProcessInstance, error)
}

type InstanceFilter struct {
	WorkflowID string
	State      string
	Page       int
	PageSize   int
}

type InstanceSearchResult struct {
	Instances []model.ProcessInstance `json:"instances"`
	Total     int64                   `json:"total"`
}

type WorkflowFilter struct {
	Page     int
	PageSize int
}

type WorkflowSearchResult struct {
	Workflows []model.Process `json:"workflows"`
	Total     int64           `json:"total"`
}
