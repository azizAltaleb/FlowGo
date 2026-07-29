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
)

const jobType = "io.artificialflow.connector.http"

func main() {
	baseURL := envOr("ARTIFICIALFLOW_BASE_URL", "http://localhost:9100/api")
	token := os.Getenv("ARTIFICIALFLOW_TOKEN")
	allowed := parseAllowlist(os.Getenv("HTTP_CONNECTOR_ALLOWED_HOSTS"))

	client, err := worker.NewClient(worker.ClientConfig{
		BaseURL:     baseURL,
		BearerToken: token,
	})
	if err != nil {
		log.Fatal(err)
	}

	w, err := worker.NewWorker(client, worker.WorkerConfig{
		JobType:         jobType,
		WorkerName:      envOr("WORKER_NAME", "http-connector"),
		MaxJobs:         5,
		ActivateTimeout: 5 * time.Second,
		LockDuration:    60 * time.Second,
		Handler: func(ctx context.Context, job model.Job) (map[string]any, error) {
			vars, err := loadInstanceVars(ctx, baseURL, token, job.ProcessInstanceKey)
			if err != nil {
				return nil, err
			}
			return executeHTTP(ctx, vars, allowed)
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	log.Printf("HTTP connector listening for %s", jobType)
	if err := w.Run(ctx); err != nil && err != context.Canceled {
		log.Fatal(err)
	}
}

func loadInstanceVars(ctx context.Context, baseURL, token string, processInstanceKey int64) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/instances/%d", strings.TrimRight(baseURL, "/"), processInstanceKey), nil)
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("load instance %d: %s", processInstanceKey, strings.TrimSpace(string(body)))
	}
	var payload struct {
		Context map[string]any `json:"context"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	if payload.Context == nil {
		return map[string]any{}, nil
	}
	return payload.Context, nil
}

func executeHTTP(ctx context.Context, vars map[string]any, allowed map[string]struct{}) (map[string]any, error) {
	urlStr, _ := vars["url"].(string)
	if urlStr == "" {
		return nil, fmt.Errorf("instance variable url is required")
	}
	parsed, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}
	if len(allowed) > 0 {
		if _, ok := allowed[strings.ToLower(parsed.Hostname())]; !ok {
			return nil, fmt.Errorf("host %q not in HTTP_CONNECTOR_ALLOWED_HOSTS", parsed.Hostname())
		}
	}
	method, _ := vars["method"].(string)
	if method == "" {
		method = http.MethodPost
	}
	timeoutMs := 10000
	switch v := vars["timeoutMs"].(type) {
	case float64:
		if v > 0 {
			timeoutMs = int(v)
		}
	case int:
		if v > 0 {
			timeoutMs = v
		}
	}
	var bodyReader io.Reader
	if body, ok := vars["body"]; ok && body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, strings.ToUpper(method), urlStr, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if headers, ok := vars["headers"].(map[string]any); ok {
		for k, v := range headers {
			req.Header.Set(k, fmt.Sprint(v))
		}
	}
	httpClient := &http.Client{Timeout: time.Duration(timeoutMs) * time.Millisecond}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	return map[string]any{
		"httpStatus": resp.StatusCode,
		"httpBody":   string(raw),
		"httpOk":     resp.StatusCode >= 200 && resp.StatusCode < 300,
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

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
