package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStandardRoles(t *testing.T) {
	roles := StandardRoles()
	expected := []string{RoleWorkflowsaClient, RoleWorkflowsaAdmin, RoleWorkflowsaViewer}
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
	principal := Principal{Roles: []string{" Workflowsa Admin "}}
	if !principal.HasRole(RoleWorkflowsaAdmin) {
		t.Fatal("expected case-insensitive role match")
	}
	if principal.HasRole(RoleWorkflowsaViewer) {
		t.Fatal("did not expect viewer role match")
	}
}

func TestRequireAnyRole(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	handler := RequireAnyRole(RoleWorkflowsaAdmin)(next)

	allowedReq := httptest.NewRequest(http.MethodGet, "/", nil)
	allowedReq = allowedReq.WithContext(WithPrincipal(allowedReq.Context(), Principal{Roles: []string{RoleWorkflowsaAdmin}}))
	allowedResp := httptest.NewRecorder()
	handler.ServeHTTP(allowedResp, allowedReq)
	if allowedResp.Code != http.StatusNoContent {
		t.Fatalf("expected allowed status %d, got %d", http.StatusNoContent, allowedResp.Code)
	}

	deniedReq := httptest.NewRequest(http.MethodGet, "/", nil)
	deniedReq = deniedReq.WithContext(WithPrincipal(deniedReq.Context(), Principal{Roles: []string{RoleWorkflowsaViewer}}))
	deniedResp := httptest.NewRecorder()
	handler.ServeHTTP(deniedResp, deniedReq)
	if deniedResp.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden status %d, got %d", http.StatusForbidden, deniedResp.Code)
	}
}
