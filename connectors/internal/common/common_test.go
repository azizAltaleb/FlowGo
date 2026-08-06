package common

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
)

func TestValidateAllowlistRequired(t *testing.T) {
	t.Setenv("HTTP_CONNECTOR_ALLOW_ANY_HOST", "")
	allowed := map[string]struct{}{}
	if err := ValidateAllowlistRequired(allowed, "HTTP_CONNECTOR_ALLOW_ANY_HOST", "HTTP_CONNECTOR_ALLOWED_HOSTS"); err == nil {
		t.Fatal("expected error when allowlist empty and allow-any unset")
	}

	t.Setenv("HTTP_CONNECTOR_ALLOW_ANY_HOST", "true")
	if err := ValidateAllowlistRequired(allowed, "HTTP_CONNECTOR_ALLOW_ANY_HOST", "HTTP_CONNECTOR_ALLOWED_HOSTS"); err != nil {
		t.Fatalf("allow-any should permit empty allowlist: %v", err)
	}

	allowed["api.example.com"] = struct{}{}
	t.Setenv("HTTP_CONNECTOR_ALLOW_ANY_HOST", "")
	if err := ValidateAllowlistRequired(allowed, "HTTP_CONNECTOR_ALLOW_ANY_HOST", "HTTP_CONNECTOR_ALLOWED_HOSTS"); err != nil {
		t.Fatalf("non-empty allowlist should pass: %v", err)
	}
}

func TestCheckRedirectAllowlist(t *testing.T) {
	allowed := map[string]struct{}{"good.example": {}}
	check := CheckRedirectAllowlist(allowed)

	req := &http.Request{URL: &url.URL{Scheme: "https", Host: "good.example", Path: "/next"}}
	if err := check(req, []*http.Request{{}}); err != nil {
		t.Fatalf("allowed redirect host: %v", err)
	}

	bad := &http.Request{URL: &url.URL{Scheme: "https", Host: "evil.example", Path: "/x"}}
	if err := check(bad, []*http.Request{{}}); err == nil {
		t.Fatal("expected reject for non-allowlisted redirect host")
	}
}

func TestFailOnNon2xxDefault(t *testing.T) {
	t.Setenv("HTTP_CONNECTOR_FAIL_ON_NON_2XX", "")
	if !FailOnNon2xxDefault(map[string]any{}, "HTTP_CONNECTOR_FAIL_ON_NON_2XX") {
		t.Fatal("default should be true")
	}
	if FailOnNon2xxDefault(map[string]any{"failOnNon2xx": false}, "HTTP_CONNECTOR_FAIL_ON_NON_2XX") {
		t.Fatal("var false should win")
	}
	t.Setenv("HTTP_CONNECTOR_FAIL_ON_NON_2XX", "false")
	if FailOnNon2xxDefault(map[string]any{}, "HTTP_CONNECTOR_FAIL_ON_NON_2XX") {
		t.Fatal("env false should disable when var missing")
	}
}

func TestApplyHeadersBounds(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
	headers := map[string]any{}
	for i := 0; i < MaxHeaderCount+1; i++ {
		headers[fmt.Sprintf("h%d", i)] = "v"
	}
	if err := ApplyHeaders(req, headers); err == nil {
		t.Fatal("expected too many headers error")
	}
}

func TestConnectorInputsFromProperties(t *testing.T) {
	props := map[string]any{
		"url":          "https://api.example.com",
		"method":       "GET",
		"unrelated":    "x",
		"emailTo":      "a@b.c",
		"emailSubject": "  ",
		"kafkaTopic":   "orders",
	}
	got := ConnectorInputsFromProperties(props)
	if got["url"] != "https://api.example.com" {
		t.Fatalf("url: %v", got["url"])
	}
	if _, ok := got["unrelated"]; ok {
		t.Fatal("unrelated key should be skipped")
	}
	if _, ok := got["emailSubject"]; ok {
		t.Fatal("empty emailSubject should be skipped")
	}
	if got["kafkaTopic"] != "orders" {
		t.Fatalf("kafkaTopic: %v", got["kafkaTopic"])
	}
}
