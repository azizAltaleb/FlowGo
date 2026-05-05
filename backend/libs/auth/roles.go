package auth

import (
	"net/http"
	"strings"
)

const (
	RoleWorkflowsaClient = "workflowsa client"
	RoleWorkflowsaAdmin  = "workflowsa admin"
	RoleWorkflowsaViewer = "workflowsa viewer"
)

func StandardRoles() []string {
	return []string{RoleWorkflowsaClient, RoleWorkflowsaAdmin, RoleWorkflowsaViewer}
}

func (p Principal) HasRole(role string) bool {
	required := strings.TrimSpace(role)
	if required == "" {
		return false
	}
	for _, candidate := range p.Roles {
		if strings.EqualFold(strings.TrimSpace(candidate), required) {
			return true
		}
	}
	return false
}

func (p Principal) HasAnyRole(roles ...string) bool {
	for _, role := range roles {
		if p.HasRole(role) {
			return true
		}
	}
	return false
}

func RequireAnyRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, ok := PrincipalFromContext(r.Context())
			if !ok {
				next.ServeHTTP(w, r)
				return
			}
			if principal.HasAnyRole(roles...) {
				next.ServeHTTP(w, r)
				return
			}
			http.Error(w, "Forbidden", http.StatusForbidden)
		})
	}
}
