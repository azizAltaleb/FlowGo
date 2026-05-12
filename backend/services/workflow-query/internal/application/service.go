package application

import (
	"context"
	"github.com/azizAltaleb/flowgo/backend/libs/model"
	"github.com/azizAltaleb/flowgo/backend/services/workflow-query/internal/domain/repository"
)

type QueryService struct {
	repo repository.QueryRepository
}

func NewQueryService(repo repository.QueryRepository) *QueryService {
	return &QueryService{repo: repo}
}

func (s *QueryService) SearchInstances(ctx context.Context, filter repository.InstanceFilter) (*repository.InstanceSearchResult, error) {
	return s.repo.SearchInstances(ctx, filter)
}

func (s *QueryService) SearchWorkflows(ctx context.Context, filter repository.WorkflowFilter) (*repository.WorkflowSearchResult, error) {
	return s.repo.SearchWorkflows(ctx, filter)
}

func (s *QueryService) GetInstance(ctx context.Context, id string) (*model.ProcessInstance, error) {
	return s.repo.GetInstance(ctx, id)
}
