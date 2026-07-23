package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/artificialflow/artificialflow/backend/libs/auth"
	"github.com/artificialflow/artificialflow/backend/libs/iam"
	"github.com/artificialflow/artificialflow/backend/libs/model"
	workerSDK "github.com/artificialflow/artificialflow/backend/libs/worker"
	"github.com/artificialflow/artificialflow/backend/services/workflow-command/internal/application"
	"github.com/artificialflow/artificialflow/backend/services/workflow-command/internal/infrastructure/messaging"
	"github.com/artificialflow/artificialflow/backend/services/workflow-command/internal/infrastructure/persistence"
	"github.com/artificialflow/artificialflow/backend/services/workflow-command/internal/interfaces/http/dto"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestHandler(t *testing.T) *Handler {
	return setupTestHandlerWithIdentityConfig(t, iam.DeploymentConfig{
		Mode:                iam.DeploymentModeExternal,
		ProviderName:        "Corporate OIDC",
		ConfigurationSource: "docker-compose",
		AuthConfig: auth.Config{
			InternalIssuerURL:   "http://identity.internal/realms/flowgo",
			ExternalIssuerURL:   "https://identity.example.com/realms/flowgo",
			ClientID:            "workflow-backend",
			TokenValidationMode: auth.TokenModeJWT,
			EnforceAudience:     false,
			AllowInsecureIssuer: true,
			ClaimSubjectPath:    "sub",
			ClaimRolesPath:      "roles",
			ClaimScopesPath:     "scope",
			ClaimTenantPath:     "tenant_id",
			ClaimEmailPath:      "email",
			ClaimNamePath:       "name",
		},
		FrontendConfig: iam.FrontendAuthConfig{
			Enabled:       true,
			OIDCAuthority: "https://identity.example.com/realms/flowgo",
			OIDCClientID:  "workflow-frontend",
		},
	})
}

func setupTestHandlerWithIdentityConfig(t *testing.T, identityConfig iam.DeploymentConfig) *Handler {
	h, _ := setupTestHandlerWithRepository(t, identityConfig)
	return h
}

func setupTestHandlerWithRepository(t *testing.T, identityConfig iam.DeploymentConfig) (*Handler, *persistence.GormRepository) {
	// Use in-memory SQLite for testing
	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open("file:"+dbName+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open sqlite db: %v", err)
	}

	// Auto-migrate schema using the helper from persistence if we made it public,
	// or just manual migration here since we are using GormRepository.
	// Actually, NewGormRepository doesn't migrate, NewPostgresRepository does.
	// We should duplicate migration logic here for tests.
	if err := db.AutoMigrate(
		&model.Process{},
		&model.ProcessInstance{},
		&model.ElementInstance{},
		&model.Variable{},
		&model.Job{},
		&model.Incident{},
		&model.Timer{},
		&model.MessageSubscription{},
		&model.IdempotencyRecord{},
		&model.OutboxMessage{},
	); err != nil {
		t.Fatalf("Failed to migrate schema: %v", err)
	}

	repo := persistence.NewGormRepository(db)
	e := application.NewEngine(repo, &messaging.NoOpPublisher{})
	return NewHandler(e, identityConfig), repo
}

func registerTestRoutes(r *mux.Router, h *Handler) {
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if _, ok := auth.PrincipalFromContext(req.Context()); !ok {
				principal := auth.Principal{
					Subject: "test-admin",
					Roles:   []string{auth.RoleArtificialFlowAdmin},
				}
				req = req.WithContext(auth.WithPrincipal(req.Context(), principal))
			}
			next.ServeHTTP(w, req)
		})
	})
	h.RegisterRoutes(r)
}

func TestIdentityConfigAPI(t *testing.T) {
	h := setupTestHandler(t)
	r := mux.NewRouter()
	registerTestRoutes(r, h)
	ts := httptest.NewServer(r)
	defer ts.Close()

	configResp, err := http.Get(ts.URL + "/identity/config")
	if err != nil {
		t.Fatalf("Failed to fetch identity config: %v", err)
	}
	if configResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(configResp.Body)
		t.Fatalf("Expected 200 OK for initial config, got %d: %s", configResp.StatusCode, string(body))
	}
	var config dto.IdentityConfigResponse
	if err := json.NewDecoder(configResp.Body).Decode(&config); err != nil {
		t.Fatalf("Failed to decode initial identity config: %v", err)
	}
	if config.DeploymentMode != iam.DeploymentModeExternal {
		t.Fatalf("Expected deployment mode %q, got %q", iam.DeploymentModeExternal, config.DeploymentMode)
	}
	if config.ConfigurationSource != "docker-compose" {
		t.Fatalf("Expected configuration source docker-compose, got %q", config.ConfigurationSource)
	}
	if config.ProviderName != "Corporate OIDC" {
		t.Fatalf("Expected provider name Corporate OIDC, got %q", config.ProviderName)
	}
	if !config.AuthEnabled {
		t.Fatalf("Expected backend auth to be enabled")
	}
	if !config.FrontendAuthEnabled {
		t.Fatalf("Expected frontend auth to be enabled")
	}
	if config.TokenValidationMode != auth.TokenModeJWT {
		t.Fatalf("Expected token mode %q, got %q", auth.TokenModeJWT, config.TokenValidationMode)
	}
	if config.InternalIssuerURL != "http://identity.internal/realms/flowgo" {
		t.Fatalf("Unexpected internal issuer %q", config.InternalIssuerURL)
	}
	if config.ExternalIssuerURL != "https://identity.example.com/realms/flowgo" {
		t.Fatalf("Unexpected external issuer %q", config.ExternalIssuerURL)
	}
	if config.ClientID != "workflow-backend" {
		t.Fatalf("Expected client ID workflow-backend, got %q", config.ClientID)
	}
	if config.FrontendOIDCAuthority != "https://identity.example.com/realms/flowgo" {
		t.Fatalf("Unexpected frontend authority %q", config.FrontendOIDCAuthority)
	}
	if config.FrontendOIDCClientID != "workflow-frontend" {
		t.Fatalf("Unexpected frontend client id %q", config.FrontendOIDCClientID)
	}
	expectedRoles := []string{auth.RoleArtificialFlowAdmin, auth.RoleArtificialFlowModeler, auth.RoleArtificialFlowClient}
	if len(config.StandardRoles) != len(expectedRoles) {
		t.Fatalf("Expected %d standard roles, got %#v", len(expectedRoles), config.StandardRoles)
	}
	for i, role := range expectedRoles {
		if config.StandardRoles[i] != role {
			t.Fatalf("Expected standard role %q at index %d, got %q", role, i, config.StandardRoles[i])
		}
	}
}

func TestIdentityManagementRoutesExternalModeReturnNotFound(t *testing.T) {
	h := setupTestHandler(t)
	r := mux.NewRouter()
	registerTestRoutes(r, h)
	req := httptest.NewRequest(http.MethodGet, "/identity/management/users", nil)
	req = req.WithContext(auth.WithPrincipal(req.Context(), auth.Principal{Subject: "admin", Roles: []string{auth.RoleArtificialFlowAdmin}}))
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("Expected 404 for external IAM management route, got %d", rec.Code)
	}
}

func TestIdentityManagementRoutesBundledRequireAdmin(t *testing.T) {
	h := setupTestHandlerWithIdentityConfig(t, iam.DeploymentConfig{Mode: iam.DeploymentModeZITADEL})
	r := mux.NewRouter()
	registerTestRoutes(r, h)
	req := httptest.NewRequest(http.MethodGet, "/identity/management/users", nil)
	req = req.WithContext(auth.WithPrincipal(req.Context(), auth.Principal{Subject: "accountant", Roles: []string{"accountant"}}))
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("Expected 403 for non-admin bundled IAM management route, got %d", rec.Code)
	}
}

func TestIdentityManagementRoutesBundledAdminListsUsers(t *testing.T) {
	tokenFile := t.TempDir() + "/owner.pat"
	stateFile := t.TempDir() + "/flowgo-zitadel.json"
	if err := os.WriteFile(tokenFile, []byte("owner-token"), 0600); err != nil {
		t.Fatalf("Failed to write owner token: %v", err)
	}
	if err := os.WriteFile(stateFile, []byte(`{"org_id":"org-1","project_id":"project-1"}`), 0600); err != nil {
		t.Fatalf("Failed to write bootstrap state: %v", err)
	}
	zitadel := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer owner-token" {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/zitadel.user.v2.UserService/ListUsers":
			_, _ = w.Write([]byte(`{"result":[{"userId":"user-1","username":"admin@example.com","preferredLoginName":"admin@example.com","state":"USER_STATE_ACTIVE","details":{"creationDate":"2026-01-01T00:00:00Z","changeDate":"2026-01-01T00:00:00Z"},"human":{"profile":{"givenName":"Admin","familyName":"User","displayName":"Admin User"},"email":{"email":"admin@example.com","isVerified":true}}},{"userId":"user-2","username":"flowgo-bootstrap","preferredLoginName":"flowgo-bootstrap","state":"USER_STATE_ACTIVE","machine":{"name":"flowgo-bootstrap"}},{"userId":"user-3","username":"login-client","preferredLoginName":"login-client","state":"USER_STATE_ACTIVE","machine":{"name":"workflow-login-client"}}]}`))
		case "/zitadel.authorization.v2.AuthorizationService/ListAuthorizations":
			_, _ = w.Write([]byte(`{"authorizations":[{"id":"auth-1","state":"STATE_ACTIVE","project":{"id":"project-1"},"user":{"id":"user-1"},"roles":[{"key":"flowgo admin","displayName":"FlowGo Admin","group":"FlowGo"}]}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer zitadel.Close()

	h := setupTestHandlerWithIdentityConfig(t, iam.DeploymentConfig{
		Mode: iam.DeploymentModeZITADEL,
		ZITADELManagement: iam.ZITADELManagementConfig{
			BaseURL:            zitadel.URL,
			OwnerPATFile:       tokenFile,
			BootstrapStateFile: stateFile,
		},
	})
	r := mux.NewRouter()
	registerTestRoutes(r, h)
	req := httptest.NewRequest(http.MethodGet, "/identity/management/users", nil)
	req = req.WithContext(auth.WithPrincipal(req.Context(), auth.Principal{Subject: "admin", Roles: []string{auth.RoleArtificialFlowAdmin}}))
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected 200 for admin bundled IAM management route, got %d: %s", rec.Code, rec.Body.String())
	}
	var response dto.ListIdentityManagementUsersResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode users response: %v", err)
	}
	hiddenIdentities := []string{"workflow-login-client", "login-client", "flowgo-bootstrap"}
	for _, user := range response.Users {
		userIdentities := []string{user.Username, user.PreferredLoginName, user.DisplayName, user.Email}
		for _, userIdentity := range userIdentities {
			for _, hiddenIdentity := range hiddenIdentities {
				if strings.EqualFold(strings.TrimSpace(userIdentity), hiddenIdentity) {
					t.Fatalf("Expected internal ZITADEL user %q to be hidden, got %#v", hiddenIdentity, response.Users)
				}
			}
		}
	}
	if len(response.Users) != 1 {
		t.Fatalf("Expected one user, got %#v", response.Users)
	}
	if response.Users[0].Email != "admin@example.com" {
		t.Fatalf("Unexpected user email %q", response.Users[0].Email)
	}
	if len(response.Users[0].Roles) != 1 || response.Users[0].Roles[0] != auth.RoleArtificialFlowAdmin {
		t.Fatalf("Unexpected user roles %#v", response.Users[0].Roles)
	}
}

func TestIdentityManagementReactivateUserRequiresBundledAdmin(t *testing.T) {
	tokenFile := t.TempDir() + "/owner.pat"
	stateFile := t.TempDir() + "/flowgo-zitadel.json"
	if err := os.WriteFile(tokenFile, []byte("owner-token"), 0600); err != nil {
		t.Fatalf("Failed to write owner token: %v", err)
	}
	if err := os.WriteFile(stateFile, []byte(`{"org_id":"org-1","project_id":"project-1"}`), 0600); err != nil {
		t.Fatalf("Failed to write bootstrap state: %v", err)
	}

	reactivateCalled := false
	zitadel := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer owner-token" {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/zitadel.user.v2.UserService/ReactivateUser":
			reactivateCalled = true
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("failed to decode reactivate payload: %v", err)
			}
			if payload["userId"] != "user-1" {
				t.Fatalf("expected user-1 reactivate payload, got %#v", payload)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer zitadel.Close()

	h := setupTestHandlerWithIdentityConfig(t, iam.DeploymentConfig{
		Mode: iam.DeploymentModeZITADEL,
		ZITADELManagement: iam.ZITADELManagementConfig{
			BaseURL:            zitadel.URL,
			OwnerPATFile:       tokenFile,
			BootstrapStateFile: stateFile,
		},
	})
	r := mux.NewRouter()
	registerTestRoutes(r, h)

	nonAdminReq := httptest.NewRequest(http.MethodPost, "/identity/management/users/user-1/reactivate", nil)
	nonAdminReq = nonAdminReq.WithContext(auth.WithPrincipal(nonAdminReq.Context(), auth.Principal{Subject: "accountant", Roles: []string{"accountant"}}))
	nonAdminRec := httptest.NewRecorder()
	r.ServeHTTP(nonAdminRec, nonAdminReq)
	if nonAdminRec.Code != http.StatusForbidden {
		t.Fatalf("expected non-admin reactivate 403, got %d: %s", nonAdminRec.Code, nonAdminRec.Body.String())
	}

	adminReq := httptest.NewRequest(http.MethodPost, "/identity/management/users/user-1/reactivate", nil)
	adminReq = adminReq.WithContext(auth.WithPrincipal(adminReq.Context(), auth.Principal{Subject: "admin", Roles: []string{auth.RoleArtificialFlowAdmin}}))
	adminRec := httptest.NewRecorder()
	r.ServeHTTP(adminRec, adminReq)
	if adminRec.Code != http.StatusNoContent {
		t.Fatalf("expected admin reactivate 204, got %d: %s", adminRec.Code, adminRec.Body.String())
	}
	if !reactivateCalled {
		t.Fatalf("expected ZITADEL ReactivateUser to be called")
	}
}

func TestIdentityManagementProtectsFlowGoPlatformRoles(t *testing.T) {
	h := setupTestHandlerWithIdentityConfig(t, iam.DeploymentConfig{Mode: iam.DeploymentModeZITADEL})
	r := mux.NewRouter()
	registerTestRoutes(r, h)
	admin := auth.Principal{Subject: "admin", Roles: []string{auth.RoleArtificialFlowAdmin}}

	requests := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{
			name:   "create viewer",
			method: http.MethodPost,
			path:   "/identity/management/roles",
			body:   `{"key":"flowgo viewer","display_name":"FlowGo Viewer","group":"FlowGo"}`,
		},
		{
			name:   "update admin",
			method: http.MethodPut,
			path:   "/identity/management/roles/flowgo%20admin",
			body:   `{"display_name":"Changed Admin","group":"FlowGo"}`,
		},
		{
			name:   "delete modeler",
			method: http.MethodDelete,
			path:   "/identity/management/roles/flowgo%20modeler",
		},
		{
			name:   "delete client",
			method: http.MethodDelete,
			path:   "/identity/management/roles/flowgo%20client",
		},
	}

	for _, tc := range requests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			req = req.WithContext(auth.WithPrincipal(req.Context(), admin))
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected protected platform role request to return 400, got %d: %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestIdentityManagementRoutesBundledAdminCreatesClientToken(t *testing.T) {
	tokenFile := t.TempDir() + "/owner.pat"
	stateFile := t.TempDir() + "/flowgo-zitadel.json"
	if err := os.WriteFile(tokenFile, []byte("owner-token"), 0600); err != nil {
		t.Fatalf("Failed to write owner token: %v", err)
	}
	if err := os.WriteFile(stateFile, []byte(`{"org_id":"org-1","project_id":"project-1"}`), 0600); err != nil {
		t.Fatalf("Failed to write bootstrap state: %v", err)
	}
	var createUserPayload map[string]any
	var createAuthorizationPayload map[string]any
	var tokenPayload map[string]any
	zitadel := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer owner-token" {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/zitadel.user.v2.UserService/CreateUser":
			if err := json.NewDecoder(r.Body).Decode(&createUserPayload); err != nil {
				t.Fatalf("failed to decode create user payload: %v", err)
			}
			_, _ = w.Write([]byte(`{"id":"client-user-1"}`))
		case "/zitadel.authorization.v2.AuthorizationService/ListAuthorizations":
			_, _ = w.Write([]byte(`{"authorizations":[]}`))
		case "/zitadel.authorization.v2.AuthorizationService/CreateAuthorization":
			if err := json.NewDecoder(r.Body).Decode(&createAuthorizationPayload); err != nil {
				t.Fatalf("failed to decode create authorization payload: %v", err)
			}
			_, _ = w.Write([]byte(`{"id":"auth-1"}`))
		case "/zitadel.user.v2.UserService/AddPersonalAccessToken":
			if err := json.NewDecoder(r.Body).Decode(&tokenPayload); err != nil {
				t.Fatalf("failed to decode token payload: %v", err)
			}
			_, _ = w.Write([]byte(`{"creationDate":"2026-01-01T00:00:00Z","tokenId":"pat-1","token":"sdk-token"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer zitadel.Close()

	h := setupTestHandlerWithIdentityConfig(t, iam.DeploymentConfig{
		Mode: iam.DeploymentModeZITADEL,
		ZITADELManagement: iam.ZITADELManagementConfig{
			BaseURL:                 zitadel.URL,
			OwnerPATFile:            tokenFile,
			BootstrapStateFile:      stateFile,
			EnableLegacyPATCreation: true,
		},
	})
	r := mux.NewRouter()
	registerTestRoutes(r, h)
	body := `{"username":"sdk-orders","name":"Orders SDK","description":"Order system","environment":"production","owner_email":"platform@example.com","purpose":"Order worker","token_expires_at":"2027-01-01T00:00:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/identity/management/clients", strings.NewReader(body))
	req = req.WithContext(auth.WithPrincipal(req.Context(), auth.Principal{Subject: "admin", Roles: []string{auth.RoleArtificialFlowAdmin}}))
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("Expected 201 for client token creation, got %d: %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Cache-Control") != "no-store" || rec.Header().Get("Pragma") != "no-cache" {
		t.Fatalf("expected one-time token response to disable caching, got headers %#v", rec.Header())
	}
	var response dto.IdentityManagementClientTokenResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode client token response: %v", err)
	}
	if response.ClientID != "client-user-1" || response.TokenID != "pat-1" || response.Token != "sdk-token" {
		t.Fatalf("Unexpected client token response: %#v", response)
	}
	if response.Role != auth.RoleArtificialFlowClient {
		t.Fatalf("Expected client role %q, got %q", auth.RoleArtificialFlowClient, response.Role)
	}
	if createUserPayload["organizationId"] != "org-1" || createUserPayload["username"] != "sdk-orders" {
		t.Fatalf("Unexpected create user payload: %#v", createUserPayload)
	}
	machine, ok := createUserPayload["machine"].(map[string]any)
	if !ok {
		t.Fatalf("Expected machine payload, got %#v", createUserPayload)
	}
	description, ok := machine["description"].(string)
	if !ok || !strings.Contains(description, "Order system") || !strings.Contains(description, "production") || !strings.Contains(description, "platform@example.com") {
		t.Fatalf("Unexpected machine description: %#v", machine["description"])
	}
	if machine["name"] != "Orders SDK" || machine["accessTokenType"] != "ACCESS_TOKEN_TYPE_JWT" {
		t.Fatalf("Unexpected machine payload: %#v", machine)
	}
	roleKeys, ok := createAuthorizationPayload["roleKeys"].([]any)
	if !ok || len(roleKeys) != 1 || roleKeys[0] != auth.RoleArtificialFlowClient {
		t.Fatalf("Unexpected authorization payload: %#v", createAuthorizationPayload)
	}
	if tokenPayload["userId"] != "client-user-1" || tokenPayload["expirationDate"] != "2027-01-01T00:00:00Z" {
		t.Fatalf("Unexpected PAT payload: %#v", tokenPayload)
	}
}

func TestIdentityManagementRoutesBundledAdminManagesClients(t *testing.T) {
	tokenFile := t.TempDir() + "/owner.pat"
	stateFile := t.TempDir() + "/flowgo-zitadel.json"
	if err := os.WriteFile(tokenFile, []byte("owner-token"), 0600); err != nil {
		t.Fatalf("Failed to write owner token: %v", err)
	}
	if err := os.WriteFile(stateFile, []byte(`{"org_id":"org-1","project_id":"project-1"}`), 0600); err != nil {
		t.Fatalf("Failed to write bootstrap state: %v", err)
	}
	var rotatePayload map[string]any
	var revokePayload map[string]any
	var deletePayload map[string]any
	zitadel := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer owner-token" {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/zitadel.user.v2.UserService/ListUsers":
			_, _ = w.Write([]byte(`{"result":[{"userId":"client-user-1","username":"sdk-orders","preferredLoginName":"sdk-orders","state":"USER_STATE_ACTIVE","details":{"creationDate":"2026-01-01T00:00:00Z","changeDate":"2026-01-02T00:00:00Z"},"machine":{"name":"Orders SDK","description":"flowgo-client:{\"description\":\"Order system\",\"environment\":\"production\",\"owner_email\":\"platform@example.com\",\"purpose\":\"Order worker\"}"}},{"userId":"bootstrap","username":"flowgo-bootstrap","state":"USER_STATE_ACTIVE","machine":{"name":"flowgo-bootstrap"}}]}`))
		case "/zitadel.authorization.v2.AuthorizationService/ListAuthorizations":
			_, _ = w.Write([]byte(`{"authorizations":[{"id":"auth-1","state":"STATE_ACTIVE","project":{"id":"project-1"},"user":{"id":"client-user-1"},"roles":[{"key":"flowgo client"}]}]}`))
		case "/zitadel.user.v2.UserService/ListPersonalAccessTokens":
			_, _ = w.Write([]byte(`{"result":[{"id":"pat-1","userId":"client-user-1","organizationId":"org-1","creationDate":"2026-01-01T00:00:00Z","changeDate":"2026-01-01T00:00:00Z","expirationDate":"2027-01-01T00:00:00Z"}]}`))
		case "/zitadel.user.v2.UserService/ListKeys":
			_, _ = w.Write([]byte(`{"keys":[]}`))
		case "/zitadel.user.v2.UserService/GetUserByID":
			_, _ = w.Write([]byte(`{"user":{"userId":"client-user-1","username":"sdk-orders","preferredLoginName":"sdk-orders","state":"USER_STATE_ACTIVE","details":{"creationDate":"2026-01-01T00:00:00Z","changeDate":"2026-01-02T00:00:00Z"},"machine":{"name":"Orders SDK","description":"flowgo-client:{\"description\":\"Order system\",\"environment\":\"production\",\"owner_email\":\"platform@example.com\",\"purpose\":\"Order worker\"}"}}}`))
		case "/zitadel.user.v2.UserService/AddPersonalAccessToken":
			if err := json.NewDecoder(r.Body).Decode(&rotatePayload); err != nil {
				t.Fatalf("failed to decode rotate payload: %v", err)
			}
			_, _ = w.Write([]byte(`{"creationDate":"2026-02-01T00:00:00Z","tokenId":"pat-2","token":"rotated-token"}`))
		case "/zitadel.user.v2.UserService/RemovePersonalAccessToken":
			if err := json.NewDecoder(r.Body).Decode(&revokePayload); err != nil {
				t.Fatalf("failed to decode revoke payload: %v", err)
			}
			_, _ = w.Write([]byte(`{"deletionDate":"2026-02-02T00:00:00Z"}`))
		case "/zitadel.user.v2.UserService/DeleteUser":
			if err := json.NewDecoder(r.Body).Decode(&deletePayload); err != nil {
				t.Fatalf("failed to decode delete payload: %v", err)
			}
			_, _ = w.Write([]byte(`{}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer zitadel.Close()

	h := setupTestHandlerWithIdentityConfig(t, iam.DeploymentConfig{
		Mode: iam.DeploymentModeZITADEL,
		ZITADELManagement: iam.ZITADELManagementConfig{
			BaseURL:                 zitadel.URL,
			OwnerPATFile:            tokenFile,
			BootstrapStateFile:      stateFile,
			EnableLegacyPATRotation: true,
		},
	})
	r := mux.NewRouter()
	registerTestRoutes(r, h)

	req := httptest.NewRequest(http.MethodGet, "/identity/management/clients", nil)
	req = req.WithContext(auth.WithPrincipal(req.Context(), auth.Principal{Subject: "admin", Roles: []string{auth.RoleArtificialFlowAdmin}}))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Expected 200 for client list, got %d: %s", rec.Code, rec.Body.String())
	}
	var listResponse dto.ListIdentityManagementClientsResponse
	if err := json.NewDecoder(rec.Body).Decode(&listResponse); err != nil {
		t.Fatalf("Failed to decode client list: %v", err)
	}
	if len(listResponse.Clients) != 1 || listResponse.Clients[0].ClientID != "client-user-1" || listResponse.Clients[0].Environment != "production" {
		t.Fatalf("Unexpected client list response: %#v", listResponse)
	}
	if len(listResponse.Clients[0].Tokens) != 1 || listResponse.Clients[0].Tokens[0].TokenID != "pat-1" {
		t.Fatalf("Unexpected token summaries: %#v", listResponse.Clients[0].Tokens)
	}

	rotateBody := `{"token_expires_at":"2028-01-01T00:00:00Z"}`
	req = httptest.NewRequest(http.MethodPost, "/identity/management/clients/client-user-1/tokens", strings.NewReader(rotateBody))
	req = req.WithContext(auth.WithPrincipal(req.Context(), auth.Principal{Subject: "admin", Roles: []string{auth.RoleArtificialFlowAdmin}}))
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("Expected 201 for client token rotation, got %d: %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Cache-Control") != "no-store" || rec.Header().Get("Pragma") != "no-cache" {
		t.Fatalf("expected rotated token response to disable caching, got headers %#v", rec.Header())
	}
	var rotateResponse dto.IdentityManagementClientTokenResponse
	if err := json.NewDecoder(rec.Body).Decode(&rotateResponse); err != nil {
		t.Fatalf("Failed to decode rotated token: %v", err)
	}
	if rotateResponse.TokenID != "pat-2" || rotateResponse.Token != "rotated-token" || rotatePayload["userId"] != "client-user-1" {
		t.Fatalf("Unexpected rotate response or payload: %#v %#v", rotateResponse, rotatePayload)
	}
	if revokePayload != nil {
		t.Fatalf("rotation must preserve the old token until explicit revocation, got revoke payload %#v", revokePayload)
	}

	req = httptest.NewRequest(http.MethodDelete, "/identity/management/clients/client-user-1/tokens/pat-1", nil)
	req = req.WithContext(auth.WithPrincipal(req.Context(), auth.Principal{Subject: "admin", Roles: []string{auth.RoleArtificialFlowAdmin}}))
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("Expected 204 for client token revoke, got %d: %s", rec.Code, rec.Body.String())
	}
	if revokePayload["userId"] != "client-user-1" || revokePayload["tokenId"] != "pat-1" {
		t.Fatalf("Unexpected revoke payload: %#v", revokePayload)
	}

	req = httptest.NewRequest(http.MethodDelete, "/identity/management/clients/client-user-1", nil)
	req = req.WithContext(auth.WithPrincipal(req.Context(), auth.Principal{Subject: "admin", Roles: []string{auth.RoleArtificialFlowAdmin}}))
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("Expected 204 for client delete, got %d: %s", rec.Code, rec.Body.String())
	}
	if deletePayload["userId"] != "client-user-1" {
		t.Fatalf("Unexpected delete payload: %#v", deletePayload)
	}
}

func TestCompleteTaskAPI(t *testing.T) {
	h := setupTestHandler(t)
	r := mux.NewRouter()
	registerTestRoutes(r, h)

	// Deploy Workflow
	steps := []model.StepDefinition{
		{ID: "start", Type: model.StepTypeStart, Outgoing: []model.Transition{{TargetRef: "task1"}}},
		{ID: "task1", Type: model.StepTypeUserTask, Outgoing: []model.Transition{{TargetRef: "end"}}},
		{ID: "end", Type: model.StepTypeEnd},
	}
	wf, err := h.engine.DeployWorkflow(context.Background(), "API Test", steps)
	if err != nil {
		t.Fatalf("Failed to deploy: %v", err)
	}

	// Start Instance
	instance, err := h.engine.StartInstance(context.Background(), strconv.FormatInt(wf.ID, 10), nil)
	if err != nil {
		t.Fatalf("Failed to start instance: %v", err)
	}

	// Complete Task without step_id
	ts := httptest.NewServer(r)
	defer ts.Close()

	url := ts.URL + "/instances/" + instance.ID + "/complete"
	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		t.Fatalf("Failed to call complete: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", resp.StatusCode)
	}

	// Verify completion
	instance, _ = h.engine.GetInstance(context.Background(), instance.ID)
	if instance.Status != model.StatusCompleted {
		t.Errorf("Expected instance completed, got %s", instance.Status)
	}
}

func TestCompleteTaskByExecutionIDAPI(t *testing.T) {
	h := setupTestHandler(t)
	r := mux.NewRouter()
	registerTestRoutes(r, h)
	ts := httptest.NewServer(r)
	defer ts.Close()

	// Deploy
	steps := []model.StepDefinition{
		{ID: "start", Type: model.StepTypeStart, Outgoing: []model.Transition{{TargetRef: "task1"}}},
		{ID: "task1", Type: model.StepTypeUserTask, Outgoing: []model.Transition{{TargetRef: "end"}}},
		{ID: "end", Type: model.StepTypeEnd},
	}
	wf, _ := h.engine.DeployWorkflow(context.Background(), "ExecID Test", steps)
	instance, _ := h.engine.StartInstance(context.Background(), strconv.FormatInt(wf.ID, 10), nil)

	// Find Active Execution ID
	instance, _ = h.engine.GetInstance(context.Background(), instance.ID)
	var execID string
	for _, ex := range instance.Executions {
		if ex.Status == "ACTIVE" {
			execID = ex.ID
			break
		}
	}

	// Complete using Execution ID
	url := ts.URL + "/instances/" + instance.ID + "/complete"
	reqBody := dto.CompleteTaskRequest{StepID: execID}
	jsonBody, _ := json.Marshal(reqBody)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		t.Fatalf("Failed to call complete: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("Expected 200 OK, got %d: %s", resp.StatusCode, string(body))
	}

	// Verify
	instance, _ = h.engine.GetInstance(context.Background(), instance.ID)
	if instance.Status != model.StatusCompleted {
		t.Errorf("Expected completed, got %s", instance.Status)
	}
}

func TestCompleteParallelTaskAPI(t *testing.T) {
	h := setupTestHandler(t)
	r := mux.NewRouter()
	registerTestRoutes(r, h)
	ts := httptest.NewServer(r)
	defer ts.Close()

	// Deploy Parallel Workflow
	steps := []model.StepDefinition{
		{ID: "start", Type: model.StepTypeStart, Outgoing: []model.Transition{{TargetRef: "split"}}},
		{ID: "split", Type: model.StepTypeGatewayParallel, Outgoing: []model.Transition{
			{TargetRef: "taskA"},
			{TargetRef: "taskB"},
		}},
		{ID: "taskA", Type: model.StepTypeUserTask, Outgoing: []model.Transition{{TargetRef: "join"}}},
		{ID: "taskB", Type: model.StepTypeUserTask, Outgoing: []model.Transition{{TargetRef: "join"}}},
		{ID: "join", Type: model.StepTypeGatewayParallel, Incoming: []string{"taskA", "taskB"}, Outgoing: []model.Transition{{TargetRef: "end"}}},
		{ID: "end", Type: model.StepTypeEnd},
	}
	wf, err := h.engine.DeployWorkflow(context.Background(), "Parallel API Test", steps)
	if err != nil {
		t.Fatalf("Failed to deploy: %v", err)
	}

	// Start Instance
	instance, err := h.engine.StartInstance(context.Background(), strconv.FormatInt(wf.ID, 10), nil)
	if err != nil {
		t.Fatalf("Failed to start instance: %v", err)
	}

	// Complete Task A explicitly using step_id
	reqBody := dto.CompleteTaskRequest{StepID: "taskA"}
	jsonBody, _ := json.Marshal(reqBody)
	url := ts.URL + "/instances/" + instance.ID + "/complete"

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		t.Fatalf("Failed to call complete for taskA: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 OK for taskA, got %d", resp.StatusCode)
	}

	// Reload instance
	instance, _ = h.engine.GetInstance(context.Background(), instance.ID)
	// Status should still be running
	if instance.Status != model.StatusRunning {
		t.Errorf("Expected running, got %s", instance.Status)
	}

	// Complete Task B explicitly using step_id
	reqBody = dto.CompleteTaskRequest{StepID: "taskB"}
	jsonBody, _ = json.Marshal(reqBody)
	resp, err = http.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		t.Fatalf("Failed to call complete for taskB: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 OK for taskB, got %d", resp.StatusCode)
	}

	// Reload instance -> Should be completed
	instance, _ = h.engine.GetInstance(context.Background(), instance.ID)
	if instance.Status != model.StatusCompleted {
		t.Errorf("Expected completed, got %s", instance.Status)
	}
}

func TestDeployBPMNAPI(t *testing.T) {
	h := setupTestHandler(t)
	r := mux.NewRouter()
	registerTestRoutes(r, h)
	ts := httptest.NewServer(r)
	defer ts.Close()

	bpmnXML := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="Definitions_1" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_1" isExecutable="true">
    <bpmn:startEvent id="StartEvent_1"/>
    <bpmn:endEvent id="EndEvent_1"/>
    <bpmn:sequenceFlow id="Flow_1" sourceRef="StartEvent_1" targetRef="EndEvent_1"/>
  </bpmn:process>
</bpmn:definitions>`

	url := ts.URL + "/workflows"
	resp, err := http.Post(url, "application/xml", bytes.NewBufferString(bpmnXML))
	if err != nil {
		t.Fatalf("Failed to deploy BPMN: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", resp.StatusCode)
	}

	var wf dto.WorkflowDefinitionResponse
	if err := json.NewDecoder(resp.Body).Decode(&wf); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if wf.ProcessDefinitionID != "Process_1" {
		t.Errorf("Expected Process_1, got %s", wf.ProcessDefinitionID)
	}
}

func TestDeployBPMNRequestBodyTooLarge(t *testing.T) {
	h := setupTestHandler(t)
	r := mux.NewRouter()
	registerTestRoutes(r, h)
	ts := httptest.NewServer(r)
	defer ts.Close()

	before, err := h.engine.ListWorkflows(context.Background())
	if err != nil {
		t.Fatalf("Failed to list workflows before oversized deploy: %v", err)
	}

	oversizedBody := bytes.Repeat([]byte("x"), int(maxBPMNDeployBodyBytes)+1)
	resp, err := http.Post(ts.URL+"/workflows", "application/xml", bytes.NewReader(oversizedBody))
	if err != nil {
		t.Fatalf("Failed to deploy oversized BPMN: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 413 Request Entity Too Large, got %d: %s", resp.StatusCode, string(body))
	}

	after, err := h.engine.ListWorkflows(context.Background())
	if err != nil {
		t.Fatalf("Failed to list workflows after oversized deploy: %v", err)
	}
	if len(after) != len(before) {
		t.Fatalf("Expected oversized deploy to create no workflows, got before=%d after=%d", len(before), len(after))
	}
}

func TestDeployBPMNInvalidXMLReturnsBadRequest(t *testing.T) {
	h := setupTestHandler(t)
	r := mux.NewRouter()
	registerTestRoutes(r, h)
	ts := httptest.NewServer(r)
	defer ts.Close()

	before, err := h.engine.ListWorkflows(context.Background())
	if err != nil {
		t.Fatalf("Failed to list workflows before invalid deploy: %v", err)
	}

	resp, err := http.Post(ts.URL+"/workflows", "application/xml", bytes.NewBufferString("not valid xml"))
	if err != nil {
		t.Fatalf("Failed to deploy invalid BPMN: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected 400 Bad Request, got %d: %s", resp.StatusCode, string(body))
	}
	if !strings.Contains(string(body), application.ErrBPMNValidation.Error()) {
		t.Fatalf("Expected response body to include validation detail, got %q", string(body))
	}

	after, err := h.engine.ListWorkflows(context.Background())
	if err != nil {
		t.Fatalf("Failed to list workflows after invalid deploy: %v", err)
	}
	if len(after) != len(before) {
		t.Fatalf("Expected invalid deploy to create no workflows, got before=%d after=%d", len(before), len(after))
	}
}

func TestModelerRoleCanDeployAndReadProcessesOnly(t *testing.T) {
	h := setupTestHandler(t)
	r := mux.NewRouter()
	registerTestRoutes(r, h)

	modeler := auth.Principal{Subject: "modeler", Roles: []string{auth.RoleArtificialFlowModeler}}
	bpmnXML := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="Definitions_ModelerOnly" targetNamespace="http://flowgo.local/test">
  <bpmn:process id="ModelerOnly" name="Modeler Only" isExecutable="true">
    <bpmn:startEvent id="start"><bpmn:outgoing>f1</bpmn:outgoing></bpmn:startEvent>
    <bpmn:endEvent id="end"><bpmn:incoming>f1</bpmn:incoming></bpmn:endEvent>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`

	deployReq := requestAs(http.MethodPost, "/workflows", strings.NewReader(bpmnXML), modeler)
	deployReq.Header.Set("Content-Type", "application/xml")
	deployRec := httptest.NewRecorder()
	r.ServeHTTP(deployRec, deployReq)
	if deployRec.Code != http.StatusOK {
		t.Fatalf("expected modeler deploy 200, got %d: %s", deployRec.Code, deployRec.Body.String())
	}
	var workflow dto.WorkflowDefinitionResponse
	if err := json.NewDecoder(deployRec.Body).Decode(&workflow); err != nil {
		t.Fatalf("failed to decode workflow: %v", err)
	}

	listReq := requestAs(http.MethodGet, "/workflows", nil, modeler)
	listRec := httptest.NewRecorder()
	r.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected modeler list workflows 200, got %d: %s", listRec.Code, listRec.Body.String())
	}

	getReq := requestAs(http.MethodGet, "/workflows/"+workflow.ID, nil, modeler)
	getRec := httptest.NewRecorder()
	r.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("expected modeler get workflow 200, got %d: %s", getRec.Code, getRec.Body.String())
	}

	startBody, _ := json.Marshal(dto.StartInstanceRequest{WorkflowID: workflow.ID, Context: map[string]any{"source": "modeler-test"}})
	startReq := requestAs(http.MethodPost, "/instances", bytes.NewBuffer(startBody), modeler)
	startReq.Header.Set("Content-Type", "application/json")
	startRec := httptest.NewRecorder()
	r.ServeHTTP(startRec, startReq)
	if startRec.Code != http.StatusForbidden {
		t.Fatalf("expected modeler start instance 403, got %d: %s", startRec.Code, startRec.Body.String())
	}

	instancesReq := requestAs(http.MethodGet, "/instances", nil, modeler)
	instancesRec := httptest.NewRecorder()
	r.ServeHTTP(instancesRec, instancesReq)
	if instancesRec.Code != http.StatusForbidden {
		t.Fatalf("expected modeler list instances 403, got %d: %s", instancesRec.Code, instancesRec.Body.String())
	}
}

func TestPublishSignalAPI(t *testing.T) {
	h := setupTestHandler(t)
	r := mux.NewRouter()
	registerTestRoutes(r, h)
	ts := httptest.NewServer(r)
	defer ts.Close()

	// Deploy Workflow with Signal
	steps := []model.StepDefinition{
		{ID: "start", Type: model.StepTypeStart, Outgoing: []model.Transition{{TargetRef: "catch"}}},
		{
			ID:   "catch",
			Type: model.StepTypeIntermediateCatchEvent,
			Properties: map[string]any{
				"signal_ref": "MySignal",
			},
			Incoming: []string{"start"},
			Outgoing: []model.Transition{{TargetRef: "end"}},
		},
		{ID: "end", Type: model.StepTypeEnd, Incoming: []string{"catch"}},
	}
	wf, _ := h.engine.DeployWorkflow(context.Background(), "Signal Test", steps)
	instance, _ := h.engine.StartInstance(context.Background(), strconv.FormatInt(wf.ID, 10), nil)

	// Publish Signal
	url := ts.URL + "/signals"
	reqBody := dto.PublishSignalRequest{
		SignalName: "MySignal",
		Payload:    map[string]any{"api_trigger": true},
	}
	jsonBody, _ := json.Marshal(reqBody)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		t.Fatalf("Failed to publish signal: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", resp.StatusCode)
	}

	// Verify
	instance, _ = h.engine.GetInstance(context.Background(), instance.ID)
	if instance.Status != model.StatusCompleted {
		t.Errorf("Expected instance completed, got %s", instance.Status)
	}
	if val, ok := instance.Context["api_trigger"]; !ok || val != true {
		t.Errorf("Expected context api_trigger=true, got %v", val)
	}
}

func TestPublishMessageAPI(t *testing.T) {
	h := setupTestHandler(t)
	r := mux.NewRouter()
	registerTestRoutes(r, h)
	ts := httptest.NewServer(r)
	defer ts.Close()

	// Deploy Workflow with Message
	steps := []model.StepDefinition{
		{ID: "start", Type: model.StepTypeStart, Outgoing: []model.Transition{{TargetRef: "catch"}}},
		{
			ID:   "catch",
			Type: model.StepTypeIntermediateCatchEvent,
			Properties: map[string]any{
				"message_ref": "MsgOrderPlaced",
			},
			Incoming: []string{"start"},
			Outgoing: []model.Transition{{TargetRef: "end"}},
		},
		{ID: "end", Type: model.StepTypeEnd, Incoming: []string{"catch"}},
	}
	wf, _ := h.engine.DeployWorkflow(context.Background(), "Message Test", steps)
	instance, _ := h.engine.StartInstance(context.Background(), strconv.FormatInt(wf.ID, 10), nil)

	// Publish Message
	url := ts.URL + "/messages"
	reqBody := dto.PublishMessageRequest{
		MessageName: "MsgOrderPlaced",
		Payload:     map[string]any{"order_id": "999"},
	}
	jsonBody, _ := json.Marshal(reqBody)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		t.Fatalf("Failed to publish message: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", resp.StatusCode)
	}

	// Verify
	instance, _ = h.engine.GetInstance(context.Background(), instance.ID)
	if instance.Status != model.StatusCompleted {
		t.Errorf("Expected instance completed, got %s", instance.Status)
	}
	if val, ok := instance.Context["order_id"]; !ok || val != "999" {
		t.Errorf("Expected context order_id=999, got %v", val)
	}
}

func TestServiceTaskAPI(t *testing.T) {
	h := setupTestHandler(t)
	r := mux.NewRouter()
	registerTestRoutes(r, h)
	ts := httptest.NewServer(r)
	defer ts.Close()

	// Register Service Handler
	h.engine.RegisterHandler("paymentService", func(ctx context.Context, instance *model.WorkflowInstance, step *model.StepDefinition) error {
		if instance.Context == nil {
			instance.Context = make(map[string]any)
		}
		instance.Context["payment_processed"] = true
		instance.Context["amount"] = 100
		return nil
	})

	// Deploy Workflow with Service Task
	steps := []model.StepDefinition{
		{ID: "start", Type: model.StepTypeStart, Outgoing: []model.Transition{{TargetRef: "service"}}},
		{
			ID:             "service",
			Type:           model.StepTypeServiceTask,
			Implementation: "paymentService",
			Incoming:       []string{"start"},
			Outgoing:       []model.Transition{{TargetRef: "end"}},
		},
		{ID: "end", Type: model.StepTypeEnd, Incoming: []string{"service"}},
	}
	wf, err := h.engine.DeployWorkflow(context.Background(), "Service API Test", steps)
	if err != nil {
		t.Fatalf("Failed to deploy: %v", err)
	}

	// Start Instance
	instance, err := h.engine.StartInstance(context.Background(), strconv.FormatInt(wf.ID, 10), nil)
	if err != nil {
		t.Fatalf("Failed to start instance: %v", err)
	}

	// Wait briefly for async execution (if any) or check immediately if sync
	// Service tasks are currently executed synchronously in autoAdvance
	instance, _ = h.engine.GetInstance(context.Background(), instance.ID)

	if instance.Status != model.StatusCompleted {
		t.Errorf("Expected completed, got %s", instance.Status)
	}

	if val, ok := instance.Context["payment_processed"]; !ok || val != true {
		t.Errorf("Expected payment_processed=true, got %v", val)
	}
}

func TestExternalWorkerActivateAndCompleteJobAPI(t *testing.T) {
	h := setupTestHandler(t)
	r := mux.NewRouter()
	registerTestRoutes(r, h)
	ts := httptest.NewServer(r)
	defer ts.Close()

	steps := []model.StepDefinition{
		{ID: "start", Type: model.StepTypeStart, Outgoing: []model.Transition{{TargetRef: "service"}}},
		{
			ID:             "service",
			Type:           model.StepTypeServiceTask,
			Implementation: "external-payment",
			Incoming:       []string{"start"},
			Outgoing:       []model.Transition{{TargetRef: "end"}},
		},
		{ID: "end", Type: model.StepTypeEnd, Incoming: []string{"service"}},
	}

	wf, err := h.engine.DeployWorkflow(context.Background(), "External Worker Test", steps)
	if err != nil {
		t.Fatalf("Failed to deploy: %v", err)
	}

	instance, err := h.engine.StartInstance(context.Background(), strconv.FormatInt(wf.ID, 10), nil)
	if err != nil {
		t.Fatalf("Failed to start instance: %v", err)
	}

	instance, _ = h.engine.GetInstance(context.Background(), instance.ID)
	if instance.Status != model.StatusRunning {
		t.Fatalf("expected running instance before external completion, got %s", instance.Status)
	}

	activateURL := ts.URL + "/jobs/activate"
	activateBody, _ := json.Marshal(dto.ActivateJobsRequest{Type: "external-payment", Worker: "worker-1", MaxJobs: 1})
	activateResp, err := http.Post(activateURL, "application/json", bytes.NewBuffer(activateBody))
	if err != nil {
		t.Fatalf("Failed to activate jobs: %v", err)
	}
	if activateResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(activateResp.Body)
		t.Fatalf("Expected 200 OK, got %d: %s", activateResp.StatusCode, string(body))
	}

	var activated dto.ActivateJobsResponse
	if err := json.NewDecoder(activateResp.Body).Decode(&activated); err != nil {
		t.Fatalf("Failed to decode activate response: %v", err)
	}
	if len(activated.Jobs) != 1 {
		t.Fatalf("expected 1 activated job, got %d", len(activated.Jobs))
	}
	job := activated.Jobs[0]
	if job.State != "ACTIVATED" {
		t.Fatalf("expected ACTIVATED job state, got %s", job.State)
	}

	completeURL := fmt.Sprintf("%s/jobs/%s/complete", ts.URL, job.Key)
	completeBody, _ := json.Marshal(dto.CompleteJobRequest{
		Worker: "worker-1",
		Variables: map[string]any{
			"approved": true,
		},
	})
	completeResp, err := http.Post(completeURL, "application/json", bytes.NewBuffer(completeBody))
	if err != nil {
		t.Fatalf("Failed to complete job: %v", err)
	}
	if completeResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(completeResp.Body)
		t.Fatalf("Expected 200 OK, got %d: %s", completeResp.StatusCode, string(body))
	}

	instance, _ = h.engine.GetInstance(context.Background(), instance.ID)
	if instance.Status != model.StatusCompleted {
		t.Fatalf("expected completed instance after job completion, got %s", instance.Status)
	}
	if val, ok := instance.Context["approved"]; !ok || val != true {
		t.Fatalf("expected approved=true in context, got %v", val)
	}
}

func TestUserTaskOwnershipRequiresEligibleRoleAndClaim(t *testing.T) {
	h := setupTestHandler(t)
	r := mux.NewRouter()
	registerTestRoutes(r, h)

	bpmn := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:flowgo="http://flowgo.com/schema/1.0/bpmn" id="Definitions_TaskOwnership" targetNamespace="http://flowgo.local/test">
  <bpmn:process id="TaskOwnership" name="Task Ownership" isExecutable="true">
    <bpmn:startEvent id="start"><bpmn:outgoing>f1</bpmn:outgoing></bpmn:startEvent>
    <bpmn:userTask id="accountantReview" name="Accountant Review" flowgo:assignee="accountant" flowgo:candidateGroups="accountant"><bpmn:incoming>f1</bpmn:incoming><bpmn:outgoing>f2</bpmn:outgoing></bpmn:userTask>
    <bpmn:endEvent id="end"><bpmn:incoming>f2</bpmn:incoming></bpmn:endEvent>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="accountantReview"/>
    <bpmn:sequenceFlow id="f2" sourceRef="accountantReview" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`

	deployReq := httptest.NewRequest(http.MethodPost, "/workflows", strings.NewReader(bpmn))
	deployReq.Header.Set("Content-Type", "application/xml")
	deployRec := httptest.NewRecorder()
	r.ServeHTTP(deployRec, deployReq)
	if deployRec.Code != http.StatusOK {
		t.Fatalf("expected deploy 200, got %d: %s", deployRec.Code, deployRec.Body.String())
	}
	var workflow dto.WorkflowDefinitionResponse
	if err := json.NewDecoder(deployRec.Body).Decode(&workflow); err != nil {
		t.Fatalf("failed to decode workflow: %v", err)
	}

	startBody, _ := json.Marshal(dto.StartInstanceRequest{WorkflowID: workflow.ID, Context: map[string]any{"source": "ownership-test"}})
	startReq := httptest.NewRequest(http.MethodPost, "/instances", bytes.NewBuffer(startBody))
	startReq.Header.Set("Content-Type", "application/json")
	startRec := httptest.NewRecorder()
	r.ServeHTTP(startRec, startReq)
	if startRec.Code != http.StatusOK {
		t.Fatalf("expected start 200, got %d: %s", startRec.Code, startRec.Body.String())
	}
	var instance dto.WorkflowInstanceResponse
	if err := json.NewDecoder(startRec.Body).Decode(&instance); err != nil {
		t.Fatalf("failed to decode instance: %v", err)
	}

	accountant := auth.Principal{Subject: "accountant", Email: "accountant@flowgo.local", Roles: []string{"accountant"}}
	reviewer := auth.Principal{Subject: "reviewer", Email: "reviewer@flowgo.local", Roles: []string{"reviewer"}}
	modeler := auth.Principal{Subject: "modeler", Roles: []string{auth.RoleArtificialFlowModeler}}
	integrationClient := auth.Principal{Subject: "sdk-client", Roles: []string{auth.RoleArtificialFlowClient}}

	reviewerInstancesReq := requestAs(http.MethodGet, "/instances", nil, reviewer)
	reviewerInstancesRec := httptest.NewRecorder()
	r.ServeHTTP(reviewerInstancesRec, reviewerInstancesReq)
	if reviewerInstancesRec.Code != http.StatusOK {
		t.Fatalf("expected reviewer scoped instances 200, got %d: %s", reviewerInstancesRec.Code, reviewerInstancesRec.Body.String())
	}
	var reviewerInstances []dto.WorkflowInstanceResponse
	if err := json.NewDecoder(reviewerInstancesRec.Body).Decode(&reviewerInstances); err != nil {
		t.Fatalf("failed to decode reviewer instances: %v", err)
	}
	if containsWorkflowInstance(reviewerInstances, instance.ID) {
		t.Fatalf("reviewer should not see accountant-owned instance %s, got %#v", instance.ID, reviewerInstances)
	}

	accountantInstancesReq := requestAs(http.MethodGet, "/instances", nil, accountant)
	accountantInstancesRec := httptest.NewRecorder()
	r.ServeHTTP(accountantInstancesRec, accountantInstancesReq)
	if accountantInstancesRec.Code != http.StatusOK {
		t.Fatalf("expected accountant scoped instances 200, got %d: %s", accountantInstancesRec.Code, accountantInstancesRec.Body.String())
	}
	var accountantInstances []dto.WorkflowInstanceResponse
	if err := json.NewDecoder(accountantInstancesRec.Body).Decode(&accountantInstances); err != nil {
		t.Fatalf("failed to decode accountant instances: %v", err)
	}
	if !containsWorkflowInstance(accountantInstances, instance.ID) {
		t.Fatalf("accountant should see assigned instance %s, got %#v", instance.ID, accountantInstances)
	}

	modelerInstancesReq := requestAs(http.MethodGet, "/instances", nil, modeler)
	modelerInstancesRec := httptest.NewRecorder()
	r.ServeHTTP(modelerInstancesRec, modelerInstancesReq)
	if modelerInstancesRec.Code != http.StatusForbidden {
		t.Fatalf("expected modeler direct instances 403, got %d: %s", modelerInstancesRec.Code, modelerInstancesRec.Body.String())
	}

	reviewerGetReq := requestAs(http.MethodGet, "/instances/"+instance.ID, nil, reviewer)
	reviewerGetRec := httptest.NewRecorder()
	r.ServeHTTP(reviewerGetRec, reviewerGetReq)
	if reviewerGetRec.Code != http.StatusForbidden {
		t.Fatalf("expected reviewer get instance 403, got %d: %s", reviewerGetRec.Code, reviewerGetRec.Body.String())
	}

	reviewerTasksReq := requestAs(http.MethodGet, "/instances/"+instance.ID+"/tasks", nil, reviewer)
	reviewerTasksRec := httptest.NewRecorder()
	r.ServeHTTP(reviewerTasksRec, reviewerTasksReq)
	if reviewerTasksRec.Code != http.StatusForbidden {
		t.Fatalf("expected reviewer list tasks 403, got %d: %s", reviewerTasksRec.Code, reviewerTasksRec.Body.String())
	}

	accountantDirectTasksReq := requestAs(http.MethodGet, "/instances/"+instance.ID+"/tasks", nil, accountant)
	accountantDirectTasksRec := httptest.NewRecorder()
	r.ServeHTTP(accountantDirectTasksRec, accountantDirectTasksReq)
	if accountantDirectTasksRec.Code != http.StatusOK {
		t.Fatalf("expected accountant direct task list 200, got %d: %s", accountantDirectTasksRec.Code, accountantDirectTasksRec.Body.String())
	}

	tasksReq := requestAsInbox(http.MethodGet, "/inbox/instances/"+instance.ID+"/tasks", nil, integrationClient, accountant)
	tasksRec := httptest.NewRecorder()
	r.ServeHTTP(tasksRec, tasksReq)
	if tasksRec.Code != http.StatusOK {
		t.Fatalf("expected accountant inbox list tasks 200, got %d: %s", tasksRec.Code, tasksRec.Body.String())
	}
	var tasks dto.ListUserTasksResponse
	if err := json.NewDecoder(tasksRec.Body).Decode(&tasks); err != nil {
		t.Fatalf("failed to decode tasks: %v", err)
	}
	if len(tasks.Tasks) != 1 {
		t.Fatalf("expected one user task, got %#v", tasks.Tasks)
	}
	if !tasks.Tasks[0].CanClaim || tasks.Tasks[0].CanComplete {
		t.Fatalf("accountant should be able to claim but not complete before claiming: %#v", tasks.Tasks[0])
	}
	executionID := tasks.Tasks[0].ExecutionID

	reviewerClaim := requestAs(http.MethodPost, "/instances/"+instance.ID+"/tasks/"+executionID+"/claim", nil, reviewer)
	reviewerClaimRec := httptest.NewRecorder()
	r.ServeHTTP(reviewerClaimRec, reviewerClaim)
	if reviewerClaimRec.Code != http.StatusForbidden {
		t.Fatalf("expected reviewer claim 403, got %d: %s", reviewerClaimRec.Code, reviewerClaimRec.Body.String())
	}

	accountantDirectClaim := requestAs(http.MethodPost, "/instances/"+instance.ID+"/tasks/"+executionID+"/claim", nil, accountant)
	accountantDirectClaimRec := httptest.NewRecorder()
	r.ServeHTTP(accountantDirectClaimRec, accountantDirectClaim)
	if accountantDirectClaimRec.Code != http.StatusForbidden {
		t.Fatalf("expected accountant direct claim 403, got %d: %s", accountantDirectClaimRec.Code, accountantDirectClaimRec.Body.String())
	}

	accountantClaim := requestAsInbox(http.MethodPost, "/inbox/instances/"+instance.ID+"/tasks/"+executionID+"/claim", nil, integrationClient, accountant)
	accountantClaimRec := httptest.NewRecorder()
	r.ServeHTTP(accountantClaimRec, accountantClaim)
	if accountantClaimRec.Code != http.StatusOK {
		t.Fatalf("expected accountant inbox claim 200, got %d: %s", accountantClaimRec.Code, accountantClaimRec.Body.String())
	}

	reviewerComplete := requestAs(http.MethodPost, "/instances/"+instance.ID+"/tasks/"+executionID+"/complete", nil, reviewer)
	reviewerCompleteRec := httptest.NewRecorder()
	r.ServeHTTP(reviewerCompleteRec, reviewerComplete)
	if reviewerCompleteRec.Code != http.StatusForbidden {
		t.Fatalf("expected reviewer complete 403, got %d: %s", reviewerCompleteRec.Code, reviewerCompleteRec.Body.String())
	}

	accountantDirectComplete := requestAs(http.MethodPost, "/instances/"+instance.ID+"/tasks/"+executionID+"/complete", nil, accountant)
	accountantDirectCompleteRec := httptest.NewRecorder()
	r.ServeHTTP(accountantDirectCompleteRec, accountantDirectComplete)
	if accountantDirectCompleteRec.Code != http.StatusForbidden {
		t.Fatalf("expected accountant direct complete 403, got %d: %s", accountantDirectCompleteRec.Code, accountantDirectCompleteRec.Body.String())
	}

	accountantComplete := requestAsInbox(http.MethodPost, "/inbox/instances/"+instance.ID+"/tasks/"+executionID+"/complete", nil, integrationClient, accountant)
	accountantCompleteRec := httptest.NewRecorder()
	r.ServeHTTP(accountantCompleteRec, accountantComplete)
	if accountantCompleteRec.Code != http.StatusOK {
		t.Fatalf("expected accountant inbox complete 200, got %d: %s", accountantCompleteRec.Code, accountantCompleteRec.Body.String())
	}

	activeTasksReq := requestAsInbox(http.MethodGet, "/inbox/instances/"+instance.ID+"/tasks", nil, integrationClient, accountant)
	activeTasksRec := httptest.NewRecorder()
	r.ServeHTTP(activeTasksRec, activeTasksReq)
	if activeTasksRec.Code != http.StatusOK {
		t.Fatalf("expected inbox active task history 200, got %d: %s", activeTasksRec.Code, activeTasksRec.Body.String())
	}
	var activeTasks dto.ListUserTasksResponse
	if err := json.NewDecoder(activeTasksRec.Body).Decode(&activeTasks); err != nil {
		t.Fatalf("failed to decode active tasks: %v", err)
	}
	if len(activeTasks.Tasks) != 0 {
		t.Fatalf("completed user task should be hidden from active task list, got %#v", activeTasks.Tasks)
	}

	historyReq := requestAsInbox(http.MethodGet, "/inbox/instances/"+instance.ID+"/tasks?includeCompleted=true", nil, integrationClient, accountant)
	historyRec := httptest.NewRecorder()
	r.ServeHTTP(historyRec, historyReq)
	if historyRec.Code != http.StatusOK {
		t.Fatalf("expected inbox completed task history 200, got %d: %s", historyRec.Code, historyRec.Body.String())
	}
	var history dto.ListUserTasksResponse
	if err := json.NewDecoder(historyRec.Body).Decode(&history); err != nil {
		t.Fatalf("failed to decode completed task history: %v", err)
	}
	if len(history.Tasks) != 1 {
		t.Fatalf("expected one completed task history row, got %#v", history.Tasks)
	}
	if history.Tasks[0].State != "COMPLETED" {
		t.Fatalf("expected completed task state, got %#v", history.Tasks[0])
	}
	if history.Tasks[0].ClaimedBy == "" {
		t.Fatalf("expected completed task actor to be recorded, got %#v", history.Tasks[0])
	}

	completedHistoryReq := requestAs(http.MethodGet, "/instances/history/completed", nil, accountant)
	completedHistoryRec := httptest.NewRecorder()
	r.ServeHTTP(completedHistoryRec, completedHistoryReq)
	if completedHistoryRec.Code != http.StatusOK {
		t.Fatalf("expected accountant direct completed history 200, got %d: %s", completedHistoryRec.Code, completedHistoryRec.Body.String())
	}
	var completedInstances []dto.WorkflowInstanceResponse
	if err := json.NewDecoder(completedHistoryRec.Body).Decode(&completedInstances); err != nil {
		t.Fatalf("failed to decode completed instance history: %v", err)
	}
	foundCompletedInstance := false
	for _, completedInstance := range completedInstances {
		if completedInstance.ID == instance.ID && completedInstance.Status == string(model.StatusCompleted) {
			foundCompletedInstance = true
			break
		}
	}
	if !foundCompletedInstance {
		t.Fatalf("expected completed instance %s in accountant history, got %#v", instance.ID, completedInstances)
	}

	reviewerHistoryReq := requestAs(http.MethodGet, "/instances/history/completed", nil, reviewer)
	reviewerHistoryRec := httptest.NewRecorder()
	r.ServeHTTP(reviewerHistoryRec, reviewerHistoryReq)
	if reviewerHistoryRec.Code != http.StatusOK {
		t.Fatalf("expected reviewer direct completed history 200, got %d: %s", reviewerHistoryRec.Code, reviewerHistoryRec.Body.String())
	}
	var reviewerCompletedInstances []dto.WorkflowInstanceResponse
	if err := json.NewDecoder(reviewerHistoryRec.Body).Decode(&reviewerCompletedInstances); err != nil {
		t.Fatalf("failed to decode reviewer completed history: %v", err)
	}
	if containsWorkflowInstance(reviewerCompletedInstances, instance.ID) {
		t.Fatalf("reviewer should not see accountant-owned completed instance %s, got %#v", instance.ID, reviewerCompletedInstances)
	}

	adminHistoryReq := requestAs(http.MethodGet, "/instances/history/completed", nil, auth.Principal{Subject: "admin", Roles: []string{auth.RoleArtificialFlowAdmin}})
	adminHistoryRec := httptest.NewRecorder()
	r.ServeHTTP(adminHistoryRec, adminHistoryReq)
	if adminHistoryRec.Code != http.StatusOK {
		t.Fatalf("expected admin completed history 200, got %d: %s", adminHistoryRec.Code, adminHistoryRec.Body.String())
	}
	var adminCompletedInstances []dto.WorkflowInstanceResponse
	if err := json.NewDecoder(adminHistoryRec.Body).Decode(&adminCompletedInstances); err != nil {
		t.Fatalf("failed to decode admin completed history: %v", err)
	}
	if !containsWorkflowInstance(adminCompletedInstances, instance.ID) {
		t.Fatalf("admin should see completed instance %s, got %#v", instance.ID, adminCompletedInstances)
	}
}

func TestTaskInboxRequiresClientIntegrationAndExcludesAdminClientHistory(t *testing.T) {
	h, repo := setupTestHandlerWithRepository(t, iam.DeploymentConfig{
		Mode:                iam.DeploymentModeExternal,
		ProviderName:        "Corporate OIDC",
		ConfigurationSource: "test",
	})
	r := mux.NewRouter()
	registerTestRoutes(r, h)

	bpmn := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:flowgo="http://flowgo.com/schema/1.0/bpmn" id="Definitions_Inbox" targetNamespace="http://flowgo.local/test">
  <bpmn:process id="InboxTask" name="Inbox Task" isExecutable="true">
    <bpmn:startEvent id="start"><bpmn:outgoing>f1</bpmn:outgoing></bpmn:startEvent>
    <bpmn:userTask id="accountantReview" name="Accountant Review" flowgo:candidateGroups="accountant"><bpmn:incoming>f1</bpmn:incoming><bpmn:outgoing>f2</bpmn:outgoing></bpmn:userTask>
    <bpmn:endEvent id="end"><bpmn:incoming>f2</bpmn:incoming></bpmn:endEvent>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="accountantReview"/>
    <bpmn:sequenceFlow id="f2" sourceRef="accountantReview" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`

	deployReq := httptest.NewRequest(http.MethodPost, "/workflows", strings.NewReader(bpmn))
	deployReq.Header.Set("Content-Type", "application/xml")
	deployRec := httptest.NewRecorder()
	r.ServeHTTP(deployRec, deployReq)
	if deployRec.Code != http.StatusOK {
		t.Fatalf("expected deploy 200, got %d: %s", deployRec.Code, deployRec.Body.String())
	}
	var workflow dto.WorkflowDefinitionResponse
	if err := json.NewDecoder(deployRec.Body).Decode(&workflow); err != nil {
		t.Fatalf("failed to decode workflow: %v", err)
	}

	startBody, _ := json.Marshal(dto.StartInstanceRequest{WorkflowID: workflow.ID, Context: map[string]any{"source": "inbox-test"}})
	startReq := httptest.NewRequest(http.MethodPost, "/instances", bytes.NewBuffer(startBody))
	startReq.Header.Set("Content-Type", "application/json")
	startRec := httptest.NewRecorder()
	r.ServeHTTP(startRec, startReq)
	if startRec.Code != http.StatusOK {
		t.Fatalf("expected start 200, got %d: %s", startRec.Code, startRec.Body.String())
	}
	var instance dto.WorkflowInstanceResponse
	if err := json.NewDecoder(startRec.Body).Decode(&instance); err != nil {
		t.Fatalf("failed to decode instance: %v", err)
	}
	instanceKey, err := strconv.ParseInt(instance.ID, 10, 64)
	if err != nil {
		t.Fatalf("failed to parse instance id: %v", err)
	}
	createdJobs, err := repo.ListJobsByProcessInstanceAndType(context.Background(), instanceKey, application.UserTaskJobType)
	if err != nil || len(createdJobs) != 1 {
		t.Fatalf("expected one newly written user task, got jobs=%#v err=%v", createdJobs, err)
	}
	if createdJobs[0].Type != application.UserTaskJobType {
		t.Fatalf("new user-task writes must use %q, got %q", application.UserTaskJobType, createdJobs[0].Type)
	}
	if err := repo.DB.Model(&model.Job{}).
		Where("key = ?", createdJobs[0].Key).
		Update("type", application.LegacyUserTaskJobType).Error; err != nil {
		t.Fatalf("failed to simulate a legacy persisted user task: %v", err)
	}

	integrationClient := auth.Principal{Subject: "sdk-client", Roles: []string{auth.RoleArtificialFlowClient}}
	actingUser := auth.Principal{
		Subject: "accountant",
		Email:   "accountant@flowgo.local",
		Roles:   []string{"accountant"},
		Claims:  map[string]any{"username": "accountant"},
	}

	nonClientReq := requestAsInbox(http.MethodGet, "/inbox", nil, auth.Principal{Subject: "admin", Roles: []string{auth.RoleArtificialFlowAdmin}}, actingUser)
	nonClientRec := httptest.NewRecorder()
	r.ServeHTTP(nonClientRec, nonClientReq)
	if nonClientRec.Code != http.StatusForbidden {
		t.Fatalf("expected non-client integration principal to be denied, got %d: %s", nonClientRec.Code, nonClientRec.Body.String())
	}

	adminIntegrationReq := requestAsInbox(http.MethodGet, "/inbox", nil, auth.Principal{Subject: "admin-client", Roles: []string{auth.RoleArtificialFlowAdmin, auth.RoleArtificialFlowClient}}, actingUser)
	adminIntegrationRec := httptest.NewRecorder()
	r.ServeHTTP(adminIntegrationRec, adminIntegrationReq)
	if adminIntegrationRec.Code != http.StatusForbidden {
		t.Fatalf("expected admin integration principal to be denied, got %d: %s", adminIntegrationRec.Code, adminIntegrationRec.Body.String())
	}

	missingActingReq := requestAs(http.MethodGet, "/inbox", nil, integrationClient)
	missingActingRec := httptest.NewRecorder()
	r.ServeHTTP(missingActingRec, missingActingReq)
	if missingActingRec.Code != http.StatusBadRequest {
		t.Fatalf("expected missing acting user to be rejected, got %d: %s", missingActingRec.Code, missingActingRec.Body.String())
	}

	actingAdminReq := requestAsInbox(http.MethodGet, "/inbox", nil, integrationClient, auth.Principal{Subject: "admin", Roles: []string{auth.RoleArtificialFlowAdmin, "accountant"}})
	actingAdminRec := httptest.NewRecorder()
	r.ServeHTTP(actingAdminRec, actingAdminReq)
	if actingAdminRec.Code != http.StatusForbidden {
		t.Fatalf("expected admin acting user to be denied, got %d: %s", actingAdminRec.Code, actingAdminRec.Body.String())
	}

	inboxReq := requestAsInbox(http.MethodGet, "/inbox", nil, integrationClient, actingUser)
	inboxRec := httptest.NewRecorder()
	r.ServeHTTP(inboxRec, inboxReq)
	if inboxRec.Code != http.StatusOK {
		t.Fatalf("expected SDK client inbox 200 for acting user, got %d: %s", inboxRec.Code, inboxRec.Body.String())
	}
	var inboxInstances []dto.WorkflowInstanceResponse
	if err := json.NewDecoder(inboxRec.Body).Decode(&inboxInstances); err != nil {
		t.Fatalf("failed to decode inbox response: %v", err)
	}
	if !containsWorkflowInstance(inboxInstances, instance.ID) {
		t.Fatalf("expected inbox to contain instance %s, got %#v", instance.ID, inboxInstances)
	}

	tasksReq := requestAsInbox(http.MethodGet, "/inbox/instances/"+instance.ID+"/tasks", nil, integrationClient, actingUser)
	tasksRec := httptest.NewRecorder()
	r.ServeHTTP(tasksRec, tasksReq)
	if tasksRec.Code != http.StatusOK {
		t.Fatalf("expected inbox tasks 200, got %d: %s", tasksRec.Code, tasksRec.Body.String())
	}
	var tasks dto.ListUserTasksResponse
	if err := json.NewDecoder(tasksRec.Body).Decode(&tasks); err != nil {
		t.Fatalf("failed to decode inbox tasks: %v", err)
	}
	if len(tasks.Tasks) != 1 {
		t.Fatalf("expected one inbox task, got %#v", tasks.Tasks)
	}
	executionID := tasks.Tasks[0].ExecutionID

	claimReq := requestAsInbox(http.MethodPost, "/inbox/instances/"+instance.ID+"/tasks/"+executionID+"/claim", nil, integrationClient, actingUser)
	claimRec := httptest.NewRecorder()
	r.ServeHTTP(claimRec, claimReq)
	if claimRec.Code != http.StatusOK {
		t.Fatalf("expected inbox claim 200, got %d: %s", claimRec.Code, claimRec.Body.String())
	}

	completeReq := requestAsInbox(http.MethodPost, "/inbox/instances/"+instance.ID+"/tasks/"+executionID+"/complete", nil, integrationClient, actingUser)
	completeRec := httptest.NewRecorder()
	r.ServeHTTP(completeRec, completeReq)
	if completeRec.Code != http.StatusOK {
		t.Fatalf("expected inbox complete 200, got %d: %s", completeRec.Code, completeRec.Body.String())
	}

	historyReq := requestAsInbox(http.MethodGet, "/inbox/history", nil, integrationClient, actingUser)
	historyRec := httptest.NewRecorder()
	r.ServeHTTP(historyRec, historyReq)
	if historyRec.Code != http.StatusOK {
		t.Fatalf("expected inbox history 200, got %d: %s", historyRec.Code, historyRec.Body.String())
	}
	var history []dto.WorkflowInstanceResponse
	if err := json.NewDecoder(historyRec.Body).Decode(&history); err != nil {
		t.Fatalf("failed to decode inbox history: %v", err)
	}
	if !containsWorkflowInstance(history, instance.ID) {
		t.Fatalf("expected my completed history to contain instance %s, got %#v", instance.ID, history)
	}
}

func TestTaskInboxFiltersSiblingTasksForActingUser(t *testing.T) {
	h := setupTestHandler(t)
	r := mux.NewRouter()
	registerTestRoutes(r, h)

	steps := []model.StepDefinition{
		{ID: "start", Type: model.StepTypeStart, Outgoing: []model.Transition{{TargetRef: "split"}}},
		{ID: "split", Type: model.StepTypeGatewayParallel, Outgoing: []model.Transition{
			{TargetRef: "accountantReview"},
			{TargetRef: "reviewerReview"},
		}},
		{
			ID:         "accountantReview",
			Type:       model.StepTypeUserTask,
			Properties: map[string]any{"candidate_groups": "accountant"},
			Outgoing:   []model.Transition{{TargetRef: "join"}},
		},
		{
			ID:         "reviewerReview",
			Type:       model.StepTypeUserTask,
			Properties: map[string]any{"candidate_groups": "reviewer"},
			Outgoing:   []model.Transition{{TargetRef: "join"}},
		},
		{ID: "join", Type: model.StepTypeGatewayParallel, Incoming: []string{"accountantReview", "reviewerReview"}, Outgoing: []model.Transition{{TargetRef: "end"}}},
		{ID: "end", Type: model.StepTypeEnd},
	}
	wf, err := h.engine.DeployWorkflow(context.Background(), "Inbox Task Filtering", steps)
	if err != nil {
		t.Fatalf("failed to deploy workflow: %v", err)
	}
	instance, err := h.engine.StartInstance(context.Background(), strconv.FormatInt(wf.ID, 10), map[string]any{"source": "task-filter-test"})
	if err != nil {
		t.Fatalf("failed to start instance: %v", err)
	}

	integrationClient := auth.Principal{Subject: "sdk-client", Roles: []string{auth.RoleArtificialFlowClient}}
	actingAccountant := auth.Principal{
		Subject: "accountant",
		Email:   "accountant@flowgo.local",
		Roles:   []string{"accountant"},
		Claims:  map[string]any{"username": "accountant"},
	}

	tasksReq := requestAsInbox(http.MethodGet, "/inbox/instances/"+instance.ID+"/tasks", nil, integrationClient, actingAccountant)
	tasksRec := httptest.NewRecorder()
	r.ServeHTTP(tasksRec, tasksReq)
	if tasksRec.Code != http.StatusOK {
		t.Fatalf("expected inbox tasks 200, got %d: %s", tasksRec.Code, tasksRec.Body.String())
	}
	var tasks dto.ListUserTasksResponse
	if err := json.NewDecoder(tasksRec.Body).Decode(&tasks); err != nil {
		t.Fatalf("failed to decode inbox tasks: %v", err)
	}
	if len(tasks.Tasks) != 1 || tasks.Tasks[0].ElementID != "accountantReview" {
		t.Fatalf("expected only accountant task, got %#v", tasks.Tasks)
	}
	if containsUserTask(tasks.Tasks, "reviewerReview") {
		t.Fatalf("reviewer task should not be exposed to accountant, got %#v", tasks.Tasks)
	}

	instanceReq := requestAsInbox(http.MethodGet, "/inbox/instances/"+instance.ID, nil, integrationClient, actingAccountant)
	instanceRec := httptest.NewRecorder()
	r.ServeHTTP(instanceRec, instanceReq)
	if instanceRec.Code != http.StatusOK {
		t.Fatalf("expected inbox instance 200, got %d: %s", instanceRec.Code, instanceRec.Body.String())
	}
	var response dto.WorkflowInstanceResponse
	if err := json.NewDecoder(instanceRec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode inbox instance: %v", err)
	}
	embeddedTasks := userTasksFromInstance(response)
	if len(embeddedTasks) != 1 || embeddedTasks[0].ElementID != "accountantReview" {
		t.Fatalf("expected only accountant embedded task, got %#v", embeddedTasks)
	}
	if containsUserTask(embeddedTasks, "reviewerReview") {
		t.Fatalf("reviewer embedded task should not be exposed to accountant, got %#v", embeddedTasks)
	}
}

func TestTaskInboxHistoryFindsSparseOlderCompletedTask(t *testing.T) {
	h, repo := setupTestHandlerWithRepository(t, iam.DeploymentConfig{
		Mode:                iam.DeploymentModeExternal,
		ProviderName:        "Corporate OIDC",
		ConfigurationSource: "test",
	})
	r := mux.NewRouter()
	registerTestRoutes(r, h)

	ctx := context.Background()
	now := time.Now().UTC()
	target := &model.ProcessInstance{
		Key:                  9001,
		ID:                   "target",
		ProcessDefinitionKey: 100,
		Version:              1,
		State:                "COMPLETED",
		CreatedAt:            now.Add(-3 * time.Hour),
		EndTime:              now.Add(-2 * time.Hour),
	}
	if err := repo.CreateProcessInstance(ctx, target); err != nil {
		t.Fatalf("failed to seed target process instance: %v", err)
	}
	if err := repo.CreateJob(ctx, &model.Job{
		Key:                19001,
		Type:               application.UserTaskJobType,
		ProcessInstanceKey: target.Key,
		ElementInstanceKey: 29001,
		ElementID:          "accountantReview",
		Worker:             " accountant@flowgo.local ",
		Retries:            1,
		State:              "COMPLETED",
		CreatedAt:          target.CreatedAt,
		UpdatedAt:          target.EndTime,
	}); err != nil {
		t.Fatalf("failed to seed target job: %v", err)
	}

	for i := 0; i < 101; i++ {
		key := int64(9100 + i)
		endTime := now.Add(time.Duration(i) * time.Minute)
		instance := &model.ProcessInstance{
			Key:                  key,
			ID:                   fmt.Sprintf("other-%d", i),
			ProcessDefinitionKey: 100,
			Version:              1,
			State:                "COMPLETED",
			CreatedAt:            endTime.Add(-time.Hour),
			EndTime:              endTime,
		}
		if err := repo.CreateProcessInstance(ctx, instance); err != nil {
			t.Fatalf("failed to seed other process instance %d: %v", i, err)
		}
		if err := repo.CreateJob(ctx, &model.Job{
			Key:                int64(19100 + i),
			Type:               application.UserTaskJobType,
			ProcessInstanceKey: key,
			ElementInstanceKey: int64(29100 + i),
			ElementID:          "reviewerReview",
			Worker:             "reviewer@flowgo.local",
			Retries:            1,
			State:              "COMPLETED",
			CreatedAt:          instance.CreatedAt,
			UpdatedAt:          instance.EndTime,
		}); err != nil {
			t.Fatalf("failed to seed other job %d: %v", i, err)
		}
	}

	integrationClient := auth.Principal{Subject: "sdk-client", Roles: []string{auth.RoleArtificialFlowClient}}
	actingAccountant := auth.Principal{
		Subject: "accountant",
		Email:   "accountant@flowgo.local",
		Roles:   []string{"accountant"},
		Claims:  map[string]any{"username": "accountant"},
	}
	historyReq := requestAsInbox(http.MethodGet, "/inbox/history?limit=1", nil, integrationClient, actingAccountant)
	historyRec := httptest.NewRecorder()
	r.ServeHTTP(historyRec, historyReq)
	if historyRec.Code != http.StatusOK {
		t.Fatalf("expected sparse inbox history 200, got %d: %s", historyRec.Code, historyRec.Body.String())
	}
	var history []dto.WorkflowInstanceResponse
	if err := json.NewDecoder(historyRec.Body).Decode(&history); err != nil {
		t.Fatalf("failed to decode sparse inbox history: %v", err)
	}
	if len(history) != 1 || history[0].ID != strconv.FormatInt(target.Key, 10) {
		t.Fatalf("expected sparse history to return target instance %d, got %#v", target.Key, history)
	}
}

func TestActingPrincipalFromRequestPrefersCanonicalHeadersAndCanonicalizesRoles(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/inbox", nil)
	req.Header.Set(actingSubjectHeader, "canonical-subject")
	req.Header.Set(legacyActingSubjectHeader, "legacy-subject")
	req.Header.Set(actingUsernameHeader, "canonical-user")
	req.Header.Set(legacyActingUsernameHeader, "legacy-user")
	req.Header.Set(actingEmailHeader, "canonical@artificialflow.io")
	req.Header.Set(legacyActingEmailHeader, "legacy@flowgo.local")
	req.Header.Set(actingNameHeader, "Canonical User")
	req.Header.Set(legacyActingNameHeader, "Legacy User")
	req.Header.Set(actingRolesHeader, "flowgo admin, Finance Reviewer")
	req.Header.Set(legacyActingRolesHeader, "legacy-only")

	principal, err := actingPrincipalFromRequest(req)
	if err != nil {
		t.Fatalf("acting principal: %v", err)
	}
	if principal.Subject != "canonical-subject" || principal.Email != "canonical@artificialflow.io" || principal.Name != "Canonical User" {
		t.Fatalf("expected canonical identity headers, got %#v", principal)
	}
	if username := principal.Claims["username"]; username != "canonical-user" {
		t.Fatalf("expected canonical username, got %#v", username)
	}
	expectedRoles := []string{auth.RoleArtificialFlowAdmin, "Finance Reviewer"}
	if !reflect.DeepEqual(principal.Roles, expectedRoles) {
		t.Fatalf("expected canonical roles %#v, got %#v", expectedRoles, principal.Roles)
	}
}

func TestActingPrincipalFromRequestAcceptsLegacyHeaders(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/inbox", nil)
	req.Header.Set(legacyActingUsernameHeader, "legacy-user")
	req.Header.Set(legacyActingRolesHeader, "flowgo modeler")

	principal, err := actingPrincipalFromRequest(req)
	if err != nil {
		t.Fatalf("acting principal: %v", err)
	}
	if principal.Subject != "legacy-user" {
		t.Fatalf("expected legacy username fallback, got %#v", principal)
	}
	if !reflect.DeepEqual(principal.Roles, []string{auth.RoleArtificialFlowModeler}) {
		t.Fatalf("expected canonicalized legacy roles, got %#v", principal.Roles)
	}
}

func requestAs(method, target string, body io.Reader, principal auth.Principal) *http.Request {
	req := httptest.NewRequest(method, target, body)
	return req.WithContext(auth.WithPrincipal(req.Context(), principal))
}

func requestAsInbox(method, target string, body io.Reader, integrationPrincipal auth.Principal, actingPrincipal auth.Principal) *http.Request {
	req := requestAs(method, target, body, integrationPrincipal)
	req.Header.Set(actingSubjectHeader, actingPrincipal.Subject)
	req.Header.Set(actingEmailHeader, actingPrincipal.Email)
	req.Header.Set(actingNameHeader, actingPrincipal.Name)
	req.Header.Set(actingRolesHeader, strings.Join(actingPrincipal.Roles, ","))
	if username, ok := actingPrincipal.Claims["username"].(string); ok {
		req.Header.Set(actingUsernameHeader, username)
	}
	return req
}

func containsWorkflowInstance(instances []dto.WorkflowInstanceResponse, id string) bool {
	for _, instance := range instances {
		if instance.ID == id {
			return true
		}
	}
	return false
}

func containsUserTask(tasks []dto.UserTaskResponse, elementID string) bool {
	for _, task := range tasks {
		if task.ElementID == elementID {
			return true
		}
	}
	return false
}

func userTasksFromInstance(instance dto.WorkflowInstanceResponse) []dto.UserTaskResponse {
	tasks := make([]dto.UserTaskResponse, 0)
	for _, execution := range instance.Executions {
		if execution.Task != nil {
			tasks = append(tasks, *execution.Task)
		}
	}
	return tasks
}

func TestExternalWorkerFailJobAPI(t *testing.T) {
	h := setupTestHandler(t)
	r := mux.NewRouter()
	registerTestRoutes(r, h)
	ts := httptest.NewServer(r)
	defer ts.Close()

	steps := []model.StepDefinition{
		{ID: "start", Type: model.StepTypeStart, Outgoing: []model.Transition{{TargetRef: "service"}}},
		{
			ID:             "service",
			Type:           model.StepTypeServiceTask,
			Implementation: "external-fail",
			Incoming:       []string{"start"},
			Outgoing:       []model.Transition{{TargetRef: "end"}},
		},
		{ID: "end", Type: model.StepTypeEnd, Incoming: []string{"service"}},
	}

	wf, err := h.engine.DeployWorkflow(context.Background(), "External Worker Fail Test", steps)
	if err != nil {
		t.Fatalf("Failed to deploy: %v", err)
	}

	instance, err := h.engine.StartInstance(context.Background(), strconv.FormatInt(wf.ID, 10), nil)
	if err != nil {
		t.Fatalf("Failed to start instance: %v", err)
	}

	activateURL := ts.URL + "/jobs/activate"
	activateBody, _ := json.Marshal(dto.ActivateJobsRequest{Type: "external-fail", Worker: "worker-1", MaxJobs: 1})
	activateResp, err := http.Post(activateURL, "application/json", bytes.NewBuffer(activateBody))
	if err != nil {
		t.Fatalf("Failed to activate jobs: %v", err)
	}

	var activated dto.ActivateJobsResponse
	if err := json.NewDecoder(activateResp.Body).Decode(&activated); err != nil {
		t.Fatalf("Failed to decode activate response: %v", err)
	}
	if len(activated.Jobs) != 1 {
		t.Fatalf("expected 1 activated job, got %d", len(activated.Jobs))
	}
	job := activated.Jobs[0]

	retries := 1
	failURL := fmt.Sprintf("%s/jobs/%s/fail", ts.URL, job.Key)
	failBody, _ := json.Marshal(dto.FailJobRequest{Worker: "worker-1", ErrorMessage: "temporary error", Retries: &retries})
	failResp, err := http.Post(failURL, "application/json", bytes.NewBuffer(failBody))
	if err != nil {
		t.Fatalf("Failed to fail job: %v", err)
	}
	if failResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(failResp.Body)
		t.Fatalf("Expected 200 OK, got %d: %s", failResp.StatusCode, string(body))
	}

	reactivateBody, _ := json.Marshal(dto.ActivateJobsRequest{Type: "external-fail", Worker: "worker-2", MaxJobs: 1})
	reactivateResp, err := http.Post(activateURL, "application/json", bytes.NewBuffer(reactivateBody))
	if err != nil {
		t.Fatalf("Failed to re-activate jobs: %v", err)
	}

	var reactivated dto.ActivateJobsResponse
	if err := json.NewDecoder(reactivateResp.Body).Decode(&reactivated); err != nil {
		t.Fatalf("Failed to decode re-activate response: %v", err)
	}
	if len(reactivated.Jobs) != 1 {
		t.Fatalf("expected 1 re-activated job, got %d", len(reactivated.Jobs))
	}

	instance, _ = h.engine.GetInstance(context.Background(), instance.ID)
	if instance.Status != model.StatusRunning {
		t.Fatalf("expected running instance after failed job, got %s", instance.Status)
	}
}

func TestExternalWorkerJobReactivationAfterLockExpiry(t *testing.T) {
	h := setupTestHandler(t)
	r := mux.NewRouter()
	registerTestRoutes(r, h)
	ts := httptest.NewServer(r)
	defer ts.Close()

	steps := []model.StepDefinition{
		{ID: "start", Type: model.StepTypeStart, Outgoing: []model.Transition{{TargetRef: "service"}}},
		{
			ID:             "service",
			Type:           model.StepTypeServiceTask,
			Implementation: "external-lock-expiry",
			Incoming:       []string{"start"},
			Outgoing:       []model.Transition{{TargetRef: "end"}},
		},
		{ID: "end", Type: model.StepTypeEnd, Incoming: []string{"service"}},
	}

	wf, err := h.engine.DeployWorkflow(context.Background(), "External Worker Lock Expiry", steps)
	if err != nil {
		t.Fatalf("Failed to deploy: %v", err)
	}

	instance, err := h.engine.StartInstance(context.Background(), strconv.FormatInt(wf.ID, 10), nil)
	if err != nil {
		t.Fatalf("Failed to start instance: %v", err)
	}

	activateURL := ts.URL + "/jobs/activate"
	activateBody, _ := json.Marshal(dto.ActivateJobsRequest{Type: "external-lock-expiry", Worker: "worker-1", MaxJobs: 1, LockDurationMs: 80})
	activateResp, err := http.Post(activateURL, "application/json", bytes.NewBuffer(activateBody))
	if err != nil {
		t.Fatalf("Failed to activate jobs: %v", err)
	}

	var firstActivation dto.ActivateJobsResponse
	if err := json.NewDecoder(activateResp.Body).Decode(&firstActivation); err != nil {
		t.Fatalf("Failed to decode activation response: %v", err)
	}
	if len(firstActivation.Jobs) != 1 {
		t.Fatalf("expected 1 activated job, got %d", len(firstActivation.Jobs))
	}

	time.Sleep(150 * time.Millisecond)

	reactivateBody, _ := json.Marshal(dto.ActivateJobsRequest{Type: "external-lock-expiry", Worker: "worker-2", MaxJobs: 1, LockDurationMs: 200})
	reactivateResp, err := http.Post(activateURL, "application/json", bytes.NewBuffer(reactivateBody))
	if err != nil {
		t.Fatalf("Failed to re-activate jobs: %v", err)
	}

	var secondActivation dto.ActivateJobsResponse
	if err := json.NewDecoder(reactivateResp.Body).Decode(&secondActivation); err != nil {
		t.Fatalf("Failed to decode re-activation response: %v", err)
	}
	if len(secondActivation.Jobs) != 1 {
		t.Fatalf("expected 1 re-activated job, got %d", len(secondActivation.Jobs))
	}

	if secondActivation.Jobs[0].Key != firstActivation.Jobs[0].Key {
		t.Fatalf("expected same job key to be re-activated")
	}
	if secondActivation.Jobs[0].Worker != "worker-2" {
		t.Fatalf("expected worker-2 to own re-activated job, got %s", secondActivation.Jobs[0].Worker)
	}

	completeURL := fmt.Sprintf("%s/jobs/%s/complete", ts.URL, secondActivation.Jobs[0].Key)
	completeBody, _ := json.Marshal(dto.CompleteJobRequest{Worker: "worker-2"})
	completeResp, err := http.Post(completeURL, "application/json", bytes.NewBuffer(completeBody))
	if err != nil {
		t.Fatalf("Failed to complete re-activated job: %v", err)
	}
	if completeResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(completeResp.Body)
		t.Fatalf("Expected 200 OK, got %d: %s", completeResp.StatusCode, string(body))
	}

	instance, _ = h.engine.GetInstance(context.Background(), instance.ID)
	if instance.Status != model.StatusCompleted {
		t.Fatalf("expected completed instance, got %s", instance.Status)
	}
}

func TestExternalWorkerExtendLockAPI(t *testing.T) {
	h := setupTestHandler(t)
	r := mux.NewRouter()
	registerTestRoutes(r, h)
	ts := httptest.NewServer(r)
	defer ts.Close()

	steps := []model.StepDefinition{
		{ID: "start", Type: model.StepTypeStart, Outgoing: []model.Transition{{TargetRef: "service"}}},
		{
			ID:             "service",
			Type:           model.StepTypeServiceTask,
			Implementation: "external-extend-lock",
			Incoming:       []string{"start"},
			Outgoing:       []model.Transition{{TargetRef: "end"}},
		},
		{ID: "end", Type: model.StepTypeEnd, Incoming: []string{"service"}},
	}

	wf, err := h.engine.DeployWorkflow(context.Background(), "External Worker Extend Lock", steps)
	if err != nil {
		t.Fatalf("Failed to deploy: %v", err)
	}

	instance, err := h.engine.StartInstance(context.Background(), strconv.FormatInt(wf.ID, 10), nil)
	if err != nil {
		t.Fatalf("Failed to start instance: %v", err)
	}

	activateURL := ts.URL + "/jobs/activate"
	activateBody, _ := json.Marshal(dto.ActivateJobsRequest{Type: "external-extend-lock", Worker: "worker-1", MaxJobs: 1, LockDurationMs: 100})
	activateResp, err := http.Post(activateURL, "application/json", bytes.NewBuffer(activateBody))
	if err != nil {
		t.Fatalf("Failed to activate jobs: %v", err)
	}

	var activation dto.ActivateJobsResponse
	if err := json.NewDecoder(activateResp.Body).Decode(&activation); err != nil {
		t.Fatalf("Failed to decode activation response: %v", err)
	}
	if len(activation.Jobs) != 1 {
		t.Fatalf("expected 1 activated job, got %d", len(activation.Jobs))
	}
	jobKey := activation.Jobs[0].Key

	extendURL := fmt.Sprintf("%s/jobs/%s/extend-lock", ts.URL, jobKey)
	extendBody, _ := json.Marshal(dto.ExtendJobLockRequest{Worker: "worker-1", LockDurationMs: 300})
	extendResp, err := http.Post(extendURL, "application/json", bytes.NewBuffer(extendBody))
	if err != nil {
		t.Fatalf("Failed to extend job lock: %v", err)
	}
	if extendResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(extendResp.Body)
		t.Fatalf("Expected 200 OK, got %d: %s", extendResp.StatusCode, string(body))
	}

	time.Sleep(150 * time.Millisecond)

	completeURL := fmt.Sprintf("%s/jobs/%s/complete", ts.URL, jobKey)
	completeBody, _ := json.Marshal(dto.CompleteJobRequest{Worker: "worker-1"})
	completeResp, err := http.Post(completeURL, "application/json", bytes.NewBuffer(completeBody))
	if err != nil {
		t.Fatalf("Failed to complete job: %v", err)
	}
	if completeResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(completeResp.Body)
		t.Fatalf("Expected 200 OK, got %d: %s", completeResp.StatusCode, string(body))
	}

	instance, _ = h.engine.GetInstance(context.Background(), instance.ID)
	if instance.Status != model.StatusCompleted {
		t.Fatalf("expected completed instance after extend-lock + complete, got %s", instance.Status)
	}
}

func TestExternalWorkerActivateLongPollTimeout(t *testing.T) {
	h := setupTestHandler(t)
	r := mux.NewRouter()
	registerTestRoutes(r, h)
	ts := httptest.NewServer(r)
	defer ts.Close()

	activateURL := ts.URL + "/jobs/activate"
	activateBody, _ := json.Marshal(dto.ActivateJobsRequest{Type: "no-jobs-of-this-type", Worker: "worker-1", MaxJobs: 1, TimeoutMs: 300})

	start := time.Now()
	activateResp, err := http.Post(activateURL, "application/json", bytes.NewBuffer(activateBody))
	if err != nil {
		t.Fatalf("Failed to call activate jobs: %v", err)
	}
	elapsed := time.Since(start)

	if activateResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(activateResp.Body)
		t.Fatalf("Expected 200 OK, got %d: %s", activateResp.StatusCode, string(body))
	}

	var activated dto.ActivateJobsResponse
	if err := json.NewDecoder(activateResp.Body).Decode(&activated); err != nil {
		t.Fatalf("Failed to decode activate response: %v", err)
	}
	if len(activated.Jobs) != 0 {
		t.Fatalf("expected 0 jobs from long poll timeout, got %d", len(activated.Jobs))
	}

	if elapsed < 250*time.Millisecond {
		t.Fatalf("expected long poll to wait near timeout, waited only %v", elapsed)
	}
}

func TestJobsCapabilitiesAPI(t *testing.T) {
	h := setupTestHandler(t)
	r := mux.NewRouter()
	registerTestRoutes(r, h)
	ts := httptest.NewServer(r)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/jobs/capabilities")
	if err != nil {
		t.Fatalf("failed to get capabilities: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200 OK, got %d: %s", resp.StatusCode, string(body))
	}

	if got := resp.Header.Get(workerSDK.HeaderEngineProtocolVersion); got != workerSDK.WorkerProtocolVersion {
		t.Fatalf("expected protocol response header %q, got %q", workerSDK.WorkerProtocolVersion, got)
	}

	var payload dto.WorkerCapabilitiesResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode capabilities response: %v", err)
	}

	if payload.ProtocolVersion != workerSDK.WorkerProtocolVersion {
		t.Fatalf("expected protocol version %q, got %q", workerSDK.WorkerProtocolVersion, payload.ProtocolVersion)
	}
	if len(payload.Capabilities) == 0 {
		t.Fatalf("expected non-empty capabilities list")
	}
}

func TestActivateJobsRejectsUnsupportedWorkerProtocol(t *testing.T) {
	h := setupTestHandler(t)
	r := mux.NewRouter()
	registerTestRoutes(r, h)
	ts := httptest.NewServer(r)
	defer ts.Close()

	req, err := http.NewRequest(http.MethodPost, ts.URL+"/jobs/activate", bytes.NewBufferString(`{"type":"any","worker":"w"}`))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(workerSDK.HeaderWorkerProtocolVersion, "v999")

	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("failed to call activate jobs: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 400 Bad Request, got %d: %s", resp.StatusCode, string(body))
	}

	body, _ := io.ReadAll(resp.Body)
	if !bytes.Contains(body, []byte("unsupported worker protocol version")) {
		t.Fatalf("expected protocol validation error, got %q", string(body))
	}
}

func TestCompleteJobRejectsOversizedIdempotencyKey(t *testing.T) {
	h := setupTestHandler(t)
	r := mux.NewRouter()
	registerTestRoutes(r, h)
	ts := httptest.NewServer(r)
	defer ts.Close()

	req, err := http.NewRequest(http.MethodPost, ts.URL+"/jobs/1/complete", bytes.NewBufferString(`{"worker":"w"}`))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(idempotencyKeyHeader, strings.Repeat("x", maxIdempotencyKeyLength+1))

	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("failed to call complete job: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 400 Bad Request, got %d: %s", resp.StatusCode, string(body))
	}
	body, _ := io.ReadAll(resp.Body)
	if !bytes.Contains(body, []byte("Idempotency-Key exceeds")) {
		t.Fatalf("expected Idempotency-Key validation error, got %q", string(body))
	}
}

func TestEngineMetricsEndpointIncludesIdempotencyCounters(t *testing.T) {
	h := setupTestHandler(t)
	r := mux.NewRouter()
	registerTestRoutes(r, h)
	ts := httptest.NewServer(r)
	defer ts.Close()

	if _, err := h.engine.HasProcessedIdempotencyKey(context.Background(), "missing", "jobs.complete:10"); err != nil {
		t.Fatalf("failed idempotency miss check: %v", err)
	}
	if err := h.engine.RecordIdempotencyKey(context.Background(), "metrics-hit", "jobs.complete:10"); err != nil {
		t.Fatalf("failed to record idempotency key: %v", err)
	}
	if _, err := h.engine.HasProcessedIdempotencyKey(context.Background(), "metrics-hit", "jobs.complete:10"); err != nil {
		t.Fatalf("failed idempotency hit check: %v", err)
	}

	resp, err := http.Get(ts.URL + "/internal/metrics")
	if err != nil {
		t.Fatalf("failed to call metrics endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200 OK, got %d: %s", resp.StatusCode, string(body))
	}

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read metrics response body: %v", err)
	}
	if !bytes.Contains(rawBody, []byte("outboxPublishLagSec")) {
		t.Fatalf("expected metrics payload to include outboxPublishLagSec, got %s", string(rawBody))
	}

	var payload dto.EngineMetricsResponse
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		t.Fatalf("failed to decode metrics response: %v", err)
	}

	if payload.IdempotencyMiss < 1 {
		t.Fatalf("expected idempotency miss counter >=1, got %d", payload.IdempotencyMiss)
	}
	if payload.IdempotencyHit < 1 {
		t.Fatalf("expected idempotency hit counter >=1, got %d", payload.IdempotencyHit)
	}
	if payload.OutboxMaxAttempts <= 0 {
		t.Fatalf("expected outbox max attempts to be exposed, got %d", payload.OutboxMaxAttempts)
	}
	if payload.OutboxPublishLagSec < 0 {
		t.Fatalf("expected outbox publish lag metric to be non-negative, got %d", payload.OutboxPublishLagSec)
	}
}

func TestCompleteJobIdempotencyReplayBypassesWorkerMismatch(t *testing.T) {
	h := setupTestHandler(t)
	r := mux.NewRouter()
	registerTestRoutes(r, h)
	ts := httptest.NewServer(r)
	defer ts.Close()

	steps := []model.StepDefinition{
		{ID: "start", Type: model.StepTypeStart, Outgoing: []model.Transition{{TargetRef: "service"}}},
		{
			ID:             "service",
			Type:           model.StepTypeServiceTask,
			Implementation: "external-idempotent-complete",
			Incoming:       []string{"start"},
			Outgoing:       []model.Transition{{TargetRef: "end"}},
		},
		{ID: "end", Type: model.StepTypeEnd, Incoming: []string{"service"}},
	}

	wf, err := h.engine.DeployWorkflow(context.Background(), "External Worker Idempotent Complete", steps)
	if err != nil {
		t.Fatalf("Failed to deploy: %v", err)
	}

	instance, err := h.engine.StartInstance(context.Background(), strconv.FormatInt(wf.ID, 10), nil)
	if err != nil {
		t.Fatalf("Failed to start instance: %v", err)
	}

	activateBody, _ := json.Marshal(dto.ActivateJobsRequest{Type: "external-idempotent-complete", Worker: "worker-1", MaxJobs: 1, LockDurationMs: 200})
	activateResp, err := http.Post(ts.URL+"/jobs/activate", "application/json", bytes.NewBuffer(activateBody))
	if err != nil {
		t.Fatalf("Failed to activate jobs: %v", err)
	}
	defer activateResp.Body.Close()

	var activation dto.ActivateJobsResponse
	if err := json.NewDecoder(activateResp.Body).Decode(&activation); err != nil {
		t.Fatalf("Failed to decode activation response: %v", err)
	}
	if len(activation.Jobs) != 1 {
		t.Fatalf("expected 1 activated job, got %d", len(activation.Jobs))
	}

	jobKey := activation.Jobs[0].Key
	idempotencyKey := "idem-complete-1"

	completeURL := fmt.Sprintf("%s/jobs/%s/complete", ts.URL, jobKey)
	firstReqBody, _ := json.Marshal(dto.CompleteJobRequest{Worker: "worker-1", Variables: map[string]any{"approved": true}})
	req1, err := http.NewRequest(http.MethodPost, completeURL, bytes.NewBuffer(firstReqBody))
	if err != nil {
		t.Fatalf("failed to create first complete request: %v", err)
	}
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set(idempotencyKeyHeader, idempotencyKey)

	resp1, err := ts.Client().Do(req1)
	if err != nil {
		t.Fatalf("failed to call first complete job: %v", err)
	}
	defer resp1.Body.Close()
	if resp1.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp1.Body)
		t.Fatalf("Expected first complete 200 OK, got %d: %s", resp1.StatusCode, string(body))
	}

	replayReqBody, _ := json.Marshal(dto.CompleteJobRequest{Worker: "worker-2", Variables: map[string]any{"approved": false}})
	req2, err := http.NewRequest(http.MethodPost, completeURL, bytes.NewBuffer(replayReqBody))
	if err != nil {
		t.Fatalf("failed to create replay complete request: %v", err)
	}
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set(idempotencyKeyHeader, idempotencyKey)

	resp2, err := ts.Client().Do(req2)
	if err != nil {
		t.Fatalf("failed to call replay complete job: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp2.Body)
		t.Fatalf("Expected replay complete 200 OK, got %d: %s", resp2.StatusCode, string(body))
	}

	instance, _ = h.engine.GetInstance(context.Background(), instance.ID)
	if instance.Status != model.StatusCompleted {
		t.Fatalf("expected completed instance after idempotent complete replay, got %s", instance.Status)
	}
	if val, ok := instance.Context["approved"]; !ok || val != true {
		t.Fatalf("expected approved=true in context after replay, got %v", val)
	}
}

func TestFailJobIdempotencyReplayBypassesWorkerMismatch(t *testing.T) {
	h := setupTestHandler(t)
	r := mux.NewRouter()
	registerTestRoutes(r, h)
	ts := httptest.NewServer(r)
	defer ts.Close()

	steps := []model.StepDefinition{
		{ID: "start", Type: model.StepTypeStart, Outgoing: []model.Transition{{TargetRef: "service"}}},
		{
			ID:             "service",
			Type:           model.StepTypeServiceTask,
			Implementation: "external-idempotent-fail",
			Incoming:       []string{"start"},
			Outgoing:       []model.Transition{{TargetRef: "end"}},
		},
		{ID: "end", Type: model.StepTypeEnd, Incoming: []string{"service"}},
	}

	wf, err := h.engine.DeployWorkflow(context.Background(), "External Worker Idempotent Fail", steps)
	if err != nil {
		t.Fatalf("Failed to deploy: %v", err)
	}

	instance, err := h.engine.StartInstance(context.Background(), strconv.FormatInt(wf.ID, 10), nil)
	if err != nil {
		t.Fatalf("Failed to start instance: %v", err)
	}

	activateBody, _ := json.Marshal(dto.ActivateJobsRequest{Type: "external-idempotent-fail", Worker: "worker-1", MaxJobs: 1, LockDurationMs: 200})
	activateResp, err := http.Post(ts.URL+"/jobs/activate", "application/json", bytes.NewBuffer(activateBody))
	if err != nil {
		t.Fatalf("Failed to activate jobs: %v", err)
	}
	defer activateResp.Body.Close()

	var activation dto.ActivateJobsResponse
	if err := json.NewDecoder(activateResp.Body).Decode(&activation); err != nil {
		t.Fatalf("Failed to decode activation response: %v", err)
	}
	if len(activation.Jobs) != 1 {
		t.Fatalf("expected 1 activated job, got %d", len(activation.Jobs))
	}

	jobKey := activation.Jobs[0].Key
	idempotencyKey := "idem-fail-1"

	failURL := fmt.Sprintf("%s/jobs/%s/fail", ts.URL, jobKey)
	retries := 1
	firstReqBody, _ := json.Marshal(dto.FailJobRequest{Worker: "worker-1", ErrorMessage: "temporary failure", Retries: &retries})
	req1, err := http.NewRequest(http.MethodPost, failURL, bytes.NewBuffer(firstReqBody))
	if err != nil {
		t.Fatalf("failed to create first fail request: %v", err)
	}
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set(idempotencyKeyHeader, idempotencyKey)

	resp1, err := ts.Client().Do(req1)
	if err != nil {
		t.Fatalf("failed to call first fail job: %v", err)
	}
	defer resp1.Body.Close()
	if resp1.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp1.Body)
		t.Fatalf("Expected first fail 200 OK, got %d: %s", resp1.StatusCode, string(body))
	}

	replayRetries := 0
	replayReqBody, _ := json.Marshal(dto.FailJobRequest{Worker: "worker-2", ErrorMessage: "should be ignored by replay", Retries: &replayRetries})
	req2, err := http.NewRequest(http.MethodPost, failURL, bytes.NewBuffer(replayReqBody))
	if err != nil {
		t.Fatalf("failed to create replay fail request: %v", err)
	}
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set(idempotencyKeyHeader, idempotencyKey)

	resp2, err := ts.Client().Do(req2)
	if err != nil {
		t.Fatalf("failed to call replay fail job: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp2.Body)
		t.Fatalf("Expected replay fail 200 OK, got %d: %s", resp2.StatusCode, string(body))
	}

	instance, _ = h.engine.GetInstance(context.Background(), instance.ID)
	if instance.Status != model.StatusRunning {
		t.Fatalf("expected running instance after idempotent fail replay, got %s", instance.Status)
	}
}

func TestExtendLockIdempotencyReplayBypassesWorkerMismatch(t *testing.T) {
	h := setupTestHandler(t)
	r := mux.NewRouter()
	registerTestRoutes(r, h)
	ts := httptest.NewServer(r)
	defer ts.Close()

	steps := []model.StepDefinition{
		{ID: "start", Type: model.StepTypeStart, Outgoing: []model.Transition{{TargetRef: "service"}}},
		{
			ID:             "service",
			Type:           model.StepTypeServiceTask,
			Implementation: "external-idempotent-lock",
			Incoming:       []string{"start"},
			Outgoing:       []model.Transition{{TargetRef: "end"}},
		},
		{ID: "end", Type: model.StepTypeEnd, Incoming: []string{"service"}},
	}

	wf, err := h.engine.DeployWorkflow(context.Background(), "External Worker Idempotent Lock", steps)
	if err != nil {
		t.Fatalf("Failed to deploy: %v", err)
	}

	_, err = h.engine.StartInstance(context.Background(), strconv.FormatInt(wf.ID, 10), nil)
	if err != nil {
		t.Fatalf("Failed to start instance: %v", err)
	}

	activateBody, _ := json.Marshal(dto.ActivateJobsRequest{Type: "external-idempotent-lock", Worker: "worker-1", MaxJobs: 1, LockDurationMs: 200})
	activateResp, err := http.Post(ts.URL+"/jobs/activate", "application/json", bytes.NewBuffer(activateBody))
	if err != nil {
		t.Fatalf("Failed to activate jobs: %v", err)
	}
	defer activateResp.Body.Close()

	var activation dto.ActivateJobsResponse
	if err := json.NewDecoder(activateResp.Body).Decode(&activation); err != nil {
		t.Fatalf("Failed to decode activation response: %v", err)
	}
	if len(activation.Jobs) != 1 {
		t.Fatalf("expected 1 activated job, got %d", len(activation.Jobs))
	}

	jobKey := activation.Jobs[0].Key
	idempotencyKey := "idem-lock-1"

	extendURL := fmt.Sprintf("%s/jobs/%s/extend-lock", ts.URL, jobKey)
	goodReqBody, _ := json.Marshal(dto.ExtendJobLockRequest{Worker: "worker-1", LockDurationMs: 500})
	req1, err := http.NewRequest(http.MethodPost, extendURL, bytes.NewBuffer(goodReqBody))
	if err != nil {
		t.Fatalf("failed to create first extend request: %v", err)
	}
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set(idempotencyKeyHeader, idempotencyKey)

	resp1, err := ts.Client().Do(req1)
	if err != nil {
		t.Fatalf("failed to call first extend lock: %v", err)
	}
	defer resp1.Body.Close()
	if resp1.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp1.Body)
		t.Fatalf("Expected first extend 200 OK, got %d: %s", resp1.StatusCode, string(body))
	}

	badReqBody, _ := json.Marshal(dto.ExtendJobLockRequest{Worker: "worker-2", LockDurationMs: 500})
	req2, err := http.NewRequest(http.MethodPost, extendURL, bytes.NewBuffer(badReqBody))
	if err != nil {
		t.Fatalf("failed to create replay extend request: %v", err)
	}
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set(idempotencyKeyHeader, idempotencyKey)

	resp2, err := ts.Client().Do(req2)
	if err != nil {
		t.Fatalf("failed to call replay extend lock: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp2.Body)
		t.Fatalf("Expected replay extend 200 OK, got %d: %s", resp2.StatusCode, string(body))
	}
}

func TestExternalWorkerSDKLockRenewalAgainstAPI(t *testing.T) {
	h := setupTestHandler(t)
	r := mux.NewRouter()
	registerTestRoutes(r, h)
	ts := httptest.NewServer(r)
	defer ts.Close()

	steps := []model.StepDefinition{
		{ID: "start", Type: model.StepTypeStart, Outgoing: []model.Transition{{TargetRef: "service"}}},
		{
			ID:             "service",
			Type:           model.StepTypeServiceTask,
			Implementation: "external-sdk-renew",
			Incoming:       []string{"start"},
			Outgoing:       []model.Transition{{TargetRef: "end"}},
		},
		{ID: "end", Type: model.StepTypeEnd, Incoming: []string{"service"}},
	}

	wf, err := h.engine.DeployWorkflow(context.Background(), "External Worker SDK Renew", steps)
	if err != nil {
		t.Fatalf("Failed to deploy: %v", err)
	}

	instance, err := h.engine.StartInstance(context.Background(), strconv.FormatInt(wf.ID, 10), nil)
	if err != nil {
		t.Fatalf("Failed to start instance: %v", err)
	}

	client, err := workerSDK.NewClient(workerSDK.ClientConfig{BaseURL: ts.URL, HTTPClient: ts.Client()})
	if err != nil {
		t.Fatalf("failed to create SDK client: %v", err)
	}

	worker, err := workerSDK.NewWorker(client, workerSDK.WorkerConfig{
		JobType:           "external-sdk-renew",
		WorkerName:        "sdk-worker-1",
		MaxJobs:           1,
		ActivateTimeout:   100 * time.Millisecond,
		LockDuration:      100 * time.Millisecond,
		LockRenewInterval: 25 * time.Millisecond,
		Handler: func(ctx context.Context, job model.Job) (map[string]any, error) {
			timer := time.NewTimer(220 * time.Millisecond)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-timer.C:
			}
			return map[string]any{"sdkRenewed": true}, nil
		},
	})
	if err != nil {
		t.Fatalf("failed to create SDK worker: %v", err)
	}

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("SDK worker run failed: %v", err)
	}

	instance, _ = h.engine.GetInstance(context.Background(), instance.ID)
	if instance.Status != model.StatusCompleted {
		t.Fatalf("expected completed instance after SDK lock renewal, got %s", instance.Status)
	}
	if val, ok := instance.Context["sdkRenewed"]; !ok || val != true {
		t.Fatalf("expected sdkRenewed=true in context, got %v", val)
	}
}

func TestStartInstanceAPI(t *testing.T) {
	h := setupTestHandler(t)
	r := mux.NewRouter()
	registerTestRoutes(r, h)
	ts := httptest.NewServer(r)
	defer ts.Close()

	// Deploy a simple workflow
	steps := []model.StepDefinition{
		{ID: "start", Type: model.StepTypeStart, Outgoing: []model.Transition{{TargetRef: "end"}}},
		{ID: "end", Type: model.StepTypeEnd},
	}
	wf, err := h.engine.DeployWorkflow(context.Background(), "Start API Test", steps)
	if err != nil {
		t.Fatalf("Failed to deploy: %v", err)
	}

	// Call Start Instance API
	url := ts.URL + "/instances"
	reqBody := dto.StartInstanceRequest{
		WorkflowID: strconv.FormatInt(wf.ID, 10),
		Context:    map[string]any{"foo": "bar"},
	}
	jsonBody, _ := json.Marshal(reqBody)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		t.Fatalf("Failed to call start instance: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", resp.StatusCode)
	}

	var instance dto.WorkflowInstanceResponse
	if err := json.NewDecoder(resp.Body).Decode(&instance); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if instance.WorkflowID != reqBody.WorkflowID {
		t.Errorf("Expected WorkflowID %s, got %s", reqBody.WorkflowID, instance.WorkflowID)
	}
	if val, ok := instance.Context["foo"]; !ok || val != "bar" {
		t.Errorf("Expected context foo=bar, got %v", val)
	}
}

func TestGetInstanceAPI(t *testing.T) {
	h := setupTestHandler(t)
	r := mux.NewRouter()
	registerTestRoutes(r, h)
	ts := httptest.NewServer(r)
	defer ts.Close()

	// Deploy and Start manually
	steps := []model.StepDefinition{
		{ID: "start", Type: model.StepTypeStart, Outgoing: []model.Transition{{TargetRef: "end"}}},
		{ID: "end", Type: model.StepTypeEnd},
	}
	wf, _ := h.engine.DeployWorkflow(context.Background(), "Get API Test", steps)
	instance, _ := h.engine.StartInstance(context.Background(), strconv.FormatInt(wf.ID, 10), map[string]any{"key": "val"})

	// Call Get Instance API
	url := ts.URL + "/instances/" + instance.ID
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("Failed to call get instance: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", resp.StatusCode)
	}

	var fetchedInstance dto.WorkflowInstanceResponse
	if err := json.NewDecoder(resp.Body).Decode(&fetchedInstance); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if fetchedInstance.ID != instance.ID {
		t.Errorf("Expected ID %s, got %s", instance.ID, fetchedInstance.ID)
	}
	if val, ok := fetchedInstance.Context["key"]; !ok || val != "val" {
		t.Errorf("Expected context key=val, got %v", val)
	}
}
