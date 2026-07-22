package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStandardRoles(t *testing.T) {
	roles := StandardRoles()
	expected := []string{RoleFlowGoAdmin, RoleFlowGoModeler, RoleFlowGoClient}
	if len(roles) != len(expected) {
		t.Fatalf("expected %d standard roles, got %d", len(expected), len(roles))
	}
	for i, role := range expected {
		if roles[i] != role {
			t.Fatalf("expected role %q at index %d, got %q", role, i, roles[i])
		}
	}
}

func TestPrincipalHasRole(t *testing.T) {
	principal := Principal{Roles: []string{" FlowGo Admin "}}
	if !principal.HasRole(RoleFlowGoAdmin) {
		t.Fatal("expected case-insensitive role match")
	}
	if principal.HasRole(RoleFlowGoClient) {
		t.Fatal("did not expect client role match")
	}
	if principal.HasRole(RoleFlowGoModeler) {
		t.Fatal("did not expect modeler role match")
	}
}

func TestRequireAnyRole(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	handler := RequireAnyRole(RoleFlowGoAdmin)(next)

	allowedReq := httptest.NewRequest(http.MethodGet, "/", nil)
	allowedReq = allowedReq.WithContext(WithPrincipal(allowedReq.Context(), Principal{Roles: []string{RoleFlowGoAdmin}}))
	allowedResp := httptest.NewRecorder()
	handler.ServeHTTP(allowedResp, allowedReq)
	if allowedResp.Code != http.StatusNoContent {
		t.Fatalf("expected allowed status %d, got %d", http.StatusNoContent, allowedResp.Code)
	}

	deniedReq := httptest.NewRequest(http.MethodGet, "/", nil)
	deniedReq = deniedReq.WithContext(WithPrincipal(deniedReq.Context(), Principal{Roles: []string{RoleFlowGoClient}}))
	deniedResp := httptest.NewRecorder()
	handler.ServeHTTP(deniedResp, deniedReq)
	if deniedResp.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden status %d, got %d", http.StatusForbidden, deniedResp.Code)
	}

	missingPrincipalReq := httptest.NewRequest(http.MethodGet, "/", nil)
	missingPrincipalResp := httptest.NewRecorder()
	handler.ServeHTTP(missingPrincipalResp, missingPrincipalReq)
	if missingPrincipalResp.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized status %d without principal, got %d", http.StatusUnauthorized, missingPrincipalResp.Code)
	}
}

func TestDisabledAuthPrincipalHasStandardRoles(t *testing.T) {
	principal := disabledAuthPrincipal()
	if principal.Subject != "local-disabled-auth" {
		t.Fatalf("unexpected disabled-auth subject %q", principal.Subject)
	}
	if !principal.HasRole(RoleFlowGoAdmin) {
		t.Fatal("expected disabled-auth principal to have admin role")
	}
	if !principal.HasRole(RoleFlowGoModeler) {
		t.Fatal("expected disabled-auth principal to have modeler role")
	}
	if !principal.HasRole(RoleFlowGoClient) {
		t.Fatal("expected disabled-auth principal to have SDK client role")
	}
}

func TestMiddlewareDisabledAuthAllowsRoleProtectedHTTPRoute(t *testing.T) {
	middleware, err := NewMiddleware(context.Background(), Config{TokenValidationMode: TokenModeJWT})
	if err != nil {
		t.Fatalf("new middleware: %v", err)
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, ok := PrincipalFromContext(r.Context())
		if !ok {
			t.Fatal("expected disabled-auth principal in request context")
		}
		if !principal.HasRole(RoleFlowGoAdmin) {
			t.Fatal("expected disabled-auth principal to have admin role")
		}
		w.WriteHeader(http.StatusNoContent)
	})

	handler := middleware.Handler(RequireAnyRole(RoleFlowGoAdmin)(next))
	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusNoContent {
		t.Fatalf("expected allowed status %d, got %d", http.StatusNoContent, resp.Code)
	}
}
