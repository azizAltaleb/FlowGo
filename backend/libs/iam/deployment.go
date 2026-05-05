package iam

import (
	"os"
	"strings"

	"workflow-engine/backend/libs/auth"
)

const (
	DeploymentModeDisabled = "disabled"
	DeploymentModeExternal = "external"
	DeploymentModeZITADEL  = "zitadel"
)

type DeploymentConfig struct {
	Mode                string
	ProviderName        string
	ConfigurationSource string
	AuthConfig          auth.Config
	FrontendConfig      FrontendAuthConfig
	ZITADELManagement   ZITADELManagementConfig
}

func ResolveDeploymentConfigFromEnv() DeploymentConfig {
	authConfig := auth.ResolveConfigFromEnv()
	frontendConfig := ResolveFrontendAuthConfigFromEnv()
	mode := normalizeDeploymentMode(os.Getenv("IAM_DEPLOYMENT_MODE"))
	if mode == "" {
		switch {
		case authConfig.Enabled() || frontendConfig.Enabled:
			mode = DeploymentModeExternal
		default:
			mode = DeploymentModeDisabled
		}
	}
	providerName := strings.TrimSpace(os.Getenv("IAM_PROVIDER_NAME"))
	if providerName == "" {
		switch mode {
		case DeploymentModeZITADEL:
			providerName = "ZITADEL"
		case DeploymentModeExternal:
			providerName = "External OIDC Provider"
		default:
			providerName = "Disabled"
		}
	}
	return DeploymentConfig{
		Mode:                mode,
		ProviderName:        providerName,
		ConfigurationSource: "docker-compose",
		AuthConfig:          authConfig,
		FrontendConfig:      frontendConfig,
		ZITADELManagement:   ResolveZITADELManagementConfigFromEnv(authConfig, frontendConfig),
	}
}

func normalizeDeploymentMode(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "external", "enterprise":
		return DeploymentModeExternal
	case "zitadel", "bundled", "internal", "solution":
		return DeploymentModeZITADEL
	case "disabled", "off", "none":
		return DeploymentModeDisabled
	default:
		return ""
	}
}
