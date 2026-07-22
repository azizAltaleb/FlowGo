package auth

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveConfigFromEnv_PrefersAuthContract(t *testing.T) {
	t.Setenv("AUTH_ISSUER_INTERNAL_URL", "https://auth-internal.example.com")
	t.Setenv("AUTH_ISSUER_PUBLIC_URL", "https://auth-public.example.com")
	t.Setenv("AUTH_CLIENT_ID", "auth-client")
	t.Setenv("AUTH_TOKEN_MODE", "jwt")
	t.Setenv("OIDC_ISSUER_INTERNAL_URL", "https://oidc-internal.example.com")
	t.Setenv("OIDC_ISSUER_PUBLIC_URL", "https://oidc-public.example.com")
	t.Setenv("OIDC_CLIENT_ID", "oidc-client")

	cfg := ResolveConfigFromEnv()
	if cfg.InternalIssuerURL != "https://auth-internal.example.com" {
		t.Fatalf("expected AUTH_ISSUER_INTERNAL_URL precedence, got %s", cfg.InternalIssuerURL)
	}
	if cfg.ExternalIssuerURL != "https://auth-public.example.com" {
		t.Fatalf("expected AUTH_ISSUER_PUBLIC_URL precedence, got %s", cfg.ExternalIssuerURL)
	}
	if cfg.ClientID != "auth-client" {
		t.Fatalf("expected AUTH_CLIENT_ID precedence, got %s", cfg.ClientID)
	}
	if cfg.TokenValidationMode != TokenModeJWT {
		t.Fatalf("expected jwt mode, got %s", cfg.TokenValidationMode)
	}
}

func TestResolveConfigFromEnv_IgnoresOIDCAliasIssuers(t *testing.T) {
	t.Setenv("OIDC_ISSUER_INTERNAL_URL", "https://oidc-internal.example.com")
	t.Setenv("OIDC_ISSUER_PUBLIC_URL", "https://oidc-public.example.com")
	cfg := ResolveConfigFromEnv()
	if cfg.InternalIssuerURL != "" {
		t.Fatalf("expected empty internal issuer without AUTH_ISSUER_INTERNAL_URL, got %s", cfg.InternalIssuerURL)
	}
	if cfg.ExternalIssuerURL != "" {
		t.Fatalf("expected empty external issuer without AUTH_ISSUER_PUBLIC_URL, got %s", cfg.ExternalIssuerURL)
	}
}

func TestResolveConfigFromEnv_IntrospectionAuthMethod(t *testing.T) {
	t.Setenv("AUTH_INTROSPECTION_AUTH_METHOD", "post")
	cfg := ResolveConfigFromEnv()
	if cfg.IntrospectionAuthMethod != "post" {
		t.Fatalf("expected AUTH_INTROSPECTION_AUTH_METHOD value, got %s", cfg.IntrospectionAuthMethod)
	}
}

func TestResolveConfigFromEnv_IntrospectionClientIDDefaultsToClientID(t *testing.T) {
	t.Setenv("AUTH_CLIENT_ID", "workflow-backend")
	t.Setenv("AUTH_INTROSPECTION_CLIENT_ID", "")
	cfg := ResolveConfigFromEnv()
	if cfg.IntrospectionClientID != "workflow-backend" {
		t.Fatalf("expected AUTH_CLIENT_ID fallback for introspection client id, got %s", cfg.IntrospectionClientID)
	}
}

func TestConfigEnabled_IntrospectionRequiresURL(t *testing.T) {
	cfg := Config{TokenValidationMode: TokenModeIntrospection}
	if cfg.Enabled() {
		t.Fatal("expected introspection mode without URL to be disabled")
	}
	cfg.IntrospectionURL = "https://auth.example.com/oauth2/introspect"
	if !cfg.Enabled() {
		t.Fatal("expected introspection mode with URL to be enabled")
	}
}

func TestResolveConfigFromEnv_LoadsIntrospectionCredentialsFromFiles(t *testing.T) {
	dir := t.TempDir()
	clientIDFile := filepath.Join(dir, "client-id")
	secretFile := filepath.Join(dir, "client-secret")
	if err := os.WriteFile(clientIDFile, []byte("generated-client\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(secretFile, []byte("generated-secret\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AUTH_INTROSPECTION_CLIENT_ID", "")
	t.Setenv("AUTH_INTROSPECTION_CLIENT_SECRET", "")
	t.Setenv("AUTH_INTROSPECTION_CLIENT_ID_FILE", clientIDFile)
	t.Setenv("AUTH_INTROSPECTION_CLIENT_SECRET_FILE", secretFile)

	cfg := ResolveConfigFromEnv()
	if cfg.IntrospectionClientID != "generated-client" {
		t.Fatalf("expected file-backed client id, got %q", cfg.IntrospectionClientID)
	}
	if cfg.IntrospectionClientSecret != "generated-secret" {
		t.Fatalf("expected file-backed client secret, got %q", cfg.IntrospectionClientSecret)
	}
}

func TestResolveConfigFromEnv_LoadsClientIDFromTrustedFile(t *testing.T) {
	clientIDFile := filepath.Join(t.TempDir(), "client-id")
	if err := os.WriteFile(clientIDFile, []byte("generated-project-id\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AUTH_CLIENT_ID", "")
	t.Setenv("AUTH_CLIENT_ID_FILE", clientIDFile)

	cfg := ResolveConfigFromEnv()
	if cfg.ClientID != "generated-project-id" {
		t.Fatalf("expected file-backed client id, got %q", cfg.ClientID)
	}
	if cfg.ClientIDFile != clientIDFile {
		t.Fatalf("expected client id file to be retained, got %q", cfg.ClientIDFile)
	}
}

func TestResolveConfigFromEnv_ExplicitClientIDOverridesFile(t *testing.T) {
	clientIDFile := filepath.Join(t.TempDir(), "client-id")
	if err := os.WriteFile(clientIDFile, []byte("file-project-id"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AUTH_CLIENT_ID", "explicit-audience")
	t.Setenv("AUTH_CLIENT_ID_FILE", clientIDFile)

	if cfg := ResolveConfigFromEnv(); cfg.ClientID != "explicit-audience" {
		t.Fatalf("expected explicit client id to win, got %q", cfg.ClientID)
	}
}

func TestResolveConfigFromEnv_ConfiguredInvalidClientIDFileFailsClosed(t *testing.T) {
	t.Setenv("AUTH_CLIENT_ID", "")
	t.Setenv("AUTH_CLIENT_ID_FILE", "relative/project-id")

	if cfg := ResolveConfigFromEnv(); cfg.ClientID != "" {
		t.Fatalf("expected configured invalid client id file to fail closed, got %q", cfg.ClientID)
	}
}

func TestResolveConfigFromEnv_ClientIDKeepsExternalDefaultWithoutFile(t *testing.T) {
	t.Setenv("AUTH_CLIENT_ID", "")
	t.Setenv("AUTH_CLIENT_ID_FILE", "")

	if cfg := ResolveConfigFromEnv(); cfg.ClientID != "workflow-backend" {
		t.Fatalf("expected legacy client id default, got %q", cfg.ClientID)
	}
}

func TestResolveConfigFromEnv_ExplicitCredentialsOverrideFiles(t *testing.T) {
	dir := t.TempDir()
	clientIDFile := filepath.Join(dir, "client-id")
	secretFile := filepath.Join(dir, "client-secret")
	if err := os.WriteFile(clientIDFile, []byte("file-client"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(secretFile, []byte("file-secret"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AUTH_INTROSPECTION_CLIENT_ID", "env-client")
	t.Setenv("AUTH_INTROSPECTION_CLIENT_SECRET", "env-secret")
	t.Setenv("AUTH_INTROSPECTION_CLIENT_ID_FILE", clientIDFile)
	t.Setenv("AUTH_INTROSPECTION_CLIENT_SECRET_FILE", secretFile)

	cfg := ResolveConfigFromEnv()
	if cfg.IntrospectionClientID != "env-client" || cfg.IntrospectionClientSecret != "env-secret" {
		t.Fatalf("expected explicit credentials to win, got client=%q secret=%q", cfg.IntrospectionClientID, cfg.IntrospectionClientSecret)
	}
}

func TestResolveConfigFromEnv_RejectsRelativeOrMissingCredentialFiles(t *testing.T) {
	t.Setenv("AUTH_INTROSPECTION_CLIENT_ID", "")
	t.Setenv("AUTH_INTROSPECTION_CLIENT_SECRET", "")
	t.Setenv("AUTH_INTROSPECTION_CLIENT_ID_FILE", "relative/client-id")
	t.Setenv("AUTH_INTROSPECTION_CLIENT_SECRET_FILE", filepath.Join(t.TempDir(), "missing"))

	cfg := ResolveConfigFromEnv()
	if cfg.IntrospectionClientID != "" {
		t.Fatalf("expected invalid client id file to fail closed, got %q", cfg.IntrospectionClientID)
	}
	if cfg.IntrospectionClientSecret != "" {
		t.Fatal("expected missing secret file to remain empty")
	}
}
