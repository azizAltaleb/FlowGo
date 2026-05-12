package auth

import "testing"

func TestPrincipalFromClaims_MapsNestedRolesAndScopes(t *testing.T) {
	claims := map[string]any{
		"sub":   "user-123",
		"email": "user@example.com",
		"name":  "Example User",
		"realm_access": map[string]any{
			"roles": []any{"admin", "reader"},
		},
		"scope": "openid profile workflows:read",
		"aud":   []any{"workflow-backend"},
	}

	cfg := Config{
		ClaimSubjectPath: "sub",
		ClaimEmailPath:   "email",
		ClaimNamePath:    "name",
		ClaimRolesPath:   "roles,realm_access.roles,groups",
		ClaimScopesPath:  "scope,scp",
	}

	principal := principalFromClaims(claims, "", cfg, TokenModeJWT)
	if principal.Subject != "user-123" {
		t.Fatalf("expected mapped subject, got %s", principal.Subject)
	}
	if len(principal.Roles) != 2 {
		t.Fatalf("expected 2 roles, got %d", len(principal.Roles))
	}
	if len(principal.Scopes) != 3 {
		t.Fatalf("expected 3 scopes, got %d", len(principal.Scopes))
	}
	if len(principal.Audience) != 1 || principal.Audience[0] != "workflow-backend" {
		t.Fatalf("expected audience mapping, got %#v", principal.Audience)
	}
}

func TestPrincipalFromClaims_PreservesRoleNamesWithSpaces(t *testing.T) {
	claims := map[string]any{
		"sub":   "user-123",
		"roles": "flowgo admin,flowgo viewer",
	}

	principal := principalFromClaims(claims, "", Config{ClaimRolesPath: "roles"}, TokenModeJWT)
	if len(principal.Roles) != 2 {
		t.Fatalf("expected 2 roles, got %#v", principal.Roles)
	}
	if principal.Roles[0] != RoleFlowGoAdmin {
		t.Fatalf("expected first role %q, got %q", RoleFlowGoAdmin, principal.Roles[0])
	}
	if principal.Roles[1] != RoleFlowGoViewer {
		t.Fatalf("expected second role %q, got %q", RoleFlowGoViewer, principal.Roles[1])
	}
}

func TestPrincipalFromClaims_MapsZitadelProjectRolesObject(t *testing.T) {
	claims := map[string]any{
		"sub": "user-123",
		"urn:zitadel:iam:org:project:roles": map[string]any{
			RoleFlowGoAdmin: map[string]any{},
		},
	}

	principal := principalFromClaims(claims, "", Config{ClaimRolesPath: "urn:zitadel:iam:org:project:roles"}, TokenModeJWT)
	if len(principal.Roles) != 1 || principal.Roles[0] != RoleFlowGoAdmin {
		t.Fatalf("expected ZITADEL role %q, got %#v", RoleFlowGoAdmin, principal.Roles)
	}
}
