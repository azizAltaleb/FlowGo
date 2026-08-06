package common

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

const (
	MaxHeaderCount     = 50
	MaxHeaderTotalSize = 8192
	MaxHeaderKeyLen    = 256
	MaxHeaderValueLen  = 4096
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

func ParseAllowlist(raw string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, p := range strings.Split(raw, ",") {
		p = strings.ToLower(strings.TrimSpace(p))
		if p != "" {
			out[p] = struct{}{}
		}
	}
	return out
}

// AllowAnyHost reports whether CONNECTOR_ALLOW_ANY_HOST-style env is enabled.
func AllowAnyHost(envKey string) bool {
	v := strings.TrimSpace(os.Getenv(envKey))
	return strings.EqualFold(v, "true") || v == "1"
}

// ValidateAllowlistRequired fails when the allowlist is empty and allow-any is not set.
// Secure by default: production must configure an allowlist or explicitly opt into any host.
func ValidateAllowlistRequired(allowed map[string]struct{}, allowAnyEnvKey, allowlistEnvKey string) error {
	if len(allowed) > 0 {
		return nil
	}
	if AllowAnyHost(allowAnyEnvKey) {
		return nil
	}
	return fmt.Errorf("%s is empty and %s is not true; set an allowlist or explicitly allow any host for local development", allowlistEnvKey, allowAnyEnvKey)
}

// HostAllowed reports whether hostname is in the allowlist (or allowlist is empty / any-host mode).
// Call ValidateAllowlistRequired before using an empty allowlist in production paths.
func HostAllowed(hostname string, allowed map[string]struct{}) bool {
	if len(allowed) == 0 {
		return true
	}
	_, ok := allowed[strings.ToLower(hostname)]
	return ok
}

// CheckRedirectAllowlist returns a CheckRedirect func that re-validates redirect hosts.
func CheckRedirectAllowlist(allowed map[string]struct{}) func(req *http.Request, via []*http.Request) error {
	return func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return fmt.Errorf("stopped after 10 redirects")
		}
		host := req.URL.Hostname()
		if !HostAllowed(host, allowed) {
			return fmt.Errorf("redirect host %q not in connector allowlist", host)
		}
		return nil
	}
}

// AssertURLHostAllowed parses urlStr and ensures the host is allowlisted.
func AssertURLHostAllowed(urlStr string, allowed map[string]struct{}, allowlistEnvKey string) (*url.URL, error) {
	parsed, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}
	host := parsed.Hostname()
	if host == "" {
		return nil, fmt.Errorf("url host is required")
	}
	if !HostAllowed(host, allowed) {
		return nil, fmt.Errorf("host %q not in %s", host, allowlistEnvKey)
	}
	return parsed, nil
}

// FailOnNon2xxDefault returns whether non-2xx responses should fail the job.
// Variable failOnNon2xx wins when present; otherwise envKey (default true when unset).
func FailOnNon2xxDefault(vars map[string]any, envKey string) bool {
	if v, ok := vars["failOnNon2xx"]; ok {
		return AsBool(v, true)
	}
	env := strings.TrimSpace(os.Getenv(envKey))
	if env == "" {
		return true
	}
	return AsBool(env, true)
}

// AsBool coerces common JSON/env representations to bool.
func AsBool(v any, def bool) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		s := strings.TrimSpace(strings.ToLower(t))
		switch s {
		case "true", "1", "yes", "on":
			return true
		case "false", "0", "no", "off":
			return false
		case "":
			return def
		default:
			return def
		}
	case float64:
		return t != 0
	case int:
		return t != 0
	default:
		return def
	}
}

// ApplyHeaders sets headers from a map or JSON object string with size/count bounds.
func ApplyHeaders(req *http.Request, raw any) error {
	headers, err := CoerceStringMap(raw)
	if err != nil {
		return err
	}
	if len(headers) > MaxHeaderCount {
		return fmt.Errorf("too many headers: %d (max %d)", len(headers), MaxHeaderCount)
	}
	total := 0
	for k, v := range headers {
		if len(k) > MaxHeaderKeyLen {
			return fmt.Errorf("header name too long (max %d)", MaxHeaderKeyLen)
		}
		if len(v) > MaxHeaderValueLen {
			return fmt.Errorf("header %q value too long (max %d)", k, MaxHeaderValueLen)
		}
		total += len(k) + len(v)
		if total > MaxHeaderTotalSize {
			return fmt.Errorf("headers exceed max total size (%d bytes)", MaxHeaderTotalSize)
		}
		req.Header.Set(k, v)
	}
	return nil
}

// CoerceStringMap accepts map[string]any or a JSON object string.
func CoerceStringMap(raw any) (map[string]string, error) {
	if raw == nil {
		return map[string]string{}, nil
	}
	switch t := raw.(type) {
	case map[string]string:
		out := make(map[string]string, len(t))
		for k, v := range t {
			out[k] = v
		}
		return out, nil
	case map[string]any:
		out := make(map[string]string, len(t))
		for k, v := range t {
			out[k] = fmt.Sprint(v)
		}
		return out, nil
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return map[string]string{}, nil
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(s), &m); err != nil {
			return nil, fmt.Errorf("headers must be a JSON object: %w", err)
		}
		out := make(map[string]string, len(m))
		for k, v := range m {
			out[k] = fmt.Sprint(v)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("headers must be an object")
	}
}

// CoerceJSONValue accepts already-decoded JSON or a JSON string.
func CoerceJSONValue(raw any) (any, error) {
	if raw == nil {
		return nil, nil
	}
	s, ok := raw.(string)
	if !ok {
		return raw, nil
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	var decoded any
	if err := json.Unmarshal([]byte(s), &decoded); err != nil {
		// Treat as plain string body
		return s, nil
	}
	return decoded, nil
}
