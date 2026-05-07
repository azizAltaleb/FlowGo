package application

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"strings"
	"time"

	pb "github.com/azizAltaleb/goflow/backend/api/v1/go"
	"github.com/azizAltaleb/goflow/backend/libs/logger"
	"github.com/azizAltaleb/goflow/backend/services/sync-worker/internal/domain/model"
	"github.com/azizAltaleb/goflow/backend/services/sync-worker/internal/domain/repository"

	"google.golang.org/protobuf/proto"
)

type SyncService struct {
	repo        repository.SyncRepository
	indexPrefix string
	log         *logger.Logger
}

func NewSyncService(repo repository.SyncRepository, indexPrefix string) *SyncService {
	return &SyncService{
		repo:        repo,
		indexPrefix: indexPrefix,
		log:         logger.New("sync-worker"),
	}
}

func (s *SyncService) ProcessMessage(ctx context.Context, topic string, msg model.DebeziumMessage) error {
	start := time.Now()
	index := s.indexNameForTopic(topic)

	// Extract document ID for logging
	var docID string
	if msg.Op == "d" {
		if id, _, err := extractIDAndDoc(msg.Before); err == nil {
			docID = id
		}
	} else {
		if id, _, err := extractIDAndDoc(msg.After); err == nil {
			docID = id
		}
	}

	logFields := map[string]any{
		"topic":     topic,
		"index":     index,
		"operation": msg.Op,
		"doc_id":    docID,
	}

	s.log.Debug(ctx, "processing message", logFields)

	// 1. Standard Sync (Mirroring)
	switch msg.Op {
	case "c", "u", "r":
		if len(msg.After) == 0 || string(msg.After) == "null" {
			s.log.Debug(ctx, "skipping empty after payload", logFields)
			return nil
		}
		id, doc, err := extractIDAndDoc(msg.After)
		if err != nil {
			s.log.Error(ctx, "failed to extract id/doc", map[string]any{
				"error": err.Error(),
				"topic": topic,
			})
			return fmt.Errorf("failed to extract id/doc: %v", err)
		}

		if err := s.repo.Upsert(ctx, index, id, doc); err != nil {
			s.log.Error(ctx, "upsert failed", map[string]any{
				"error":  err.Error(),
				"index":  index,
				"doc_id": id,
			})
			return fmt.Errorf("upsert failed: %v", err)
		}

		s.log.Debug(ctx, "upsert completed", map[string]any{
			"index":       index,
			"doc_id":      id,
			"duration_ms": time.Since(start).Milliseconds(),
		})

	case "d":
		if len(msg.Before) == 0 || string(msg.Before) == "null" {
			s.log.Debug(ctx, "skipping empty before payload for delete", logFields)
			return nil
		}
		id, _, err := extractIDAndDoc(msg.Before)
		if err != nil {
			s.log.Error(ctx, "failed to extract id for delete", map[string]any{
				"error": err.Error(),
				"topic": topic,
			})
			return fmt.Errorf("failed to extract id for delete: %v", err)
		}

		s.log.Info(ctx, "deleting document", map[string]any{
			"index":  index,
			"doc_id": id,
		})

		if err := s.repo.Delete(ctx, index, id); err != nil {
			s.log.Error(ctx, "delete failed", map[string]any{
				"error":  err.Error(),
				"index":  index,
				"doc_id": id,
			})
			return fmt.Errorf("delete failed: %v", err)
		}

		s.log.Info(ctx, "delete completed", map[string]any{
			"index":       index,
			"doc_id":      id,
			"duration_ms": time.Since(start).Milliseconds(),
		})

	default:
		s.log.Debug(ctx, "ignoring unknown operation", logFields)
	}

	// 2. Denormalization (CQRS Enhancement)
	// Check if this is a Variable update
	if strings.HasSuffix(index, "variable") {
		if err := s.denormalizeVariable(ctx, msg); err != nil {
			s.log.Error(ctx, "failed to denormalize variable", map[string]any{
				"error": err.Error(),
				"index": index,
			})
			return fmt.Errorf("failed to denormalize variable: %v", err)
		}
	}

	return nil
}

func (s *SyncService) denormalizeVariable(ctx context.Context, msg model.DebeziumMessage) error {
	start := time.Now()
	isDelete := msg.Op == "d"
	var sourceMap map[string]any
	var err error

	if isDelete {
		if len(msg.Before) == 0 || string(msg.Before) == "null" {
			return nil
		}
		_, sourceMap, err = extractIDAndDoc(msg.Before)
	} else {
		// c, u, r
		if len(msg.After) == 0 || string(msg.After) == "null" {
			return nil
		}
		_, sourceMap, err = extractIDAndDoc(msg.After)
	}
	if err != nil {
		return err
	}

	// Extract Process Instance Key
	piKeyVal, ok := sourceMap["process_instance_key"]
	if !ok {
		s.log.Warn(ctx, "process_instance_key missing in variable, skipping denormalization", nil)
		return fmt.Errorf("process_instance_key missing in variable")
	}
	piKey := fmt.Sprint(piKeyVal)

	// Extract Variable Name
	nameVal, ok := sourceMap["name"]
	if !ok {
		s.log.Warn(ctx, "name missing in variable, skipping denormalization", nil)
		return fmt.Errorf("name missing in variable")
	}
	name := fmt.Sprint(nameVal)

	// Target Index: goflow-process_instance
	targetIndex := "goflow-process_instance"
	if s.indexPrefix != "" {
		targetIndex = s.indexPrefix + "-process_instance"
	}

	logFields := map[string]any{
		"process_instance_key": piKey,
		"variable_name":        name,
		"target_index":         targetIndex,
		"operation":            msg.Op,
	}

	if isDelete {
		s.log.Debug(ctx, "denormalizing variable delete", logFields)

		script := "if (ctx._source.context != null) { ctx._source.context.remove(params.name); }"
		params := map[string]any{"name": name}
		if err := s.repo.UpdateWithScript(ctx, targetIndex, piKey, script, params); err != nil {
			// 404 is expected if the process instance was already deleted
			if strings.Contains(err.Error(), "404") {
				s.log.Debug(ctx, "variable denormalization delete skipped - process instance already deleted", map[string]any{
					"process_instance_key": piKey,
					"variable_name":        name,
				})
				return nil
			}
			s.log.Error(ctx, "variable denormalization delete failed", map[string]any{
				"error":                err.Error(),
				"process_instance_key": piKey,
				"variable_name":        name,
			})
			return err
		}

		s.log.Info(ctx, "variable denormalization delete completed", map[string]any{
			"process_instance_key": piKey,
			"variable_name":        name,
			"duration_ms":          time.Since(start).Milliseconds(),
		})
		return nil
	}

	// Upsert
	val, ok := sourceMap["value"]
	if !ok {
		s.log.Debug(ctx, "variable value missing, skipping denormalization", logFields)
		return nil
	}

	// Try to unmarshal the value if it's a string containing JSON
	// The database stores values as JSON strings (e.g. "\"bar\"", "123", "{\"x\":1}")
	// We want to store the actual value in the ES context map.
	originalVal := val
	if strVal, ok := val.(string); ok {
		var unmarshaled any
		if err := json.Unmarshal([]byte(strVal), &unmarshaled); err == nil {
			val = unmarshaled
		}
	}

	s.log.Debug(ctx, "denormalizing variable upsert", map[string]any{
		"process_instance_key": piKey,
		"variable_name":        name,
		"original_value":       originalVal,
		"parsed_value":         val,
	})

	// Denormalize into 'context' field
	script := "if (ctx._source.context == null) { ctx._source.context = new HashMap(); } ctx._source.context[params.name] = params.value;"
	params := map[string]any{
		"name":  name,
		"value": val,
	}
	if err := s.repo.UpdateWithScript(ctx, targetIndex, piKey, script, params); err != nil {
		s.log.Error(ctx, "variable denormalization upsert failed", map[string]any{
			"error":                err.Error(),
			"process_instance_key": piKey,
			"variable_name":        name,
		})
		return err
	}

	s.log.Info(ctx, "variable denormalization upsert completed", map[string]any{
		"process_instance_key": piKey,
		"variable_name":        name,
		"duration_ms":          time.Since(start).Milliseconds(),
	})
	return nil
}

func (s *SyncService) indexNameForTopic(topic string) string {
	// Debezium topic format: <topic.prefix>.<schema>.<table>
	parts := strings.Split(topic, ".")
	table := topic
	if len(parts) >= 3 {
		table = parts[len(parts)-1]
	}

	if s.indexPrefix == "" {
		return table
	}
	return s.indexPrefix + "-" + table
}

func extractIDAndDoc(raw json.RawMessage) (string, map[string]any, error) {
	var doc map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&doc); err != nil {
		return "", nil, err
	}

	// Convention: most tables have PK column "key". Variable uses "scope_key".
	if v, ok := doc["key"]; ok {
		return fmt.Sprint(v), doc, nil
	}
	if v, ok := doc["scope_key"]; ok {
		return fmt.Sprint(v), doc, nil
	}

	return "", doc, fmt.Errorf("unable to determine document id (expected key or scope_key)")
}

func (s *SyncService) ProcessEvent(ctx context.Context, eventType string, data []byte) error {
	start := time.Now()

	switch eventType {
	case "ProcessInstanceCreated":
		var event pb.ProcessInstanceCreated
		if err := proto.Unmarshal(data, &event); err != nil {
			return fmt.Errorf("failed to unmarshal ProcessInstanceCreated: %w", err)
		}

		doc := map[string]any{
			"key":                    event.Key,
			"id":                     event.Id,
			"process_definition_key": event.ProcessDefinitionKey,
			"version":                event.Version,
			"bpmn_process_id":        event.BpmnProcessId,
			"state":                  "ACTIVE",
			"created_at":             event.CreatedAt.AsTime(),
		}

		index := s.indexNameForTopic("process_instance")
		if err := s.repo.Upsert(ctx, index, fmt.Sprintf("%d", event.Key), doc); err != nil {
			return err
		}

	case "ProcessInstanceCompleted":
		var event pb.ProcessInstanceCompleted
		if err := proto.Unmarshal(data, &event); err != nil {
			return fmt.Errorf("failed to unmarshal ProcessInstanceCompleted: %w", err)
		}

		// Partial update? Repo.Upsert overwrites.
		// Use UpdateWithScript or Repo.Update?
		// Repo interface: Upsert(ctx, index, id, doc) - implies overwrite or merge?
		// ES Upsert usually merges if using "doc_as_upsert"?
		// Current Upsert implementation in es_repository.go calls `client.Index`. This OVERWRITES.
		// I need a partial update method or read-modify-write.
		// Or assume Debezium/Events provide FULL state?
		// Events only provide partial change.
		// I should use `Update` (which uses `client.Update`).
		// I'll check `es_repository.go` later, assume `Update` exists or `Upsert` handles it?
		// Wait, `process_instance` created has full state. Completed has only end time.
		// If I overwrite with only end time, I lose other fields!
		// I need partial update.
		// `ProcessInstanceCompleted` event should ideally contain full state or I use `repo.Update`.
		// Repo has `UpdateWithScript`. Does it have `Update` (partial)?

		index := s.indexNameForTopic("process_instance")
		// Use script update for simple field updates
		script := "ctx._source.state = params.state; ctx._source.end_time = params.end_time;"
		params := map[string]any{
			"state":    "COMPLETED",
			"end_time": event.EndTime.AsTime(),
		}
		if err := s.repo.UpdateWithScript(ctx, index, fmt.Sprintf("%d", event.Key), script, params); err != nil {
			return err
		}

	case "VariableUpdated":
		var event pb.VariableUpdated
		if err := proto.Unmarshal(data, &event); err != nil {
			return fmt.Errorf("failed to unmarshal VariableUpdated: %w", err)
		}

		// Generate Deterministic Key
		keyStr := fmt.Sprintf("%d_%s", event.ProcessInstanceKey, event.Name)
		key := generateDeterministicKey(keyStr)

		// 1. Index Variable Document
		varDoc := map[string]any{
			"key":                  key,
			"name":                 event.Name,
			"value":                event.Value,
			"process_instance_key": event.ProcessInstanceKey,
			"scope_key":            event.ScopeKey,
			"updated_at":           event.UpdatedAt.AsTime(),
		}
		indexVar := s.indexNameForTopic("variable")
		if err := s.repo.Upsert(ctx, indexVar, fmt.Sprintf("%d", key), varDoc); err != nil {
			return err
		}

		// 2. Denormalize to Process Instance
		// Parse value JSON?
		var val any
		json.Unmarshal([]byte(event.Value), &val)

		indexPi := s.indexNameForTopic("process_instance")
		script := "if (ctx._source.context == null) { ctx._source.context = new HashMap(); } ctx._source.context[params.name] = params.value;"
		params := map[string]any{
			"name":  event.Name,
			"value": val,
		}
		if err := s.repo.UpdateWithScript(ctx, indexPi, fmt.Sprintf("%d", event.ProcessInstanceKey), script, params); err != nil {
			return err
		}

	case "JobCreated":
		var event pb.JobCreated
		if err := proto.Unmarshal(data, &event); err != nil {
			return fmt.Errorf("failed to unmarshal JobCreated: %w", err)
		}
		doc := map[string]any{
			"key":                    event.Key,
			"id":                     event.Id,
			"type":                   event.Type,
			"process_instance_key":   event.ProcessInstanceKey,
			"element_instance_key":   event.ElementInstanceKey,
			"process_definition_key": event.ProcessDefinitionKey,
			"element_id":             event.ElementId,
			"state":                  "CREATED",
			"created_at":             event.CreatedAt.AsTime(),
		}
		index := s.indexNameForTopic("job")
		if err := s.repo.Upsert(ctx, index, fmt.Sprintf("%d", event.Key), doc); err != nil {
			return err
		}

	case "JobActivated":
		var event pb.JobActivated
		if err := proto.Unmarshal(data, &event); err != nil {
			return fmt.Errorf("failed to unmarshal JobActivated: %w", err)
		}
		index := s.indexNameForTopic("job")
		script := "ctx._source.state = params.state; ctx._source.worker = params.worker; ctx._source.updated_at = params.updated_at;"
		params := map[string]any{
			"state":      "ACTIVATED",
			"worker":     event.Worker,
			"updated_at": time.Now(),
		}
		if err := s.repo.UpdateWithScript(ctx, index, fmt.Sprintf("%d", event.Key), script, params); err != nil {
			return err
		}

	case "JobCompleted":
		var event pb.JobCompleted
		if err := proto.Unmarshal(data, &event); err != nil {
			return fmt.Errorf("failed to unmarshal JobCompleted: %w", err)
		}
		index := s.indexNameForTopic("job")
		script := "ctx._source.state = params.state; ctx._source.worker = params.worker; ctx._source.updated_at = params.updated_at;"
		params := map[string]any{
			"state":      "COMPLETED",
			"worker":     event.Worker,
			"updated_at": event.UpdatedAt.AsTime(),
		}
		if err := s.repo.UpdateWithScript(ctx, index, fmt.Sprintf("%d", event.Key), script, params); err != nil {
			return err
		}

	case "JobFailed":
		var event pb.JobFailed
		if err := proto.Unmarshal(data, &event); err != nil {
			return fmt.Errorf("failed to unmarshal JobFailed: %w", err)
		}
		index := s.indexNameForTopic("job")
		script := "ctx._source.state = params.state; ctx._source.retries = params.retries; ctx._source.error_message = params.error_message; ctx._source.updated_at = params.updated_at;"
		params := map[string]any{
			"state":         "FAILED",
			"retries":       event.Retries,
			"error_message": event.ErrorMessage,
			"updated_at":    event.UpdatedAt.AsTime(),
		}
		if err := s.repo.UpdateWithScript(ctx, index, fmt.Sprintf("%d", event.Key), script, params); err != nil {
			return err
		}

	case "ElementInstanceActivated":
		var event pb.ElementInstanceActivated
		if err := proto.Unmarshal(data, &event); err != nil {
			return fmt.Errorf("failed to unmarshal ElementInstanceActivated: %w", err)
		}
		doc := map[string]any{
			"key":                    event.Key,
			"id":                     event.Id,
			"process_instance_key":   event.ProcessInstanceKey,
			"process_definition_key": event.ProcessDefinitionKey,
			"element_id":             event.ElementId,
			"bpmn_element_type":      event.BpmnElementType,
			"flow_scope_key":         event.FlowScopeKey,
			"state":                  "ACTIVE",
			"created_at":             event.CreatedAt.AsTime(),
		}
		index := s.indexNameForTopic("element_instance")
		if err := s.repo.Upsert(ctx, index, fmt.Sprintf("%d", event.Key), doc); err != nil {
			return err
		}

	case "ElementInstanceCompleted":
		var event pb.ElementInstanceCompleted
		if err := proto.Unmarshal(data, &event); err != nil {
			return fmt.Errorf("failed to unmarshal ElementInstanceCompleted: %w", err)
		}
		index := s.indexNameForTopic("element_instance")
		script := "ctx._source.state = params.state; ctx._source.end_time = params.end_time;"
		params := map[string]any{
			"state":    "COMPLETED",
			"end_time": event.EndTime.AsTime(),
		}
		if err := s.repo.UpdateWithScript(ctx, index, fmt.Sprintf("%d", event.Key), script, params); err != nil {
			return err
		}
	}

	s.log.Info(ctx, "processed event", map[string]any{
		"type":        eventType,
		"duration_ms": time.Since(start).Milliseconds(),
	})

	return nil
}

// generateDeterministicKey uses FNV to generate keys for variables/deduplication
func generateDeterministicKey(s string) int64 {
	h := fnv.New64a()
	h.Write([]byte(s))
	// Ensure positive key
	return int64(h.Sum64() & 0x7FFFFFFFFFFFFFFF)
}
