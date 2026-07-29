package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/artificialflow/artificialflow/backend/libs/model"
	"github.com/artificialflow/artificialflow/backend/libs/worker"
	"github.com/artificialflow/artificialflow/connectors/internal/common"
	"github.com/segmentio/kafka-go"
)

const jobType = "io.artificialflow.connector.kafka"

func main() {
	baseURL := common.EnvOr("ARTIFICIALFLOW_BASE_URL", "http://localhost:9100/api")
	token := os.Getenv("ARTIFICIALFLOW_TOKEN")
	brokers := strings.Split(common.EnvOr("KAFKA_BROKERS", "localhost:9092"), ",")

	client, err := worker.NewClient(worker.ClientConfig{BaseURL: baseURL, BearerToken: token})
	if err != nil {
		log.Fatal(err)
	}
	w, err := worker.NewWorker(client, worker.WorkerConfig{
		JobType:         jobType,
		WorkerName:      common.EnvOr("WORKER_NAME", "kafka-connector"),
		MaxJobs:         5,
		ActivateTimeout: 5 * time.Second,
		LockDuration:    60 * time.Second,
		Handler: func(ctx context.Context, job model.Job) (map[string]any, error) {
			vars, err := common.LoadInstanceVars(ctx, baseURL, token, job.ProcessInstanceKey)
			if err != nil {
				return nil, err
			}
			return publishKafka(ctx, brokers, vars)
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	log.Printf("kafka connector listening for %s brokers=%v", jobType, brokers)
	if err := w.Run(ctx); err != nil && err != context.Canceled {
		log.Fatal(err)
	}
}

func publishKafka(ctx context.Context, brokers []string, vars map[string]any) (map[string]any, error) {
	topic := common.StringVar(vars, "kafkaTopic")
	if topic == "" {
		topic = common.StringVar(vars, "topic")
	}
	if topic == "" {
		return nil, fmt.Errorf("kafkaTopic is required")
	}
	key := common.StringVar(vars, "kafkaKey")
	var value []byte
	if raw, ok := vars["kafkaValue"].(string); ok && raw != "" {
		value = []byte(raw)
	} else if payload, ok := vars["payload"]; ok {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		value = b
	} else {
		value = []byte("{}")
	}
	writer := &kafka.Writer{
		Addr:         kafka.TCP(trimBrokers(brokers)...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireOne,
	}
	defer writer.Close()
	msg := kafka.Message{Value: value}
	if key != "" {
		msg.Key = []byte(key)
	}
	if err := writer.WriteMessages(ctx, msg); err != nil {
		return nil, err
	}
	return map[string]any{
		"kafkaTopic":   topic,
		"kafkaBytes":   len(value),
		"kafkaPublished": true,
	}, nil
}

func trimBrokers(in []string) []string {
	out := make([]string, 0, len(in))
	for _, b := range in {
		b = strings.TrimSpace(b)
		if b != "" {
			out = append(out, b)
		}
	}
	return out
}
