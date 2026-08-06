package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func allowlistForServer(srvURL string) map[string]struct{} {
	u, err := url.Parse(srvURL)
	if err != nil {
		return map[string]struct{}{}
	}
	return map[string]struct{}{strings.ToLower(u.Hostname()): {}}
}

func TestExecuteHTTP_MissingURL(t *testing.T) {
	_, err := executeHTTP(context.Background(), map[string]any{}, map[string]struct{}{"example.com": {}})
	if err == nil || !strings.Contains(err.Error(), "url is required") {
		t.Fatalf("expected missing url error, got %v", err)
	}
}

func TestExecuteHTTP_AllowlistReject(t *testing.T) {
	_, err := executeHTTP(context.Background(), map[string]any{
		"url": "https://evil.example/path",
	}, map[string]struct{}{"good.example": {}})
	if err == nil || !strings.Contains(err.Error(), "not in HTTP_CONNECTOR_ALLOWED_HOSTS") {
		t.Fatalf("expected allowlist error, got %v", err)
	}
}

func TestExecuteHTTP_FailOnNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("nope"))
	}))
	defer srv.Close()

	allowed := allowlistForServer(srv.URL)

	_, err := executeHTTP(context.Background(), map[string]any{
		"url":          srv.URL,
		"method":       "GET",
		"failOnNon2xx": true,
	}, allowed)
	if err == nil || !strings.Contains(err.Error(), "failOnNon2xx") {
		t.Fatalf("expected failOnNon2xx error, got %v", err)
	}

	out, err := executeHTTP(context.Background(), map[string]any{
		"url":          srv.URL,
		"method":       "GET",
		"failOnNon2xx": false,
	}, allowed)
	if err != nil {
		t.Fatalf("failOnNon2xx=false should not error: %v", err)
	}
	if out["httpOk"] != false {
		t.Fatalf("expected httpOk false, got %v", out["httpOk"])
	}
}

func TestExecuteHTTP_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Test") != "1" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	allowed := allowlistForServer(srv.URL)

	out, err := executeHTTP(context.Background(), map[string]any{
		"url":     srv.URL,
		"method":  "GET",
		"headers": map[string]any{"X-Test": "1"},
	}, allowed)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out["httpOk"] != true {
		t.Fatalf("expected httpOk true, got %v", out)
	}
}
