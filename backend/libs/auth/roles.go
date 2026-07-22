package auth

import (
	"net/http"
	"strings"
)

const (
	RoleFlowGoAdmin   = "flowgo admin"
	RoleFlowGoModeler = "flowgo modeler"
	RoleFlowGoClient  = "flowgo client"
)

func StandardRoles() []string {
	return []string{RoleFlowGoAdmin, RoleFlowGoModeler, RoleFlowGoClient}
}

func RequireAuthenticated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := PrincipalFromContext(r.Context()); !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
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
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
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
