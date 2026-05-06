package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	esadapter "workflow-engine/backend/libs/elasticsearch"
	"workflow-engine/backend/libs/logger"
	"workflow-engine/backend/services/sync-worker/internal/application"
	"workflow-engine/backend/services/sync-worker/internal/infrastructure/messaging"
	"workflow-engine/backend/services/sync-worker/internal/infrastructure/persistence"
)

func main() {
	log := logger.New("sync-worker")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	kafkaBrokers := envOrDefault("KAFKA_BROKERS", "localhost:9092")
	groupID := envOrDefault("KAFKA_GROUP_ID", "workflowsa-sync-worker")
	eventTopic := strings.TrimSpace(envOrDefault("KAFKA_TOPIC_EVENTS", "workflow.events.v1"))
	projectionContract := normalizeProjectionContract(envOrDefault("SYNC_PROJECTION_CONTRACT", "hybrid"))
	kafkaTopics := parseTopicsEnv(os.Getenv("KAFKA_TOPICS"))
	if len(kafkaTopics) == 0 {
		if eventTopic != "" {
			kafkaTopics = []string{eventTopic}
			log.Warn(ctx, "KAFKA_TOPICS is empty; using legacy KAFKA_TOPIC_EVENTS fallback", map[string]any{
				"topic": eventTopic,
			})
		}
	}
	if err := validateProjectionContract(projectionContract, kafkaTopics, eventTopic); err != nil {
		log.Error(ctx, "invalid sync projection contract configuration", map[string]any{
			"error":               err.Error(),
			"projection_contract": projectionContract,
			"topics":              kafkaTopics,
			"event_topic":         eventTopic,
		})
		os.Exit(1)
	}
	dlqTopic := strings.TrimSpace(envOrDefault("KAFKA_DLQ_TOPIC", "workflow.sync.dlq"))
	maxProcessRetries := envInt("SYNC_MAX_PROCESS_RETRIES", 3)
	retryBackoff := parseDurationEnv("SYNC_RETRY_BACKOFF", 500*time.Millisecond)

	esAddr := envOrDefault("ES_ADDR", "http://localhost:9200")
	indexPrefix := os.Getenv("ES_INDEX_PREFIX")
	logLevel := envOrDefault("LOG_LEVEL", "INFO")
	eventBusType := strings.ToLower(strings.TrimSpace(envOrDefault("EVENT_BUS_TYPE", "kafka")))
	connectBootstrapEnabled := envBool("CONNECT_BOOTSTRAP_ENABLED", true)
	connectURL := strings.TrimSpace(envOrDefault("CONNECT_URL", "http://connect:8083"))
	connectorName := strings.TrimSpace(envOrDefault("CONNECTOR_NAME", "workflowsa-postgres-connector"))
	connectorFile := strings.TrimSpace(os.Getenv("CONNECTOR_FILE"))
	connectorJSON := strings.TrimSpace(os.Getenv("CONNECTOR_JSON"))
	connectWaitTimeoutSec := envInt("CONNECT_WAIT_TIMEOUT_SEC", 180)
	if connectWaitTimeoutSec < 1 {
		connectWaitTimeoutSec = 180
	}
	connectWaitTimeout := time.Duration(connectWaitTimeoutSec) * time.Second
	metricsAddr := strings.TrimSpace(envOrDefault("SYNC_METRICS_ADDR", ":8092"))
	freshnessSLOSec := int64(envInt("SYNC_FRESHNESS_SLO_SEC", 120))
	failOnStale := envBool("SYNC_HEALTH_FAIL_ON_STALE", false)

	log.Info(ctx, "initializing sync-worker", map[string]any{
		"kafka_brokers":     kafkaBrokers,
		"kafka_group":       groupID,
		"kafka_topics":      kafkaTopics,
		"event_topic":       eventTopic,
		"contract":          projectionContract,
		"kafka_dlq":         dlqTopic,
		"retry_max":         maxProcessRetries,
		"retry_backoff":     retryBackoff.String(),
		"es_addr":           esAddr,
		"index_prefix":      indexPrefix,
		"event_bus_type":    eventBusType,
		"connect_bootstrap": connectBootstrapEnabled,
		"connector_name":    connectorName,
		"metrics_addr":      metricsAddr,
		"freshness_slo_sec": freshnessSLOSec,
		"fail_on_stale":     failOnStale,
		"log_level":         logLevel,
	})

	esClient, err := esadapter.NewClient(esadapter.Config{Addresses: splitAndTrim(esAddr)})
	if err != nil {
		log.Error(ctx, "failed to init elasticsearch client", map[string]any{
			"error": err.Error(),
		})
		os.Exit(1)
	}
	esClientRepo := esadapter.NewRepository(esClient)

	// DDD Assembly
	repo := persistence.NewESRepository(esClientRepo)
	service := application.NewSyncService(repo, indexPrefix)
	// Messaging Configuration
	var consumer messaging.EventConsumer

	switch eventBusType {
	case "nats":
		natsURL := envOrDefault("NATS_URL", "nats://localhost:4222")
		log.Info(ctx, "initializing nats consumer", map[string]any{"url": natsURL})
		consumer = messaging.NewNatsConsumer(natsURL, service)
	default:
		// Default to Kafka
		log.Info(ctx, "initializing kafka consumer", map[string]any{
			"brokers": kafkaBrokers,
			"group":   groupID,
			"topics":  kafkaTopics,
		})
		consumer = messaging.NewKafkaConsumer(messaging.Config{
			Brokers:           splitAndTrim(kafkaBrokers),
			GroupID:           groupID,
			Topics:            kafkaTopics,
			DLQTopic:          dlqTopic,
			MaxProcessRetries: maxProcessRetries,
			RetryBackoff:      retryBackoff,
		}, service)
	}

	var statsProvider messaging.ConsumerStatsProvider
	if provider, ok := consumer.(messaging.ConsumerStatsProvider); ok {
		statsProvider = provider
	}
	startSyncHealthServer(ctx, log, metricsAddr, statsProvider, freshnessSLOSec, failOnStale)

	if eventBusType != "nats" {
		if err := ensureConnectorBootstrap(ctx, log, connectorBootstrapOptions{
			Enabled:       connectBootstrapEnabled,
			ConnectURL:    connectURL,
			ConnectorName: connectorName,
			ConnectorFile: connectorFile,
			ConnectorJSON: connectorJSON,
			WaitTimeout:   connectWaitTimeout,
			PollInterval:  3 * time.Second,
		}); err != nil {
			log.Error(ctx, "failed to bootstrap connector", map[string]any{"error": err.Error()})
			os.Exit(1)
		}
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		log.Info(ctx, "received shutdown signal", map[string]any{
			"signal": sig.String(),
		})
		cancel()
	}()

	time.Sleep(500 * time.Millisecond)

	log.Info(ctx, "sync-worker starting", nil)
	if err := consumer.Start(ctx); err != nil && err != context.Canceled {
		log.Error(ctx, "sync-worker stopped with error", map[string]any{
			"error": err.Error(),
		})
		os.Exit(1)
	}

	log.Info(ctx, "sync-worker stopped gracefully", nil)
}

func envOrDefault(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

func splitAndTrim(v string) []string {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}

func parseTopicsEnv(raw string) []string {
	topics := splitAndTrim(raw)
	if len(topics) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(topics))
	unique := make([]string, 0, len(topics))
	for _, topic := range topics {
		if _, ok := seen[topic]; ok {
			continue
		}
		seen[topic] = struct{}{}
		unique = append(unique, topic)
	}
	return unique
}

func normalizeProjectionContract(raw string) string {
	contract := strings.ToLower(strings.TrimSpace(raw))
	switch contract {
	case "", "hybrid":
		return "hybrid"
	case "event-first", "event_first", "eventfirst", "events":
		return "event-first"
	case "debezium":
		return "debezium"
	default:
		return contract
	}
}

func validateProjectionContract(contract string, topics []string, eventTopic string) error {
	if len(topics) == 0 {
		return fmt.Errorf("kafka topics are required for projection contract validation")
	}

	contract = normalizeProjectionContract(contract)
	eventTopic = strings.TrimSpace(eventTopic)
	if eventTopic == "" {
		eventTopic = "workflow.events.v1"
	}

	hasEventTopic := containsTopic(topics, eventTopic)
	hasDebeziumTopics := containsDebeziumTopics(topics)

	switch contract {
	case "hybrid":
		if !hasEventTopic {
			return fmt.Errorf("hybrid contract requires event topic %q in KAFKA_TOPICS", eventTopic)
		}
		if !hasDebeziumTopics {
			return fmt.Errorf("hybrid contract requires at least one Debezium table topic in KAFKA_TOPICS (expected *.public.<table>)")
		}
		return nil
	case "event-first":
		if !hasEventTopic {
			return fmt.Errorf("event-first contract requires event topic %q in KAFKA_TOPICS", eventTopic)
		}
		if hasDebeziumTopics {
			return fmt.Errorf("event-first contract does not allow Debezium table topics in KAFKA_TOPICS")
		}
		return nil
	case "debezium":
		if !hasDebeziumTopics {
			return fmt.Errorf("debezium contract requires at least one Debezium table topic in KAFKA_TOPICS")
		}
		return nil
	default:
		return fmt.Errorf("unsupported SYNC_PROJECTION_CONTRACT=%q (expected hybrid|event-first|debezium)", contract)
	}
}

func containsTopic(topics []string, expected string) bool {
	expected = strings.TrimSpace(expected)
	if expected == "" {
		return false
	}
	for _, topic := range topics {
		if strings.TrimSpace(topic) == expected {
			return true
		}
	}
	return false
}

func containsDebeziumTopics(topics []string) bool {
	for _, topic := range topics {
		topic = strings.TrimSpace(topic)
		if topic == "" {
			continue
		}
		parts := strings.Split(topic, ".")
		if len(parts) >= 3 && strings.EqualFold(parts[len(parts)-2], "public") {
			return true
		}
	}
	return false
}

func startSyncHealthServer(ctx context.Context, log *logger.Logger, addr string, provider messaging.ConsumerStatsProvider, sloSec int64, failOnStale bool) {
	if strings.TrimSpace(addr) == "" {
		return
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		snapshot := messaging.ConsumerStatsSnapshot{}
		if provider != nil {
			snapshot = provider.Snapshot()
		}
		status, lagSec, stale, httpStatus := evaluateSyncHealth(snapshot, sloSec, failOnStale)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(httpStatus)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":             status,
			"lagSeconds":         lagSec,
			"stale":              stale,
			"freshnessSloSec":    sloSec,
			"processed":          snapshot.Processed,
			"succeeded":          snapshot.Succeeded,
			"failed":             snapshot.Failed,
			"retried":            snapshot.Retried,
			"dlqPublished":       snapshot.DLQPublished,
			"lastProcessedAt":    snapshot.LastProcessedAt,
			"topics":             snapshot.Topics,
			"failOnStaleEnabled": failOnStale,
		})
	})

	server := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()
	go func() {
		log.Info(ctx, "sync-worker health server starting", map[string]any{"addr": addr})
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error(ctx, "sync-worker health server stopped with error", map[string]any{"error": err.Error()})
		}
	}()
}

func evaluateSyncHealth(snapshot messaging.ConsumerStatsSnapshot, sloSec int64, failOnStale bool) (string, int64, bool, int) {
	lagSec := int64(-1)
	if !snapshot.LastProcessedAt.IsZero() {
		lagSec = int64(time.Since(snapshot.LastProcessedAt).Seconds())
		if lagSec < 0 {
			lagSec = 0
		}
	}

	if sloSec <= 0 {
		return "ok", lagSec, false, 200
	}
	if snapshot.LastProcessedAt.IsZero() {
		return "starting", lagSec, false, 200
	}
	if lagSec <= sloSec {
		return "ok", lagSec, false, 200
	}
	if failOnStale {
		return "unhealthy", lagSec, true, 503
	}
	return "degraded", lagSec, true, 200
}

func envInt(key string, defaultValue int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return defaultValue
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return defaultValue
	}
	return v
}

func parseDurationEnv(key string, defaultValue time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return defaultValue
	}
	v, err := time.ParseDuration(raw)
	if err != nil {
		return defaultValue
	}
	return v
}

func envBool(key string, defaultValue bool) bool {
	val := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	switch val {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return defaultValue
	}
}
