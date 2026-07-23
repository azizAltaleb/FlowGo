package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStandardRoles(t *testing.T) {
	roles := StandardRoles()
	expected := []string{RoleArtificialFlowAdmin, RoleArtificialFlowModeler, RoleArtificialFlowClient}
	if len(roles) != len(expected) {
		t.Fatalf("expected %d standard roles, got %d", len(expected), len(roles))
	}
	for i, role := range expected {
		if roles[i] != role {
			t.Fatalf("expected role %q at index %d, got %q", role, i, roles[i])
		}
	}
}

func TestDeprecatedRoleAliasesRemainCanonical(t *testing.T) {
	if RoleFlowGoAdmin != RoleArtificialFlowAdmin ||
		RoleFlowGoModeler != RoleArtificialFlowModeler ||
		RoleFlowGoClient != RoleArtificialFlowClient {
		t.Fatal("deprecated role aliases must emit canonical role values")
	}
}

func TestPrincipalHasRole(t *testing.T) {
	principal := Principal{Roles: []string{" FlowGo Admin "}}
	if !principal.HasRole(RoleArtificialFlowAdmin) {
		t.Fatal("expected legacy admin role to authorize as canonical admin")
	}
	if principal.HasRole(RoleArtificialFlowClient) {
		t.Fatal("did not expect client role match")
	}
	if principal.HasRole(RoleArtificialFlowModeler) {
		t.Fatal("did not expect modeler role match")
	}
}

func TestCanonicalizeRolesMigratesLegacyRolesAndPreservesCustomRoles(t *testing.T) {
	roles := CanonicalizeRoles([]string{
		" FlowGo Admin ",
		"ARTIFICIALFLOW ADMIN",
		"flowgo modeler",
		RoleArtificialFlowClient,
		"Finance Reviewer",
		" finance reviewer ",
	})
	expected := []string{
		RoleArtificialFlowAdmin,
		RoleArtificialFlowModeler,
		RoleArtificialFlowClient,
		"Finance Reviewer",
	}
	if len(roles) != len(expected) {
		t.Fatalf("expected canonical roles %#v, got %#v", expected, roles)
	}
	for index := range expected {
		if roles[index] != expected[index] {
			t.Fatalf("expected role %q at index %d, got %q", expected[index], index, roles[index])
		}
	}
}

func TestRequireAnyRole(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	handler := RequireAnyRole(RoleArtificialFlowAdmin)(next)

	allowedReq := httptest.NewRequest(http.MethodGet, "/", nil)
	allowedReq = allowedReq.WithContext(WithPrincipal(allowedReq.Context(), Principal{Roles: []string{RoleArtificialFlowAdmin}}))
	allowedResp := httptest.NewRecorder()
	handler.ServeHTTP(allowedResp, allowedReq)
	if allowedResp.Code != http.StatusNoContent {
		t.Fatalf("expected allowed status %d, got %d", http.StatusNoContent, allowedResp.Code)
	}

	deniedReq := httptest.NewRequest(http.MethodGet, "/", nil)
	deniedReq = deniedReq.WithContext(WithPrincipal(deniedReq.Context(), Principal{Roles: []string{RoleArtificialFlowClient}}))
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
	if !principal.HasRole(RoleArtificialFlowAdmin) {
		t.Fatal("expected disabled-auth principal to have admin role")
	}
	if !principal.HasRole(RoleArtificialFlowModeler) {
		t.Fatal("expected disabled-auth principal to have modeler role")
	}
	if !principal.HasRole(RoleArtificialFlowClient) {
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
		if !principal.HasRole(RoleArtificialFlowAdmin) {
			t.Fatal("expected disabled-auth principal to have admin role")
		}
		w.WriteHeader(http.StatusNoContent)
	})

	handler := middleware.Handler(RequireAnyRole(RoleArtificialFlowAdmin)(next))
	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusNoContent {
		t.Fatalf("expected allowed status %d, got %d", http.StatusNoContent, resp.Code)
	}
}
