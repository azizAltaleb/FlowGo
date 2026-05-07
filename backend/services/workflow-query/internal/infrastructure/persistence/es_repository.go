package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/azizAltaleb/goflow/backend/libs/model"
	"github.com/azizAltaleb/goflow/backend/libs/search"
	"github.com/azizAltaleb/goflow/backend/services/workflow-query/internal/domain/repository"
	"strconv"
	"strings"
	"time"
)

type ESRepository struct {
	client        search.Backend
	instanceIndex string
	processIndex  string
}

// Ensure implementation
var _ repository.QueryRepository = &ESRepository{}

func NewESRepository(client search.Backend, indexPrefix string) *ESRepository {
	instanceIndex := "goflow-process_instance"
	processIndex := "goflow-process"
	if indexPrefix != "" {
		instanceIndex = indexPrefix + "-process_instance"
		processIndex = indexPrefix + "-process"
	}
	return &ESRepository{
		client:        client,
		instanceIndex: instanceIndex,
		processIndex:  processIndex,
	}
}

func (r *ESRepository) GetInstance(ctx context.Context, id string) (*model.ProcessInstance, error) {
	instanceKey, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid instance id: %w", err)
	}

	// Search by exact key match
	queryMap := map[string]any{
		"query": map[string]any{
			"term": map[string]any{
				"key": instanceKey,
			},
		},
	}

	bodyBytes, err := json.Marshal(queryMap)
	if err != nil {
		return nil, err
	}

	resBytes, err := r.client.Search(ctx, r.instanceIndex, bodyBytes)
	if err != nil {
		if strings.Contains(err.Error(), "status=404") {
			return nil, repository.ErrInstanceNotFound
		}
		return nil, err
	}

	var esResp struct {
		Hits struct {
			Hits []struct {
				Source esProcessInstance `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.Unmarshal(resBytes, &esResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ES response: %v", err)
	}

	if len(esResp.Hits.Hits) == 0 {
		return nil, fmt.Errorf("instance not found")
	}

	src := esResp.Hits.Hits[0].Source
	return &model.ProcessInstance{
		Key:                      src.Key,
		ProcessDefinitionKey:     src.ProcessDefinitionKey,
		Version:                  src.Version,
		ParentProcessInstanceKey: src.ParentProcessInstanceKey,
		ParentElementInstanceKey: src.ParentElementInstanceKey,
		State:                    src.State,
		CreatedAt:                src.CreatedAt,
		EndTime:                  src.EndTime,
		Context:                  src.Context,
	}, nil
}

func (r *ESRepository) SearchInstances(ctx context.Context, filter repository.InstanceFilter) (*repository.InstanceSearchResult, error) {
	filterClauses := make([]map[string]any, 0, 2)
	if filter.WorkflowID != "" {
		workflowKey, err := strconv.ParseInt(filter.WorkflowID, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid workflow id filter %q: %w", filter.WorkflowID, err)
		}
		filterClauses = append(filterClauses, map[string]any{
			"term": map[string]any{
				"process_definition_key": workflowKey,
			},
		})
	}
	if filter.State != "" {
		filterClauses = append(filterClauses, map[string]any{
			"term": map[string]any{
				"state": filter.State,
			},
		})
	}

	queryMap := map[string]any{
		"bool": map[string]any{
			"filter": filterClauses,
		},
	}

	from := (filter.Page - 1) * filter.PageSize
	if from < 0 {
		from = 0
	}

	searchBody := map[string]any{
		"query": queryMap,
		"from":  from,
		"size":  filter.PageSize,
		"sort": []map[string]any{
			{"created_at": "desc"},
		},
	}

	bodyBytes, err := json.Marshal(searchBody)
	if err != nil {
		return nil, err
	}

	resBytes, err := r.client.Search(ctx, r.instanceIndex, bodyBytes)
	if err != nil {
		if strings.Contains(err.Error(), "status=404") {
			return &repository.InstanceSearchResult{
				Instances: make([]model.ProcessInstance, 0),
				Total:     0,
			}, nil
		}
		return nil, err
	}

	var esResp struct {
		Hits struct {
			Total struct {
				Value int64 `json:"value"`
			} `json:"total"`
			Hits []struct {
				Source esProcessInstance `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.Unmarshal(resBytes, &esResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ES response: %v", err)
	}

	result := &repository.InstanceSearchResult{
		Instances: make([]model.ProcessInstance, 0),
		Total:     esResp.Hits.Total.Value,
	}

	for _, hit := range esResp.Hits.Hits {
		src := hit.Source
		pi := model.ProcessInstance{
			Key:                      src.Key,
			ProcessDefinitionKey:     src.ProcessDefinitionKey,
			Version:                  src.Version,
			ParentProcessInstanceKey: src.ParentProcessInstanceKey,
			ParentElementInstanceKey: src.ParentElementInstanceKey,
			State:                    src.State,
			CreatedAt:                src.CreatedAt,
			EndTime:                  src.EndTime,
			Context:                  src.Context,
		}
		result.Instances = append(result.Instances, pi)
	}

	return result, nil
}

func (r *ESRepository) SearchWorkflows(ctx context.Context, filter repository.WorkflowFilter) (*repository.WorkflowSearchResult, error) {
	from := (filter.Page - 1) * filter.PageSize
	if from < 0 {
		from = 0
	}

	searchBody := map[string]any{
		"query": map[string]any{
			"match_all": map[string]any{},
		},
		"from": from,
		"size": filter.PageSize,
		"sort": []map[string]any{
			{"created_at": "desc"},
		},
	}

	bodyBytes, err := json.Marshal(searchBody)
	if err != nil {
		return nil, err
	}

	resBytes, err := r.client.Search(ctx, r.processIndex, bodyBytes)
	if err != nil {
		if strings.Contains(err.Error(), "status=404") {
			return &repository.WorkflowSearchResult{
				Workflows: make([]model.Process, 0),
				Total:     0,
			}, nil
		}
		return nil, err
	}

	var esResp struct {
		Hits struct {
			Total struct {
				Value int64 `json:"value"`
			} `json:"total"`
			Hits []struct {
				Source esProcess `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.Unmarshal(resBytes, &esResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ES response: %v", err)
	}

	result := &repository.WorkflowSearchResult{
		Workflows: make([]model.Process, 0),
		Total:     esResp.Hits.Total.Value,
	}

	for _, hit := range esResp.Hits.Hits {
		src := hit.Source
		p := model.Process{
			Key:              src.Key,
			BpmnProcessID:    src.BpmnProcessID,
			Version:          src.Version,
			ResourceName:     src.ResourceName,
			DeploymentKey:    src.DeploymentKey,
			Resource:         src.Resource,
			ResourceChecksum: src.ResourceChecksum,
			TenantID:         src.TenantID,
			CreatedAt:        src.CreatedAt,
		}
		result.Workflows = append(result.Workflows, p)
	}

	return result, nil
}

// esProcessInstance matches the Elasticsearch document structure (snake_case)
type esProcessInstance struct {
	Key                      int64          `json:"key"`
	ProcessDefinitionKey     int64          `json:"process_definition_key"`
	Version                  int            `json:"version"`
	ParentProcessInstanceKey int64          `json:"parent_process_instance_key"`
	ParentElementInstanceKey int64          `json:"parent_element_instance_key"`
	State                    string         `json:"state"`
	CreatedAt                time.Time      `json:"created_at"`
	EndTime                  time.Time      `json:"end_time"`
	Context                  map[string]any `json:"context"`
}

// esProcess matches the Elasticsearch document structure for processes (snake_case)
type esProcess struct {
	Key              int64     `json:"key"`
	BpmnProcessID    string    `json:"bpmn_process_id"`
	Version          int       `json:"version"`
	ResourceName     string    `json:"resource_name"`
	DeploymentKey    int64     `json:"deployment_key"`
	Resource         []byte    `json:"resource"`
	ResourceChecksum string    `json:"resource_checksum"`
	TenantID         string    `json:"tenant_id"`
	CreatedAt        time.Time `json:"created_at"`
}
