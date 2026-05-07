package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/azizAltaleb/goflow/backend/libs/logger"
	"github.com/azizAltaleb/goflow/backend/libs/metrics"
	"github.com/azizAltaleb/goflow/backend/libs/tracing"
	"github.com/azizAltaleb/goflow/backend/services/workflow-command/internal/application"
	"github.com/azizAltaleb/goflow/backend/services/workflow-command/internal/infrastructure/messaging"
	"github.com/azizAltaleb/goflow/backend/services/workflow-command/internal/infrastructure/persistence"
)

func main() {
	log := logger.New("workflow-runtime")
	ctx := context.Background()

	runtimeEnabled := envBool("RUNTIME_ENABLED", true)
	if !runtimeEnabled {
		log.Info(ctx, "runtime scheduler disabled by config", nil)
		return
	}

	// Initialize tracing for runtime worker.
	tp, err := tracing.InitTracer(ctx, "workflow-runtime")
	if err != nil {
		log.Error(ctx, "failed to init tracer", map[string]any{"error": err.Error()})
	} else {
		defer func() {
			if err := tp.Shutdown(context.Background()); err != nil {
				log.Error(ctx, "failed to shutdown tracer", map[string]any{"error": err.Error()})
			}
		}()
	}

	pgDSN := os.Getenv("PG_DSN")
	if pgDSN == "" {
		pgDSN = "host=localhost user=user password=password dbname=workflow_db port=5433 sslmode=disable TimeZone=UTC"
	}

	tickInterval := parseDurationEnv("RUNTIME_TICK_INTERVAL", time.Second)
	taskTimeout := parseDurationEnv("RUNTIME_TASK_TIMEOUT", 5*time.Second)

	repo, err := persistence.NewPostgresRepository(pgDSN)
	if err != nil {
		log.Error(ctx, "failed to initialize storage", map[string]any{"error": err.Error()})
		os.Exit(1)
	}

	kafkaBrokers := envOrDefault("KAFKA_BROKERS", "localhost:9092")
	eventTopic := envOrDefault("KAFKA_TOPIC_EVENTS", "workflow.events.v1")
	eventBusType := envOrDefault("EVENT_BUS_TYPE", "kafka")
	var publisher messaging.EventPublisher

	switch strings.ToLower(eventBusType) {
	case "nats":
		natsURL := envOrDefault("NATS_URL", "nats://localhost:4222")
		log.Info(ctx, "using nats event bus", map[string]any{"url": natsURL})
		np, err := messaging.NewNatsPublisher(natsURL)
		if err != nil {
			log.Error(ctx, "failed to create nats publisher", map[string]any{"error": err.Error()})
			os.Exit(1)
		}
		publisher = np
	default:
		if kafkaBrokers != "" {
			publisher = messaging.NewKafkaPublisher(splitAndTrim(kafkaBrokers), eventTopic)
			log.Info(ctx, "kafka publisher initialized", map[string]any{
				"brokers": kafkaBrokers,
				"topic":   eventTopic,
			})
		} else {
			publisher = &messaging.NoOpPublisher{}
			log.Warn(ctx, "kafka publisher disabled (using no-op)", nil)
		}
	}
	defer func() {
		if publisher != nil {
			_ = publisher.Close()
		}
	}()

	outboxRelayBatchSize := envInt("OUTBOX_RELAY_BATCH_SIZE", 200)
	if outboxRelayBatchSize <= 0 {
		outboxRelayBatchSize = 200
	}
	outboxRelayMaxAttempts := envInt("OUTBOX_RELAY_MAX_ATTEMPTS", 10)
	idempotencyCleanupEnabled := envBool("IDEMPOTENCY_CLEANUP_ENABLED", true)
	idempotencyRetention := parseDurationEnv("IDEMPOTENCY_RETENTION", 168*time.Hour)
	idempotencyCleanupInterval := parseDurationEnv("IDEMPOTENCY_CLEANUP_INTERVAL", time.Hour)
	idempotencyCleanupTimeout := parseDurationEnv("IDEMPOTENCY_CLEANUP_TIMEOUT", 10*time.Second)
	idempotencyCleanupBatchSize := envInt("IDEMPOTENCY_CLEANUP_BATCH_SIZE", 500)
	if idempotencyCleanupBatchSize <= 0 {
		idempotencyCleanupBatchSize = 500
	}

	eng := application.NewEngine(repo, publisher)
	eng.SetOutboxMaxAttempts(outboxRelayMaxAttempts)

	if idempotencyCleanupEnabled && idempotencyRetention > 0 {
		go func() {
			ticker := time.NewTicker(idempotencyCleanupInterval)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					cleanupCtx, cancel := context.WithTimeout(context.Background(), idempotencyCleanupTimeout)
					result, err := eng.RunIdempotencyCleanup(cleanupCtx, idempotencyRetention, idempotencyCleanupBatchSize)
					cancel()
					if err != nil {
						log.Error(ctx, "idempotency cleanup failed", map[string]any{"error": err.Error()})
						continue
					}
					if result.Deleted > 0 {
						log.Info(ctx, "idempotency cleanup completed", map[string]any{
							"deleted": result.Deleted,
							"cutoff":  result.Cutoff.Format(time.RFC3339),
						})
					}
				}
			}
		}()
	}

	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	metricsAddr := envOrDefault("METRICS_ADDR", ":9091")
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", metrics.Handler())
		srv := &http.Server{Addr: metricsAddr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
		go func() {
			<-ctx.Done()
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = srv.Shutdown(shutdownCtx)
		}()
		log.Info(ctx, "runtime metrics server started", map[string]any{"addr": metricsAddr})
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error(ctx, "runtime metrics server failed", map[string]any{"error": err.Error()})
		}
	}()

	log.Info(ctx, "runtime scheduler started", map[string]any{
		"tick_interval":               tickInterval.String(),
		"task_timeout":                taskTimeout.String(),
		"outbox_relay_max_attempts":   outboxRelayMaxAttempts,
		"idempotency_cleanup_enabled": idempotencyCleanupEnabled && idempotencyRetention > 0,
	})

	for {
		select {
		case sig := <-sigCh:
			log.Info(ctx, "received shutdown signal", map[string]any{"signal": sig.String()})
			return
		case <-ticker.C:
			runCtx, cancel := context.WithTimeout(context.Background(), taskTimeout)
			if _, err := eng.RunOutboxRelayCycle(runCtx, outboxRelayBatchSize); err != nil {
				log.Error(runCtx, "outbox relay cycle failed", map[string]any{"error": err.Error()})
			}
			if err := eng.CheckTimers(runCtx); err != nil {
				log.Error(runCtx, "check timers failed", map[string]any{"error": err.Error()})
			}
			if err := eng.CheckSLAs(runCtx); err != nil {
				log.Error(runCtx, "check slas failed", map[string]any{"error": err.Error()})
			}
			cancel()
		}
	}
}

func envOrDefault(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
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

func parseDurationEnv(key string, defaultValue time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return defaultValue
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return defaultValue
	}
	return d
}

func envBool(key string, defaultValue bool) bool {
	val := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	switch val {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return defaultValue
	}
}
