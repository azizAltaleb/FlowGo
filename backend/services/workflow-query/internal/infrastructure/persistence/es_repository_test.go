package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/artificialflow/artificialflow/backend/libs/model"
	queryrepository "github.com/artificialflow/artificialflow/backend/services/workflow-query/internal/domain/repository"
)

type searchCall struct {
	index string
	body  json.RawMessage
}

type fakeSearchBackend struct {
	responses map[string]json.RawMessage
	errors    map[string]error
	calls     []searchCall
}

func (f *fakeSearchBackend) Search(_ context.Context, index string, body json.RawMessage) (json.RawMessage, error) {
	f.calls = append(f.calls, searchCall{index: index, body: append(json.RawMessage(nil), body...)})
	if err := f.errors[index]; err != nil {
		return nil, err
	}
	return f.responses[index], nil
}

func (f *fakeSearchBackend) Upsert(context.Context, string, string, any) error {
	return nil
}

func (f *fakeSearchBackend) Delete(context.Context, string, string) error {
	return nil
}

func TestNewESRepositoryDefaultsToCanonicalPrefix(t *testing.T) {
	repo := NewESRepository(&fakeSearchBackend{}, "")

	if repo.instanceIndex != "artificialflow-process_instance" || repo.processIndex != "artificialflow-process" {
		t.Fatalf("expected canonical indexes, got instance=%q process=%q", repo.instanceIndex, repo.processIndex)
	}
	if repo.legacyInstanceIndex != "flowgo-process_instance" || repo.legacyProcessIndex != "flowgo-process" {
		t.Fatalf("expected legacy fallbacks, got instance=%q process=%q", repo.legacyInstanceIndex, repo.legacyProcessIndex)
	}
}

func TestESRepositorySearchInstancesCanonicalOnly(t *testing.T) {
	backend := &fakeSearchBackend{
		responses: map[string]json.RawMessage{
			"artificialflow-process_instance": searchResponse(
				map[string]any{"key": 2, "state": "ACTIVE", "created_at": "2026-01-02T00:00:00Z"},
				map[string]any{"key": 1, "state": "COMPLETED", "created_at": "2026-01-01T00:00:00Z"},
			),
		},
		errors: map[string]error{
			"flowgo-process_instance": errors.New(`status=404 type=index_not_found_exception`),
		},
	}
	repo := NewESRepository(backend, "")

	result, err := repo.SearchInstances(context.Background(), queryrepository.InstanceFilter{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("canonical-only search failed: %v", err)
	}
	if result.Total != 2 || keysOfInstances(result.Instances) != "2,1" {
		t.Fatalf("unexpected canonical-only result: %#v", result)
	}
	assertSearchOrder(t, backend.calls, "artificialflow-process_instance", "flowgo-process_instance")

	var body map[string]any
	if err := json.Unmarshal(backend.calls[0].body, &body); err != nil {
		t.Fatalf("decode search body: %v", err)
	}
	if body["from"] != float64(0) || body["size"] != float64(transitionMergeMaxDocuments) || body["track_total_hits"] != true {
		t.Fatalf("search must request the complete bounded result set: %#v", body)
	}
}

func TestESRepositorySearchWorkflowsLegacyOnly(t *testing.T) {
	backend := &fakeSearchBackend{
		responses: map[string]json.RawMessage{
			"flowgo-process": searchResponse(
				map[string]any{"key": 7, "bpmn_process_id": "legacy", "created_at": "2026-01-07T00:00:00Z"},
			),
		},
		errors: map[string]error{
			"artificialflow-process": errors.New(`status=404 type=index_not_found_exception`),
		},
	}
	repo := NewESRepository(backend, "")

	result, err := repo.SearchWorkflows(context.Background(), queryrepository.WorkflowFilter{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("legacy-only search failed: %v", err)
	}
	if result.Total != 1 || len(result.Workflows) != 1 || result.Workflows[0].BpmnProcessID != "legacy" {
		t.Fatalf("unexpected legacy-only result: %#v", result)
	}
	assertSearchOrder(t, backend.calls, "artificialflow-process", "flowgo-process")
}

func TestESRepositorySearchInstancesMergesWithCanonicalPrecedenceAndPagination(t *testing.T) {
	backend := &fakeSearchBackend{
		responses: map[string]json.RawMessage{
			"artificialflow-process_instance": searchResponse(
				map[string]any{"key": 4, "state": "CANONICAL-FOUR", "created_at": "2026-01-04T00:00:00Z"},
				map[string]any{"key": 2, "state": "CANONICAL-WINS", "created_at": "2026-01-02T00:00:00Z"},
			),
			"flowgo-process_instance": searchResponse(
				map[string]any{"key": 3, "state": "LEGACY-THREE", "created_at": "2026-01-03T00:00:00Z"},
				map[string]any{"key": 2, "state": "LEGACY-LOSES", "created_at": "2026-01-02T00:00:00Z"},
				map[string]any{"key": 1, "state": "LEGACY-ONE", "created_at": "2026-01-01T00:00:00Z"},
			),
		},
		errors: map[string]error{},
	}
	repo := NewESRepository(backend, "")

	result, err := repo.SearchInstances(context.Background(), queryrepository.InstanceFilter{Page: 2, PageSize: 2})
	if err != nil {
		t.Fatalf("mixed search failed: %v", err)
	}
	if result.Total != 4 || keysOfInstances(result.Instances) != "2,1" {
		t.Fatalf("unexpected merged page: %#v", result)
	}
	if result.Instances[0].State != "CANONICAL-WINS" {
		t.Fatalf("canonical duplicate did not take precedence: %#v", result.Instances[0])
	}
}

func TestESRepositorySearchWorkflowsMergesCanonicalAndLegacy(t *testing.T) {
	backend := &fakeSearchBackend{
		responses: map[string]json.RawMessage{
			"artificialflow-process": searchResponse(
				map[string]any{"key": 2, "bpmn_process_id": "canonical", "created_at": "2026-01-02T00:00:00Z"},
			),
			"flowgo-process": searchResponse(
				map[string]any{"key": 2, "bpmn_process_id": "legacy-duplicate", "created_at": "2026-01-02T00:00:00Z"},
				map[string]any{"key": 1, "bpmn_process_id": "legacy", "created_at": "2026-01-01T00:00:00Z"},
			),
		},
		errors: map[string]error{},
	}
	repo := NewESRepository(backend, "")

	result, err := repo.SearchWorkflows(context.Background(), queryrepository.WorkflowFilter{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("mixed workflow search failed: %v", err)
	}
	if result.Total != 2 || len(result.Workflows) != 2 ||
		result.Workflows[0].BpmnProcessID != "canonical" ||
		result.Workflows[1].BpmnProcessID != "legacy" {
		t.Fatalf("unexpected mixed workflow result: %#v", result)
	}
}

func TestESRepositoryGetInstancePrefersCanonicalDocument(t *testing.T) {
	backend := &fakeSearchBackend{
		responses: map[string]json.RawMessage{
			"artificialflow-process_instance": searchResponse(
				map[string]any{"key": 42, "process_definition_key": 7, "state": "CANONICAL"},
			),
		},
		errors: map[string]error{
			"flowgo-process_instance": errors.New("status=503 should not be queried"),
		},
	}
	repo := NewESRepository(backend, "")

	instance, err := repo.GetInstance(context.Background(), "42")
	if err != nil {
		t.Fatalf("canonical get failed: %v", err)
	}
	if instance.State != "CANONICAL" || len(backend.calls) != 1 {
		t.Fatalf("canonical lookup was not preferred: instance=%#v calls=%#v", instance, backend.calls)
	}
}

func TestESRepositoryGetInstanceFallsBackForEmptyCanonicalResult(t *testing.T) {
	backend := &fakeSearchBackend{
		responses: map[string]json.RawMessage{
			"artificialflow-process_instance": searchResponse(),
			"flowgo-process_instance": searchResponse(
				map[string]any{"key": 42, "process_definition_key": 7, "state": "LEGACY"},
			),
		},
		errors: map[string]error{},
	}
	repo := NewESRepository(backend, "")

	instance, err := repo.GetInstance(context.Background(), "42")
	if err != nil {
		t.Fatalf("legacy get fallback failed: %v", err)
	}
	if instance.State != "LEGACY" {
		t.Fatalf("unexpected legacy instance: %#v", instance)
	}
	assertSearchOrder(t, backend.calls, "artificialflow-process_instance", "flowgo-process_instance")
}

func TestESRepositoryTreatsMissingIndicesAsEmpty(t *testing.T) {
	missing := errors.New(`search failed: status=404 type=index_not_found_exception`)
	backend := &fakeSearchBackend{
		responses: map[string]json.RawMessage{},
		errors: map[string]error{
			"artificialflow-process_instance": missing,
			"flowgo-process_instance":         missing,
			"artificialflow-process":          missing,
			"flowgo-process":                  missing,
		},
	}
	repo := NewESRepository(backend, "")

	instances, err := repo.SearchInstances(context.Background(), queryrepository.InstanceFilter{Page: 1, PageSize: 10})
	if err != nil || instances.Total != 0 || len(instances.Instances) != 0 {
		t.Fatalf("missing instance indices should be empty, got result=%#v err=%v", instances, err)
	}
	workflows, err := repo.SearchWorkflows(context.Background(), queryrepository.WorkflowFilter{Page: 1, PageSize: 10})
	if err != nil || workflows.Total != 0 || len(workflows.Workflows) != 0 {
		t.Fatalf("missing workflow indices should be empty, got result=%#v err=%v", workflows, err)
	}
	if _, err := repo.GetInstance(context.Background(), "42"); !errors.Is(err, queryrepository.ErrInstanceNotFound) {
		t.Fatalf("missing instance indices should return not found, got %v", err)
	}
}

func TestESRepositoryDoesNotFallbackOrMergeOnArbitraryErrors(t *testing.T) {
	t.Run("canonical error", func(t *testing.T) {
		backend := &fakeSearchBackend{
			responses: map[string]json.RawMessage{
				"flowgo-process_instance": searchResponse(),
			},
			errors: map[string]error{
				"artificialflow-process_instance": errors.New("status=503 cluster unavailable"),
			},
		}
		repo := NewESRepository(backend, "")

		if _, err := repo.SearchInstances(context.Background(), queryrepository.InstanceFilter{Page: 1, PageSize: 10}); err == nil {
			t.Fatal("expected canonical search error")
		}
		assertSearchOrder(t, backend.calls, "artificialflow-process_instance")
	})

	t.Run("legacy error", func(t *testing.T) {
		backend := &fakeSearchBackend{
			responses: map[string]json.RawMessage{
				"artificialflow-process": searchResponse(),
			},
			errors: map[string]error{
				"flowgo-process": errors.New("status=503 cluster unavailable"),
			},
		}
		repo := NewESRepository(backend, "")

		if _, err := repo.SearchWorkflows(context.Background(), queryrepository.WorkflowFilter{Page: 1, PageSize: 10}); err == nil {
			t.Fatal("expected legacy search error")
		}
		assertSearchOrder(t, backend.calls, "artificialflow-process", "flowgo-process")
	})
}

func TestESRepositoryValidatesTransitionPaginationAndCompleteness(t *testing.T) {
	repo := NewESRepository(&fakeSearchBackend{}, "")
	if _, err := repo.SearchInstances(context.Background(), queryrepository.InstanceFilter{
		Page:     2,
		PageSize: transitionMergeMaxDocuments,
	}); err == nil || !strings.Contains(err.Error(), "first 10000") {
		t.Fatalf("expected transition page-window error, got %v", err)
	}

	backend := &fakeSearchBackend{
		responses: map[string]json.RawMessage{
			"artificialflow-process": json.RawMessage(`{"hits":{"total":{"value":10001,"relation":"eq"},"hits":[]}}`),
		},
		errors: map[string]error{
			"flowgo-process": errors.New(`status=404 type=index_not_found_exception`),
		},
	}
	repo = NewESRepository(backend, "")
	if _, err := repo.SearchWorkflows(context.Background(), queryrepository.WorkflowFilter{Page: 1, PageSize: 10}); err == nil ||
		!strings.Contains(err.Error(), "maximum is 10000") {
		t.Fatalf("expected per-index merge limit error, got %v", err)
	}
}

func TestESRepositoryExplicitLegacyOverrideDoesNotQueryCanonical(t *testing.T) {
	backend := &fakeSearchBackend{
		responses: map[string]json.RawMessage{
			"flowgo-process_instance": searchResponse(),
		},
		errors: map[string]error{},
	}
	repo := NewESRepository(backend, "flowgo")

	if _, err := repo.SearchInstances(context.Background(), queryrepository.InstanceFilter{Page: 1, PageSize: 10}); err != nil {
		t.Fatalf("explicit legacy override failed: %v", err)
	}
	assertSearchOrder(t, backend.calls, "flowgo-process_instance")
}

func searchResponse(documents ...map[string]any) json.RawMessage {
	hits := make([]map[string]any, 0, len(documents))
	for _, document := range documents {
		hits = append(hits, map[string]any{"_source": document})
	}
	response, err := json.Marshal(map[string]any{
		"hits": map[string]any{
			"total": map[string]any{"value": len(hits), "relation": "eq"},
			"hits":  hits,
		},
	})
	if err != nil {
		panic(err)
	}
	return response
}

func keysOfInstances(instances []model.ProcessInstance) string {
	keys := make([]string, 0, len(instances))
	for _, instance := range instances {
		keys = append(keys, strconv.FormatInt(instance.Key, 10))
	}
	return strings.Join(keys, ",")
}

func assertSearchOrder(t *testing.T, calls []searchCall, want ...string) {
	t.Helper()
	got := make([]string, 0, len(calls))
	for _, call := range calls {
		got = append(got, call.index)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected search order: got %v want %v", got, want)
	}
}
