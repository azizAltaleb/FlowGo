package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/artificialflow/artificialflow/backend/libs/model"
	"github.com/artificialflow/artificialflow/backend/libs/worker"
	"github.com/artificialflow/artificialflow/connectors/internal/common"
)

const jobType = "io.artificialflow.connector.webhook"

func main() {
	baseURL := common.EnvOr("ARTIFICIALFLOW_BASE_URL", "http://localhost:9100/api")
	token := os.Getenv("ARTIFICIALFLOW_TOKEN")
	allowed := parseAllowlist(os.Getenv("WEBHOOK_CONNECTOR_ALLOWED_HOSTS"))

	client, err := worker.NewClient(worker.ClientConfig{BaseURL: baseURL, BearerToken: token})
	if err != nil {
		log.Fatal(err)
	}
	w, err := worker.NewWorker(client, worker.WorkerConfig{
		JobType:         jobType,
		WorkerName:      common.EnvOr("WORKER_NAME", "webhook-connector"),
		MaxJobs:         5,
		ActivateTimeout: 5 * time.Second,
		LockDuration:    60 * time.Second,
		Handler: func(ctx context.Context, job model.Job) (map[string]any, error) {
			vars, err := common.LoadInstanceVars(ctx, baseURL, token, job.ProcessInstanceKey)
			if err != nil {
				return nil, err
			}
			return executeWebhook(ctx, vars, allowed)
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	log.Printf("webhook connector listening for %s", jobType)
	if err := w.Run(ctx); err != nil && err != context.Canceled {
		log.Fatal(err)
	}
}

func executeWebhook(ctx context.Context, vars map[string]any, allowed map[string]struct{}) (map[string]any, error) {
	urlStr := common.StringVar(vars, "webhookUrl")
	if urlStr == "" {
		urlStr = common.StringVar(vars, "url")
	}
	if urlStr == "" {
		return nil, fmt.Errorf("webhookUrl (or url) is required")
	}
	parsed, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}
	if len(allowed) > 0 {
		if _, ok := allowed[strings.ToLower(parsed.Hostname())]; !ok {
			return nil, fmt.Errorf("host %q not in WEBHOOK_CONNECTOR_ALLOWED_HOSTS", parsed.Hostname())
		}
	}
	payload := vars["payload"]
	if payload == nil {
		payload = vars["body"]
	}
	var bodyReader io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(b)
	} else {
		bodyReader = bytes.NewReader([]byte("{}"))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, urlStr, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if token := common.StringVar(vars, "webhookToken"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	return map[string]any{
		"webhookStatus": resp.StatusCode,
		"webhookBody":   string(raw),
		"webhookOk":     resp.StatusCode >= 200 && resp.StatusCode < 300,
	}, nil
}

func parseAllowlist(raw string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, p := range strings.Split(raw, ",") {
		p = strings.ToLower(strings.TrimSpace(p))
		if p != "" {
			out[p] = struct{}{}
		}
	}
	return out
}
