package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/artificialflow/artificialflow/backend/libs/model"
	"github.com/artificialflow/artificialflow/backend/libs/worker"
)

func main() {
	baseURL := envOr("ARTIFICIALFLOW_BASE_URL", "http://localhost:9100/api")
	token := os.Getenv("ARTIFICIALFLOW_TOKEN")
	jobType := envOr("WORKER_JOB_TYPE", "golden-validate")

	client, err := worker.NewClient(worker.ClientConfig{
		BaseURL:     baseURL,
		BearerToken: token,
	})
	if err != nil {
		log.Fatal(err)
	}

	w, err := worker.NewWorker(client, worker.WorkerConfig{
		JobType:         jobType,
		WorkerName:      envOr("WORKER_NAME", "go-sample-worker"),
		MaxJobs:         5,
		ActivateTimeout: 5 * time.Second,
		LockDuration:    30 * time.Second,
		Handler: func(ctx context.Context, job model.Job) (map[string]any, error) {
			log.Printf("handling job %d type=%s element=%s", job.Key, job.Type, job.ElementID)
			return map[string]any{"validatedBy": "go-sample-worker", "ok": true}, nil
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	log.Printf("Go worker listening for %s at %s", jobType, baseURL)
	if err := w.Run(ctx); err != nil && err != context.Canceled {
		log.Fatal(err)
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
