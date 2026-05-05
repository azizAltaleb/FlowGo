package iam

import (
	"os"
	"strings"
)

type FrontendAuthConfig struct {
	Enabled       bool   `json:"enabled"`
	OIDCAuthority string `json:"oidc_authority"`
	OIDCClientID  string `json:"oidc_client_id"`
}

func ResolveFrontendAuthConfigFromEnv() FrontendAuthConfig {
	authority := strings.TrimSpace(firstNonEmpty(os.Getenv("FRONTEND_AUTH_OIDC_AUTHORITY"), os.Getenv("VITE_OIDC_AUTHORITY")))
	clientID := strings.TrimSpace(firstNonEmpty(os.Getenv("FRONTEND_AUTH_OIDC_CLIENT_ID"), os.Getenv("VITE_OIDC_CLIENT_ID")))
	if clientID == "" {
		clientID = readFrontendClientIDFile(os.Getenv("FRONTEND_AUTH_OIDC_CLIENT_ID_FILE"))
	}
	return FrontendAuthConfig{
		Enabled:       authority != "" && clientID != "",
		OIDCAuthority: authority,
		OIDCClientID:  clientID,
	}
}

func readFrontendClientIDFile(path string) string {
	trimmedPath := strings.TrimSpace(path)
	if trimmedPath == "" {
		return ""
	}
	content, err := os.ReadFile(trimmedPath)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(content))
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
