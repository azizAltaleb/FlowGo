package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/azizAltaleb/goflow/backend/libs/auth"
	"github.com/azizAltaleb/goflow/backend/libs/iam"
	"github.com/azizAltaleb/goflow/backend/services/workflow-command/internal/interfaces/http/dto"

	"github.com/gorilla/mux"
)

func (h *Handler) registerIdentityManagementRoutes(r *mux.Router) {
	r.Handle("/identity/management/clients", h.requireBundledIAMAdmin(h.listIdentityManagementClients)).Methods("GET")
	r.Handle("/identity/management/clients", h.requireBundledIAMAdmin(h.createIdentityManagementClientToken)).Methods("POST")
	r.Handle("/identity/management/clients/{id}", h.requireBundledIAMAdmin(h.deleteIdentityManagementClient)).Methods("DELETE")
	r.Handle("/identity/management/clients/{id}/tokens", h.requireBundledIAMAdmin(h.rotateIdentityManagementClientToken)).Methods("POST")
	r.Handle("/identity/management/clients/{id}/tokens/{tokenId}", h.requireBundledIAMAdmin(h.revokeIdentityManagementClientToken)).Methods("DELETE")
	r.Handle("/identity/management/users", h.requireBundledIAMAdmin(h.listIdentityManagementUsers)).Methods("GET")
	r.Handle("/identity/management/users", h.requireBundledIAMAdmin(h.createIdentityManagementUser)).Methods("POST")
	r.Handle("/identity/management/users/{id}", h.requireBundledIAMAdmin(h.updateIdentityManagementUser)).Methods("PUT")
	r.Handle("/identity/management/users/{id}", h.requireBundledIAMAdmin(h.deleteIdentityManagementUser)).Methods("DELETE")
	r.Handle("/identity/management/users/{id}/terminate", h.requireBundledIAMAdmin(h.terminateIdentityManagementUser)).Methods("POST")
	r.Handle("/identity/management/users/{id}/reactivate", h.requireBundledIAMAdmin(h.reactivateIdentityManagementUser)).Methods("POST")
	r.Handle("/identity/management/roles", h.requireBundledIAMAdmin(h.listIdentityManagementRoles)).Methods("GET")
	r.Handle("/identity/management/roles", h.requireBundledIAMAdmin(h.createIdentityManagementRole)).Methods("POST")
	r.Handle("/identity/management/roles/{roleKey}", h.requireBundledIAMAdmin(h.updateIdentityManagementRole)).Methods("PUT")
	r.Handle("/identity/management/roles/{roleKey}", h.requireBundledIAMAdmin(h.deleteIdentityManagementRole)).Methods("DELETE")
}

func (h *Handler) requireBundledIAMAdmin(fn http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.identityConfig.Mode != iam.DeploymentModeZITADEL {
			http.NotFound(w, r)
			return
		}
		principal, ok := auth.PrincipalFromContext(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if !principal.HasRole(auth.RoleGoFlowAdmin) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		fn(w, r)
	})
}

func (h *Handler) listIdentityManagementUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.identityManager.ListUsers(r.Context())
	if err != nil {
		writeIdentityManagementError(w, err)
		return
	}
	response := dto.ListIdentityManagementUsersResponse{Users: make([]dto.IdentityManagementUserResponse, 0, len(users))}
	for _, user := range users {
		response.Users = append(response.Users, toIdentityManagementUserResponse(user))
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) listIdentityManagementClients(w http.ResponseWriter, r *http.Request) {
	clients, err := h.identityManager.ListClients(r.Context())
	if err != nil {
		writeIdentityManagementError(w, err)
		return
	}
	response := dto.ListIdentityManagementClientsResponse{Clients: make([]dto.IdentityManagementClientResponse, 0, len(clients))}
	for _, client := range clients {
		response.Clients = append(response.Clients, toIdentityManagementClientResponse(client))
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) createIdentityManagementUser(w http.ResponseWriter, r *http.Request) {
	var request dto.CreateIdentityManagementUserRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(request.Email) == "" || strings.TrimSpace(request.GivenName) == "" || strings.TrimSpace(request.FamilyName) == "" || strings.TrimSpace(request.Password) == "" {
		http.Error(w, "email, given_name, family_name, and password are required", http.StatusBadRequest)
		return
	}
	user, err := h.identityManager.CreateUser(r.Context(), iam.ManagedUserCreate{
		Username:               request.Username,
		GivenName:              request.GivenName,
		FamilyName:             request.FamilyName,
		Email:                  request.Email,
		Password:               request.Password,
		PasswordChangeRequired: request.PasswordChangeRequired,
		RoleKeys:               request.Roles,
	})
	if err != nil {
		writeIdentityManagementError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toIdentityManagementUserResponse(user))
}

func (h *Handler) createIdentityManagementClientToken(w http.ResponseWriter, r *http.Request) {
	var request dto.CreateIdentityManagementClientTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(request.Name) == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	token, err := h.identityManager.CreateClientToken(r.Context(), iam.ManagedClientTokenCreate{
		Username:       request.Username,
		Name:           request.Name,
		Description:    request.Description,
		Environment:    request.Environment,
		OwnerEmail:     request.OwnerEmail,
		Purpose:        request.Purpose,
		TokenExpiresAt: request.TokenExpiresAt,
	})
	if err != nil {
		writeIdentityManagementError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toIdentityManagementClientTokenResponse(token))
}

func (h *Handler) rotateIdentityManagementClientToken(w http.ResponseWriter, r *http.Request) {
	var request dto.RotateIdentityManagementClientTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	token, err := h.identityManager.RotateClientToken(r.Context(), mux.Vars(r)["id"], iam.ManagedClientTokenRotate{
		TokenExpiresAt: request.TokenExpiresAt,
	})
	if err != nil {
		writeIdentityManagementError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toIdentityManagementClientTokenResponse(token))
}

func (h *Handler) revokeIdentityManagementClientToken(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if err := h.identityManager.RevokeClientToken(r.Context(), vars["id"], vars["tokenId"]); err != nil {
		writeIdentityManagementError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) deleteIdentityManagementClient(w http.ResponseWriter, r *http.Request) {
	if err := h.identityManager.DeleteClient(r.Context(), mux.Vars(r)["id"]); err != nil {
		writeIdentityManagementError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) updateIdentityManagementUser(w http.ResponseWriter, r *http.Request) {
	var request dto.UpdateIdentityManagementUserRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	input := iam.ManagedUserUpdate{
		Username:    request.Username,
		GivenName:   request.GivenName,
		FamilyName:  request.FamilyName,
		DisplayName: request.DisplayName,
		Email:       request.Email,
	}
	if request.Roles != nil {
		input.RoleKeys = *request.Roles
		input.UpdateRoleKeys = true
	}
	user, err := h.identityManager.UpdateUser(r.Context(), mux.Vars(r)["id"], input)
	if err != nil {
		writeIdentityManagementError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toIdentityManagementUserResponse(user))
}

func (h *Handler) terminateIdentityManagementUser(w http.ResponseWriter, r *http.Request) {
	if err := h.identityManager.TerminateUser(r.Context(), mux.Vars(r)["id"]); err != nil {
		writeIdentityManagementError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) reactivateIdentityManagementUser(w http.ResponseWriter, r *http.Request) {
	if err := h.identityManager.ReactivateUser(r.Context(), mux.Vars(r)["id"]); err != nil {
		writeIdentityManagementError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) deleteIdentityManagementUser(w http.ResponseWriter, r *http.Request) {
	if err := h.identityManager.DeleteUser(r.Context(), mux.Vars(r)["id"]); err != nil {
		writeIdentityManagementError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listIdentityManagementRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := h.identityManager.ListRoles(r.Context())
	if err != nil {
		writeIdentityManagementError(w, err)
		return
	}
	response := dto.ListIdentityManagementRolesResponse{Roles: make([]dto.IdentityManagementRoleResponse, 0, len(roles))}
	for _, role := range roles {
		response.Roles = append(response.Roles, toIdentityManagementRoleResponse(role))
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) createIdentityManagementRole(w http.ResponseWriter, r *http.Request) {
	var request dto.CreateIdentityManagementRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(request.Key) == "" || strings.TrimSpace(request.DisplayName) == "" {
		http.Error(w, "key and display_name are required", http.StatusBadRequest)
		return
	}
	role, err := h.identityManager.CreateRole(r.Context(), iam.ManagedRoleCreate{Key: request.Key, DisplayName: request.DisplayName, Group: request.Group})
	if err != nil {
		writeIdentityManagementError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toIdentityManagementRoleResponse(role))
}

func (h *Handler) updateIdentityManagementRole(w http.ResponseWriter, r *http.Request) {
	var request dto.UpdateIdentityManagementRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	role, err := h.identityManager.UpdateRole(r.Context(), mux.Vars(r)["roleKey"], iam.ManagedRoleUpdate{DisplayName: request.DisplayName, Group: request.Group})
	if err != nil {
		writeIdentityManagementError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toIdentityManagementRoleResponse(role))
}

func (h *Handler) deleteIdentityManagementRole(w http.ResponseWriter, r *http.Request) {
	if err := h.identityManager.DeleteRole(r.Context(), mux.Vars(r)["roleKey"]); err != nil {
		writeIdentityManagementError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func toIdentityManagementUserResponse(user iam.ManagedUser) dto.IdentityManagementUserResponse {
	return dto.IdentityManagementUserResponse{
		ID:                 user.ID,
		Username:           user.Username,
		PreferredLoginName: user.PreferredLoginName,
		DisplayName:        user.DisplayName,
		GivenName:          user.GivenName,
		FamilyName:         user.FamilyName,
		Email:              user.Email,
		EmailVerified:      user.EmailVerified,
		State:              user.State,
		Type:               user.Type,
		CreatedAt:          user.CreatedAt,
		ChangedAt:          user.ChangedAt,
		Roles:              user.Roles,
	}
}

func toIdentityManagementRoleResponse(role iam.ManagedRole) dto.IdentityManagementRoleResponse {
	return dto.IdentityManagementRoleResponse{Key: role.Key, DisplayName: role.DisplayName, Group: role.Group}
}

func toIdentityManagementClientResponse(client iam.ManagedClient) dto.IdentityManagementClientResponse {
	response := dto.IdentityManagementClientResponse{
		ClientID:    client.ClientID,
		Username:    client.Username,
		Name:        client.Name,
		Description: client.Description,
		Environment: client.Environment,
		OwnerEmail:  client.OwnerEmail,
		Purpose:     client.Purpose,
		Role:        client.Role,
		State:       client.State,
		CreatedAt:   client.CreatedAt,
		ChangedAt:   client.ChangedAt,
		Tokens:      make([]dto.IdentityManagementClientTokenSummaryResponse, 0, len(client.Tokens)),
	}
	for _, token := range client.Tokens {
		response.Tokens = append(response.Tokens, dto.IdentityManagementClientTokenSummaryResponse{
			TokenID:        token.TokenID,
			TokenCreatedAt: token.TokenCreatedAt,
			TokenChangedAt: token.TokenChangedAt,
			TokenExpiresAt: token.TokenExpiresAt,
			Status:         token.Status,
		})
	}
	return response
}

func toIdentityManagementClientTokenResponse(token iam.ManagedClientToken) dto.IdentityManagementClientTokenResponse {
	return dto.IdentityManagementClientTokenResponse{
		ClientID:       token.ClientID,
		Username:       token.Username,
		Name:           token.Name,
		Description:    token.Description,
		Environment:    token.Environment,
		OwnerEmail:     token.OwnerEmail,
		Purpose:        token.Purpose,
		Role:           token.Role,
		TokenID:        token.TokenID,
		Token:          token.Token,
		TokenCreatedAt: token.TokenCreatedAt,
		TokenExpiresAt: token.TokenExpiresAt,
	}
}

func writeIdentityManagementError(w http.ResponseWriter, err error) {
	var zitadelErr *iam.ZITADELError
	switch {
	case errors.Is(err, iam.ErrZITADELManagementNotConfigured):
		http.Error(w, "ZITADEL management is not configured", http.StatusServiceUnavailable)
	case errors.Is(err, iam.ErrZITADELManagedClientNotFound):
		http.Error(w, "GoFlow client was not found", http.StatusNotFound)
	case errors.As(err, &zitadelErr):
		status := http.StatusBadGateway
		if zitadelErr.StatusCode == http.StatusBadRequest || zitadelErr.StatusCode == http.StatusNotFound || zitadelErr.StatusCode == http.StatusConflict {
			status = zitadelErr.StatusCode
		}
		http.Error(w, err.Error(), status)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
