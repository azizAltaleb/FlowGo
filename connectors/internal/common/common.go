package common

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

func EnvOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func LoadInstanceVars(ctx context.Context, baseURL, token string, processInstanceKey int64) (map[string]any, error) {
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

func StringVar(vars map[string]any, key string) string {
	v, _ := vars[key].(string)
	return strings.TrimSpace(v)
}
