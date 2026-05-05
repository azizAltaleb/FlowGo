package messaging

import (
	"context"
	"fmt"
	"time"
	workflowapi "workflow-engine/backend/api/v1/go"
	"workflow-engine/backend/libs/id"

	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"
)

type EventPublisher interface {
	Publish(ctx context.Context, event proto.Message, eventType string) error
	Close() error
}

type KafkaPublisher struct {
	writer *kafka.Writer
}

func NewKafkaPublisher(brokers []string, topic string) *KafkaPublisher {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic, // Single topic for all events? Or one topic? "workflow.events.v1"
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 10 * time.Millisecond,
	}
	return &KafkaPublisher{writer: writer}
}

func (p *KafkaPublisher) Publish(ctx context.Context, event proto.Message, eventType string) error {
	data, err := proto.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	aggregateID := aggregateIDForEvent(event)
	if aggregateID == "" {
		aggregateID = "event:" + eventType
	}
	eventID := id.GenerateUUIDv7()
	occurredAt := time.Now().UTC().Format(time.RFC3339Nano)

	msg := kafka.Message{
		Key:   []byte(aggregateID),
		Value: data,
		Headers: []kafka.Header{
			{Key: "type", Value: []byte(eventType)},
			{Key: "event_id", Value: []byte(eventID)},
			{Key: "aggregate_id", Value: []byte(aggregateID)},
			{Key: "occurred_at", Value: []byte(occurredAt)},
			{Key: "schema_version", Value: []byte("1")},
			{Key: "source", Value: []byte("workflow-command")},
		},
	}

	return p.writer.WriteMessages(ctx, msg)
}

func aggregateIDForEvent(event proto.Message) string {
	switch e := event.(type) {
	case *workflowapi.ProcessInstanceCreated:
		return fmt.Sprintf("process-instance:%d", e.Key)
	case *workflowapi.ProcessInstanceCompleted:
		return fmt.Sprintf("process-instance:%d", e.Key)
	case *workflowapi.ProcessInstanceTerminated:
		return fmt.Sprintf("process-instance:%d", e.Key)
	case *workflowapi.ElementInstanceActivated:
		if e.ProcessInstanceKey > 0 {
			return fmt.Sprintf("process-instance:%d", e.ProcessInstanceKey)
		}
		return fmt.Sprintf("element-instance:%d", e.Key)
	case *workflowapi.ElementInstanceCompleted:
		if e.ProcessInstanceKey > 0 {
			return fmt.Sprintf("process-instance:%d", e.ProcessInstanceKey)
		}
		return fmt.Sprintf("element-instance:%d", e.Key)
	case *workflowapi.ElementInstanceTerminated:
		if e.ProcessInstanceKey > 0 {
			return fmt.Sprintf("process-instance:%d", e.ProcessInstanceKey)
		}
		return fmt.Sprintf("element-instance:%d", e.Key)
	case *workflowapi.JobCreated:
		if e.ProcessInstanceKey > 0 {
			return fmt.Sprintf("process-instance:%d", e.ProcessInstanceKey)
		}
		return fmt.Sprintf("job:%d", e.Key)
	case *workflowapi.JobActivated:
		return fmt.Sprintf("job:%d", e.Key)
	case *workflowapi.JobCompleted:
		return fmt.Sprintf("job:%d", e.Key)
	case *workflowapi.JobFailed:
		return fmt.Sprintf("job:%d", e.Key)
	case *workflowapi.VariableUpdated:
		if e.ProcessInstanceKey > 0 {
			return fmt.Sprintf("process-instance:%d", e.ProcessInstanceKey)
		}
		return fmt.Sprintf("scope:%d", e.ScopeKey)
	case *workflowapi.IncidentCreated:
		if e.ProcessInstanceKey > 0 {
			return fmt.Sprintf("process-instance:%d", e.ProcessInstanceKey)
		}
		return fmt.Sprintf("incident:%d", e.Key)
	case *workflowapi.IncidentResolved:
		if e.ProcessInstanceKey > 0 {
			return fmt.Sprintf("process-instance:%d", e.ProcessInstanceKey)
		}
		return fmt.Sprintf("incident:%d", e.Key)
	default:
		return ""
	}
}

func (p *KafkaPublisher) Close() error {
	return p.writer.Close()
}

// NoOpPublisher for tests
type NoOpPublisher struct{}

func (p *NoOpPublisher) Publish(ctx context.Context, event proto.Message, eventType string) error {
	return nil
}

func (p *NoOpPublisher) Close() error {
	return nil
}
