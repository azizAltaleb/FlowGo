package auth

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	TokenModeJWT           = "jwt"
	TokenModeIntrospection = "introspection"
)

type Config struct {
	InternalIssuerURL string
	ExternalIssuerURL string
	ClientID          string
	ClientIDFile      string

	TokenValidationMode       string
	IntrospectionURL          string
	IntrospectionClientID     string
	IntrospectionClientSecret string
	IntrospectionAuthMethod   string
	IntrospectionClientIDFile string
	IntrospectionSecretFile   string

	EnforceAudience     bool
	AllowInsecureIssuer bool

	ClaimSubjectPath string
	ClaimRolesPath   string
	ClaimScopesPath  string
	ClaimTenantPath  string
	ClaimEmailPath   string
	ClaimNamePath    string
}

func (c Config) Enabled() bool {
	switch c.TokenValidationMode {
	case TokenModeIntrospection:
		return strings.TrimSpace(c.IntrospectionURL) != ""
	default:
		return strings.TrimSpace(c.InternalIssuerURL) != ""
	}
}

func ResolveConfigFromEnv() Config {
	clientID := strings.TrimSpace(os.Getenv("AUTH_CLIENT_ID"))
	clientIDFile := strings.TrimSpace(os.Getenv("AUTH_CLIENT_ID_FILE"))
	cfg := Config{
		ClientID:                  clientID,
		ClientIDFile:              clientIDFile,
		TokenValidationMode:       normalizeTokenMode(firstNonEmpty(os.Getenv("AUTH_TOKEN_MODE"), TokenModeJWT)),
		IntrospectionAuthMethod:   normalizeIntrospectionAuthMethod(firstNonEmpty(os.Getenv("AUTH_INTROSPECTION_AUTH_METHOD"), "basic")),
		EnforceAudience:           envBoolOrDefault("AUTH_ENFORCE_AUDIENCE", true),
		AllowInsecureIssuer:       envBoolOrDefault("AUTH_ALLOW_INSECURE_ISSUER", false),
		ClaimSubjectPath:          strings.TrimSpace(firstNonEmpty(os.Getenv("AUTH_CLAIM_SUBJECT_PATH"), "sub")),
		ClaimRolesPath:            strings.TrimSpace(firstNonEmpty(os.Getenv("AUTH_CLAIM_ROLES_PATH"), "roles,realm_access.roles,groups")),
		ClaimScopesPath:           strings.TrimSpace(firstNonEmpty(os.Getenv("AUTH_CLAIM_SCOPES_PATH"), "scope,scp")),
		ClaimTenantPath:           strings.TrimSpace(firstNonEmpty(os.Getenv("AUTH_CLAIM_TENANT_PATH"), "tenant_id")),
		ClaimEmailPath:            strings.TrimSpace(firstNonEmpty(os.Getenv("AUTH_CLAIM_EMAIL_PATH"), "email")),
		ClaimNamePath:             strings.TrimSpace(firstNonEmpty(os.Getenv("AUTH_CLAIM_NAME_PATH"), "name")),
		IntrospectionURL:          strings.TrimSpace(os.Getenv("AUTH_INTROSPECTION_URL")),
		IntrospectionClientID:     strings.TrimSpace(os.Getenv("AUTH_INTROSPECTION_CLIENT_ID")),
		IntrospectionClientSecret: strings.TrimSpace(os.Getenv("AUTH_INTROSPECTION_CLIENT_SECRET")),
		IntrospectionClientIDFile: strings.TrimSpace(os.Getenv("AUTH_INTROSPECTION_CLIENT_ID_FILE")),
		IntrospectionSecretFile:   strings.TrimSpace(os.Getenv("AUTH_INTROSPECTION_CLIENT_SECRET_FILE")),
	}

	fileTimeout := envDurationSeconds("AUTH_INTROSPECTION_CREDENTIAL_FILE_TIMEOUT_SECONDS")
	clientIDFileTimeout := envDurationSeconds("AUTH_CLIENT_ID_FILE_TIMEOUT_SECONDS")
	if clientIDFileTimeout == 0 {
		clientIDFileTimeout = fileTimeout
	}
	if cfg.ClientID == "" && cfg.ClientIDFile != "" {
		cfg.ClientID = readDeploymentCredentialFile(cfg.ClientIDFile, clientIDFileTimeout)
	}
	if cfg.ClientID == "" && cfg.ClientIDFile == "" {
		cfg.ClientID = "workflow-backend"
	}
	if cfg.IntrospectionClientID == "" && cfg.IntrospectionClientIDFile != "" {
		cfg.IntrospectionClientID = readDeploymentCredentialFile(cfg.IntrospectionClientIDFile, fileTimeout)
	}
	if cfg.IntrospectionClientSecret == "" && cfg.IntrospectionSecretFile != "" {
		cfg.IntrospectionClientSecret = readDeploymentCredentialFile(cfg.IntrospectionSecretFile, fileTimeout)
	}

	cfg.InternalIssuerURL = strings.TrimSpace(os.Getenv("AUTH_ISSUER_INTERNAL_URL"))
	cfg.ExternalIssuerURL = strings.TrimSpace(os.Getenv("AUTH_ISSUER_PUBLIC_URL"))

	if cfg.InternalIssuerURL == "" {
		cfg.InternalIssuerURL = cfg.ExternalIssuerURL
	}
	if cfg.ExternalIssuerURL == "" {
		cfg.ExternalIssuerURL = cfg.InternalIssuerURL
	}

	if cfg.IntrospectionClientID == "" && cfg.IntrospectionClientIDFile == "" {
		cfg.IntrospectionClientID = cfg.ClientID
	}

	return cfg
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func normalizeTokenMode(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case TokenModeIntrospection, "opaque":
		return TokenModeIntrospection
	default:
		return TokenModeJWT
	}
}

func normalizeIntrospectionAuthMethod(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "post":
		return "post"
	default:
		return "basic"
	}
}

func envBoolOrDefault(key string, defaultValue bool) bool {
	val := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	switch val {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return defaultValue
	}
}

func envDurationSeconds(key string) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return 0
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds <= 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}

func readDeploymentCredentialFile(rawPath string, timeout time.Duration) string {
	path := strings.TrimSpace(rawPath)
	if path == "" || !filepath.IsAbs(path) {
		return ""
	}
	path = filepath.Clean(path)
	deadline := time.Now().Add(timeout)
	for {
		// #nosec G304 G703 -- path is deployment-controlled and constrained to a cleaned absolute path.
		content, err := os.ReadFile(path)
		if err == nil {
			if value := strings.TrimSpace(string(content)); value != "" {
				return value
			}
		}
		if timeout <= 0 || !time.Now().Before(deadline) {
			return ""
		}
		time.Sleep(250 * time.Millisecond)
	}
}
