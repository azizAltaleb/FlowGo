package messaging

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/artificialflow/artificialflow/backend/services/sync-worker/internal/application"
	"github.com/segmentio/kafka-go"
)

func TestNewKafkaConsumerAppliesConfigDefaults(t *testing.T) {
	consumer := NewKafkaConsumer(Config{
		MaxProcessRetries: -1,
		RetryBackoff:      0,
	}, nil)

	if consumer.cfg.MaxProcessRetries != 0 {
		t.Fatalf("expected max retries defaulted to 0, got %d", consumer.cfg.MaxProcessRetries)
	}
	if consumer.cfg.RetryBackoff <= 0 {
		t.Fatalf("expected retry backoff to be defaulted, got %s", consumer.cfg.RetryBackoff)
	}
}

func TestReaderConfigUsesSingleGroupSubscriptionForAllTopics(t *testing.T) {
	topics := []string{"workflow.events.v1", "flowgo.public.process", "flowgo.public.job"}
	consumer := NewKafkaConsumer(Config{
		Brokers: []string{"kafka:29092"},
		GroupID: "flowgo-sync-worker",
		Topics:  topics,
	}, nil)

	cfg := consumer.readerConfig()
	if cfg.Topic != "" {
		t.Fatalf("expected single-topic subscription to be empty, got %q", cfg.Topic)
	}
	if !reflect.DeepEqual(cfg.GroupTopics, topics) {
		t.Fatalf("expected group topics %v, got %v", topics, cfg.GroupTopics)
	}
	if cfg.GroupID != "flowgo-sync-worker" {
		t.Fatalf("expected group id to be preserved, got %q", cfg.GroupID)
	}
	if !cfg.WatchPartitionChanges {
		t.Fatal("expected partition watching so fresh Debezium topics trigger a rebalance")
	}
	if cfg.PartitionWatchInterval <= 0 {
		t.Fatalf("expected a positive partition watch interval, got %s", cfg.PartitionWatchInterval)
	}
}

func TestReaderDiscoversTopicCreatedAfterInitialGroupJoin(t *testing.T) {
	broker := os.Getenv("KAFKA_INTEGRATION_BROKER")
	if broker == "" {
		t.Skip("set KAFKA_INTEGRATION_BROKER to run Kafka integration coverage")
	}

	suffix := time.Now().UnixNano()
	topic := fmt.Sprintf("flowgo.sync.late-topic.%d", suffix)
	groupID := fmt.Sprintf("flowgo-sync-late-topic-%d", suffix)
	consumer := NewKafkaConsumer(Config{
		Brokers: []string{broker},
		GroupID: groupID,
		Topics:  []string{topic},
	}, nil)
	reader := kafka.NewReader(consumer.readerConfig())
	defer reader.Close()
	defer deleteKafkaTopic(t, broker, topic)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	messageCh := make(chan kafka.Message, 1)
	errorCh := make(chan error, 1)
	go func() {
		for {
			message, err := reader.FetchMessage(ctx)
			if err == nil {
				messageCh <- message
				return
			}
			if ctx.Err() != nil {
				errorCh <- ctx.Err()
				return
			}
		}
	}()

	// Let the consumer group complete its first assignment while the topic is
	// still absent, reproducing a clean-install Debezium startup race.
	time.Sleep(3 * time.Second)
	if err := createKafkaTopic(broker, topic); err != nil {
		t.Fatalf("create late topic: %v", err)
	}
	writer := &kafka.Writer{
		Addr:  kafka.TCP(broker),
		Topic: topic,
	}
	if err := writer.WriteMessages(ctx, kafka.Message{Value: []byte(`{"late":true}`)}); err != nil {
		_ = writer.Close()
		t.Fatalf("create late topic and publish message: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close Kafka writer: %v", err)
	}

	select {
	case message := <-messageCh:
		if message.Topic != topic {
			t.Fatalf("expected message from %q, got %q", topic, message.Topic)
		}
	case err := <-errorCh:
		t.Fatalf("reader did not discover late-created topic: %v", err)
	case <-ctx.Done():
		t.Fatalf("reader did not rebalance for late-created topic: %v", ctx.Err())
	}
}

func createKafkaTopic(broker, topic string) error {
	connection, err := kafka.Dial("tcp", broker)
	if err != nil {
		return err
	}
	defer connection.Close()
	controller, err := connection.Controller()
	if err != nil {
		return err
	}
	controllerConnection, err := kafka.Dial("tcp", controller.Host+":"+fmt.Sprint(controller.Port))
	if err != nil {
		return err
	}
	defer controllerConnection.Close()
	return controllerConnection.CreateTopics(kafka.TopicConfig{
		Topic:             topic,
		NumPartitions:     1,
		ReplicationFactor: 1,
	})
}

func deleteKafkaTopic(t *testing.T, broker, topic string) {
	t.Helper()
	connection, err := kafka.Dial("tcp", broker)
	if err != nil {
		t.Logf("cleanup: connect to Kafka: %v", err)
		return
	}
	defer connection.Close()
	controller, err := connection.Controller()
	if err != nil {
		t.Logf("cleanup: resolve Kafka controller: %v", err)
		return
	}
	controllerConnection, err := kafka.Dial("tcp", controller.Host+":"+fmt.Sprint(controller.Port))
	if err != nil {
		t.Logf("cleanup: connect to Kafka controller: %v", err)
		return
	}
	defer controllerConnection.Close()
	if err := controllerConnection.DeleteTopics(topic); err != nil {
		t.Logf("cleanup: delete topic %q: %v", topic, err)
	}
}

func TestSnapshotIncludesCountersAndTopicStats(t *testing.T) {
	consumer := NewKafkaConsumer(Config{}, nil)
	now := time.Now().UTC()

	consumer.processedCount.Store(10)
	consumer.successCount.Store(8)
	consumer.failureCount.Store(2)
	consumer.retryCount.Store(3)
	consumer.dlqCount.Store(1)
	consumer.lastProcessed.Store(now.UnixNano())
	consumer.recordTopicOutcome("flowgo.public.variable", 42, nil)

	snapshot := consumer.Snapshot()
	if snapshot.Processed != 10 || snapshot.Succeeded != 8 || snapshot.Failed != 2 {
		t.Fatalf("unexpected counters in snapshot: %+v", snapshot)
	}
	if snapshot.Retried != 3 || snapshot.DLQPublished != 1 {
		t.Fatalf("unexpected retry/dlq counters in snapshot: %+v", snapshot)
	}
	if snapshot.LastProcessedAt.IsZero() {
		t.Fatalf("expected last processed timestamp to be set")
	}

	topicStats, ok := snapshot.Topics["flowgo.public.variable"]
	if !ok {
		t.Fatalf("expected topic stats entry")
	}
	if topicStats.LastOffset != 42 {
		t.Fatalf("expected last offset 42, got %d", topicStats.LastOffset)
	}
}

func TestProcessMessageWithRetry_SucceedsAfterRetry(t *testing.T) {
	repo := &flakyRepo{failUpsertAttempts: 1}
	service := application.NewSyncService(repo, "flowgo")
	consumer := NewKafkaConsumer(Config{MaxProcessRetries: 2, RetryBackoff: time.Millisecond}, service)

	err := consumer.processMessageWithRetry(context.Background(), kafka.Message{
		Topic:     "flowgo.public.process_instance",
		Partition: 0,
		Offset:    9,
		Value:     []byte(`{"before":null,"after":{"key":1},"op":"c"}`),
	})
	if err != nil {
		t.Fatalf("expected retry to eventually succeed, got error: %v", err)
	}
	if got := consumer.retryCount.Load(); got != 1 {
		t.Fatalf("expected retry count=1, got %d", got)
	}
	if repo.upsertCalls != 2 {
		t.Fatalf("expected 2 upsert attempts, got %d", repo.upsertCalls)
	}
}

func TestProcessMessageWithRetry_FailsAfterMaxAttempts(t *testing.T) {
	repo := &flakyRepo{failUpsertAttempts: 5}
	service := application.NewSyncService(repo, "flowgo")
	consumer := NewKafkaConsumer(Config{MaxProcessRetries: 1, RetryBackoff: time.Millisecond}, service)

	err := consumer.processMessageWithRetry(context.Background(), kafka.Message{
		Topic:     "flowgo.public.process_instance",
		Partition: 0,
		Offset:    10,
		Value:     []byte(`{"before":null,"after":{"key":1},"op":"c"}`),
	})
	if err == nil {
		t.Fatalf("expected retries to exhaust with an error")
	}
	if got := consumer.retryCount.Load(); got != 1 {
		t.Fatalf("expected retry count=1, got %d", got)
	}
	if repo.upsertCalls != 2 {
		t.Fatalf("expected 2 upsert attempts with max retries=1, got %d", repo.upsertCalls)
	}
}

type flakyRepo struct {
	failUpsertAttempts int
	upsertCalls        int
}

func (r *flakyRepo) Upsert(_ context.Context, _ string, _ string, _ map[string]any) error {
	r.upsertCalls++
	if r.upsertCalls <= r.failUpsertAttempts {
		return errors.New("injected upsert failure")
	}
	return nil
}

func (r *flakyRepo) Delete(_ context.Context, _ string, _ string) error {
	return nil
}

func (r *flakyRepo) UpdateWithScript(_ context.Context, _ string, _ string, _ string, _ map[string]any) error {
	return nil
}
