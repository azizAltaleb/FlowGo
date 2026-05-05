package auth

import (
	"os"
	"strings"
)

const (
	TokenModeJWT           = "jwt"
	TokenModeIntrospection = "introspection"
)

type Config struct {
	InternalIssuerURL string
	ExternalIssuerURL string
	ClientID          string

	TokenValidationMode       string
	IntrospectionURL          string
	IntrospectionClientID     string
	IntrospectionClientSecret string
	IntrospectionAuthMethod   string

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
	cfg := Config{
		ClientID:                  strings.TrimSpace(firstNonEmpty(os.Getenv("AUTH_CLIENT_ID"), "workflow-backend")),
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
	}

	cfg.InternalIssuerURL = strings.TrimSpace(os.Getenv("AUTH_ISSUER_INTERNAL_URL"))
	cfg.ExternalIssuerURL = strings.TrimSpace(os.Getenv("AUTH_ISSUER_PUBLIC_URL"))

	if cfg.InternalIssuerURL == "" {
		cfg.InternalIssuerURL = cfg.ExternalIssuerURL
	}
	if cfg.ExternalIssuerURL == "" {
		cfg.ExternalIssuerURL = cfg.InternalIssuerURL
	}

	if cfg.IntrospectionClientID == "" {
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
