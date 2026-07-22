package iam

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/azizAltaleb/flowgo/backend/libs/auth"
)

func TestResolveDeploymentConfigFromEnv_ExternalIAMRetainsJWTMode(t *testing.T) {
	t.Setenv("IAM_DEPLOYMENT_MODE", "external")
	t.Setenv("AUTH_ISSUER_INTERNAL_URL", "https://identity.example.com")
	t.Setenv("AUTH_ISSUER_PUBLIC_URL", "https://identity.example.com")
	t.Setenv("AUTH_TOKEN_MODE", "jwt")
	t.Setenv("FRONTEND_AUTH_OIDC_AUTHORITY", "https://identity.example.com")
	t.Setenv("FRONTEND_AUTH_OIDC_CLIENT_ID", "flowgo-frontend")

	cfg := ResolveDeploymentConfigFromEnv()
	if cfg.Mode != DeploymentModeExternal {
		t.Fatalf("expected external mode, got %q", cfg.Mode)
	}
	if cfg.AuthConfig.TokenValidationMode != auth.TokenModeJWT {
		t.Fatalf("expected external IAM JWT behavior to remain unchanged, got %q", cfg.AuthConfig.TokenValidationMode)
	}
}

func TestResolveZITADELManagementConfigLegacyPATControlsAreExplicit(t *testing.T) {
	t.Setenv("FLOWGO_IAM_ENABLE_LEGACY_PAT_CREATION", "")
	t.Setenv("FLOWGO_IAM_ENABLE_LEGACY_PAT_ROTATION", "")
	config := ResolveZITADELManagementConfigFromEnv(auth.Config{}, FrontendAuthConfig{})
	if config.EnableLegacyPATCreation || config.EnableLegacyPATRotation {
		t.Fatal("legacy PAT issuance must be disabled by default")
	}

	t.Setenv("FLOWGO_IAM_ENABLE_LEGACY_PAT_CREATION", "true")
	t.Setenv("FLOWGO_IAM_ENABLE_LEGACY_PAT_ROTATION", "1")
	config = ResolveZITADELManagementConfigFromEnv(auth.Config{}, FrontendAuthConfig{})
	if !config.EnableLegacyPATCreation || !config.EnableLegacyPATRotation {
		t.Fatal("explicit emergency controls must enable legacy PAT issuance")
	}
}

func TestResolveZITADELManagementConfigBoundsClientKeyLifetimes(t *testing.T) {
	t.Setenv("FLOWGO_IAM_CLIENT_KEY_DEFAULT_LIFETIME", "48h")
	t.Setenv("FLOWGO_IAM_CLIENT_KEY_MAX_LIFETIME", "720h")
	config := ResolveZITADELManagementConfigFromEnv(auth.Config{}, FrontendAuthConfig{})
	if config.ClientKeyDefaultLifetime != 48*time.Hour || config.ClientKeyMaxLifetime != 720*time.Hour {
		t.Fatalf("unexpected key lifetimes: %#v", config)
	}

	t.Setenv("FLOWGO_IAM_CLIENT_KEY_DEFAULT_LIFETIME", "999999h")
	t.Setenv("FLOWGO_IAM_CLIENT_KEY_MAX_LIFETIME", "invalid")
	config = ResolveZITADELManagementConfigFromEnv(auth.Config{}, FrontendAuthConfig{})
	if config.ClientKeyDefaultLifetime != defaultClientCredentialLifetime || config.ClientKeyMaxLifetime != maxClientCredentialLifetime {
		t.Fatalf("invalid key lifetime settings must fall back to safe bounds: %#v", config)
	}
}

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

func TestReadTrustedConfigFileRejectsRelativePath(t *testing.T) {
	if _, err := readTrustedConfigFile("relative/client-id"); err == nil {
		t.Fatal("expected relative trusted config path to be rejected")
	}
}

func TestReadTrustedConfigFileCleansAbsolutePath(t *testing.T) {
	dir := t.TempDir()
	clientIDFile := filepath.Join(dir, "nested", "..", "client-id")
	if err := os.WriteFile(filepath.Clean(clientIDFile), []byte("client-id"), 0o600); err != nil {
		t.Fatalf("write trusted config file: %v", err)
	}

	content, err := readTrustedConfigFile(clientIDFile)
	if err != nil {
		t.Fatalf("read trusted config file: %v", err)
	}
	if string(content) != "client-id" {
		t.Fatalf("expected trusted config content, got %q", string(content))
	}
}
