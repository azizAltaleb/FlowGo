package messaging

import (
	"context"
	"fmt"
	"time"

	"workflow-engine/backend/libs/logger"
	"workflow-engine/backend/services/sync-worker/internal/application"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type NatsConsumer struct {
	url     string
	service *application.SyncService
	nc      *nats.Conn
	js      jetstream.JetStream
	log     *logger.Logger
}

func NewNatsConsumer(url string, service *application.SyncService) *NatsConsumer {
	return &NatsConsumer{
		url:     url,
		service: service,
		log:     logger.New("nats-consumer"),
	}
}

func (c *NatsConsumer) Start(ctx context.Context) error {
	nc, err := nats.Connect(c.url)
	if err != nil {
		return fmt.Errorf("failed to connect to nats: %w", err)
	}
	c.nc = nc
	defer nc.Close()

	js, err := jetstream.New(nc)
	if err != nil {
		return fmt.Errorf("failed to create jetstream context: %w", err)
	}
	c.js = js

	// Ensure Stream Exists
	ctxStream, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	streamName := "WORKFLOW_EVENTS"
	_, err = js.CreateOrUpdateStream(ctxStream, jetstream.StreamConfig{
		Name:     streamName,
		Subjects: []string{"workflow.events.>"},
	})
	if err != nil {
		return fmt.Errorf("failed to create stream: %w", err)
	}

	// Create Durable Consumer
	consumer, err := js.CreateOrUpdateConsumer(ctxStream, streamName, jetstream.ConsumerConfig{
		Durable:   "SYNC_WORKER",
		AckPolicy: jetstream.AckExplicitPolicy,
	})
	if err != nil {
		return fmt.Errorf("failed to create consumer: %w", err)
	}

	c.log.Info(ctx, "nats consumer started", map[string]any{"stream": streamName})

	// Consume
	iter, err := consumer.Consume(func(msg jetstream.Msg) {
		// Process Message
		meta, _ := msg.Metadata()
		headers := msg.Headers()
		eventType := headers.Get("type")

		c.log.Debug(ctx, "received nats message", map[string]any{
			"subject": msg.Subject(),
			"type":    eventType,
			"seq":     meta.Sequence.Stream,
		})

		if eventType == "" {
			// Try to derive from subject if header missing?
			// For now, require header.
			c.log.Warn(ctx, "missing event type header", nil)
			msg.Ack()
			return
		}

		// Process
		// Note: MsgCtx is not passed from Consume callback in standard way?
		// We can create a new context or use background?
		// Ideally pass correlation ID if available.

		if err := c.service.ProcessEvent(ctx, eventType, msg.Data()); err != nil {
			c.log.Error(ctx, "process event failed", map[string]any{"error": err.Error()})
			// Nak or Term?
			msg.Nak()
			return
		}

		msg.Ack()
	})
	if err != nil {
		return fmt.Errorf("failed to start consume: %w", err)
	}
	defer iter.Stop()

	<-ctx.Done()
	return ctx.Err()
}
