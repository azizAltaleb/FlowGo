package api

import (
	"encoding/json"
	"net/http"

	"github.com/azizAltaleb/flowgo/backend/libs/auth"
	"github.com/azizAltaleb/flowgo/backend/services/workflow-command/internal/interfaces/http/dto"
)

func (h *Handler) getIdentityConfig(w http.ResponseWriter, _ *http.Request) {
	config := h.identityConfig
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(dto.IdentityConfigResponse{
		DeploymentMode:          config.Mode,
		ConfigurationSource:     config.ConfigurationSource,
		ProviderName:            config.ProviderName,
		AuthEnabled:             config.AuthConfig.Enabled(),
		FrontendAuthEnabled:     config.FrontendConfig.Enabled,
		FrontendOIDCAuthority:   config.FrontendConfig.OIDCAuthority,
		FrontendOIDCClientID:    config.FrontendConfig.OIDCClientID,
		TokenValidationMode:     config.AuthConfig.TokenValidationMode,
		InternalIssuerURL:       config.AuthConfig.InternalIssuerURL,
		ExternalIssuerURL:       config.AuthConfig.ExternalIssuerURL,
		ClientID:                config.AuthConfig.ClientID,
		IntrospectionURL:        config.AuthConfig.IntrospectionURL,
		IntrospectionClientID:   config.AuthConfig.IntrospectionClientID,
		IntrospectionAuthMethod: config.AuthConfig.IntrospectionAuthMethod,
		EnforceAudience:         config.AuthConfig.EnforceAudience,
		AllowInsecureIssuer:     config.AuthConfig.AllowInsecureIssuer,
		ClaimSubjectPath:        config.AuthConfig.ClaimSubjectPath,
		ClaimRolesPath:          config.AuthConfig.ClaimRolesPath,
		ClaimScopesPath:         config.AuthConfig.ClaimScopesPath,
		ClaimTenantPath:         config.AuthConfig.ClaimTenantPath,
		ClaimEmailPath:          config.AuthConfig.ClaimEmailPath,
		ClaimNamePath:           config.AuthConfig.ClaimNamePath,
		StandardRoles:           auth.StandardRoles(),
	})
}
