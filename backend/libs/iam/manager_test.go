package iam

import (
	"os"
	"testing"
)

func TestResolveFrontendAuthConfigReadsClientIDFile(t *testing.T) {
	clientIDFile := t.TempDir() + "/client-id"
	if err := os.WriteFile(clientIDFile, []byte(" generated-client-id \n"), 0o600); err != nil {
		t.Fatalf("write client id file: %v", err)
	}

	t.Setenv("FRONTEND_AUTH_OIDC_AUTHORITY", "http://localhost:9180")
	t.Setenv("FRONTEND_AUTH_OIDC_CLIENT_ID", "")
	t.Setenv("VITE_OIDC_CLIENT_ID", "")
	t.Setenv("FRONTEND_AUTH_OIDC_CLIENT_ID_FILE", clientIDFile)

	cfg := ResolveFrontendAuthConfigFromEnv()
	if !cfg.Enabled {
		t.Fatal("expected frontend auth to be enabled")
	}
	if cfg.OIDCClientID != "generated-client-id" {
		t.Fatalf("expected generated client id, got %q", cfg.OIDCClientID)
	}
}

func TestResolveFrontendAuthConfigEnvClientIDWinsOverFile(t *testing.T) {
	clientIDFile := t.TempDir() + "/client-id"
	if err := os.WriteFile(clientIDFile, []byte("generated-client-id"), 0o600); err != nil {
		t.Fatalf("write client id file: %v", err)
	}

	t.Setenv("FRONTEND_AUTH_OIDC_AUTHORITY", "http://localhost:9180")
	t.Setenv("FRONTEND_AUTH_OIDC_CLIENT_ID", "explicit-client-id")
	t.Setenv("FRONTEND_AUTH_OIDC_CLIENT_ID_FILE", clientIDFile)

	cfg := ResolveFrontendAuthConfigFromEnv()
	if cfg.OIDCClientID != "explicit-client-id" {
		t.Fatalf("expected explicit client id, got %q", cfg.OIDCClientID)
	}
}
