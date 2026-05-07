package messaging

import (
	"context"
	"fmt"
	"github.com/azizAltaleb/goflow/backend/libs/id"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"google.golang.org/protobuf/proto"
)

type NatsPublisher struct {
	nc *nats.Conn
	js jetstream.JetStream
}

func NewNatsPublisher(url string) (*NatsPublisher, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to nats: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create jetstream context: %w", err)
	}

	return &NatsPublisher{
		nc: nc,
		js: js,
	}, nil
}

func (p *NatsPublisher) Publish(ctx context.Context, event proto.Message, eventType string) error {
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

	// Subject format: workflow.events.<eventType>
	// or just workflow.events.v1 for single stream?
	// Kafka used "workflow.events.v1" as TOPIC.
	// NATS Stream can be WORKFLOW_EVENTS.
	// Subjects can vary.
	// Let's use a consistent subject prefix provided by config or specific to event type?
	// Migration guide says: "Update Configuration for Event Bus Selection".
	// Let's assume we map Kafka Topic "workflow.events.v1" to Nats Subject "workflow.events.v1".
	// But NATS allows granular subjects.
	// To keep it compatible and simple for now: "workflow.events.v1" matching Kafka.
	// But usually in NATS we'd do "workflow.events.v1.<eventType>" for filtering.

	// For now, let's publish to "workflow.events.v1".
	// The Consumer will filter by header "type".

	msg := &nats.Msg{
		Subject: "workflow.events.v1",
		Data:    data,
		Header:  nats.Header{},
	}
	msg.Header.Set("type", eventType)
	msg.Header.Set("event_id", eventID)
	msg.Header.Set("aggregate_id", aggregateID)
	msg.Header.Set("occurred_at", occurredAt)
	msg.Header.Set("schema_version", "1")
	msg.Header.Set("source", "workflow-command")

	// PublishAsync is better for throughput, but interface expects error on Publish.
	// Publish (Sync) waits for Ack.
	_, err = p.js.PublishMsg(ctx, msg)
	if err != nil {
		return fmt.Errorf("failed to publish to nats: %w", err)
	}

	return nil
}

func (p *NatsPublisher) Close() error {
	// Drain before closing?
	return p.nc.Drain()
}
