package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/azizAltaleb/flowgo/backend/libs/auth"
	"github.com/azizAltaleb/flowgo/backend/libs/iam"
	"github.com/azizAltaleb/flowgo/backend/libs/model"
	workerSDK "github.com/azizAltaleb/flowgo/backend/libs/worker"
	"github.com/azizAltaleb/flowgo/backend/services/workflow-command/internal/application"
	"github.com/azizAltaleb/flowgo/backend/services/workflow-command/internal/infrastructure/messaging"
	"github.com/azizAltaleb/flowgo/backend/services/workflow-command/internal/infrastructure/persistence"
	"github.com/azizAltaleb/flowgo/backend/services/workflow-command/internal/interfaces/http/dto"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
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
	// Use in-memory SQLite for testing
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
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
	return NewHandler(e, identityConfig)
}

func TestIdentityConfigAPI(t *testing.T) {
	h := setupTestHandler(t)
	r := mux.NewRouter()
	h.RegisterRoutes(r)
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
	if len(config.StandardRoles) != 3 {
		t.Fatalf("Expected 3 standard roles, got %#v", config.StandardRoles)
	}
	if config.StandardRoles[0] != auth.RoleFlowGoClient {
		t.Fatalf("Expected first standard role %q, got %q", auth.RoleFlowGoClient, config.StandardRoles[0])
	}
}

func TestIdentityManagementRoutesExternalModeReturnNotFound(t *testing.T) {
	h := setupTestHandler(t)
	r := mux.NewRouter()
	h.RegisterRoutes(r)
	req := httptest.NewRequest(http.MethodGet, "/identity/management/users", nil)
	req = req.WithContext(auth.WithPrincipal(req.Context(), auth.Principal{Subject: "admin", Roles: []string{auth.RoleFlowGoAdmin}}))
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("Expected 404 for external IAM management route, got %d", rec.Code)
	}
}

func TestIdentityManagementRoutesBundledRequireAdmin(t *testing.T) {
	h := setupTestHandlerWithIdentityConfig(t, iam.DeploymentConfig{Mode: iam.DeploymentModeZITADEL})
	r := mux.NewRouter()
	h.RegisterRoutes(r)
	req := httptest.NewRequest(http.MethodGet, "/identity/management/users", nil)
	req = req.WithContext(auth.WithPrincipal(req.Context(), auth.Principal{Subject: "viewer", Roles: []string{auth.RoleFlowGoViewer}}))
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
	h.RegisterRoutes(r)
	req := httptest.NewRequest(http.MethodGet, "/identity/management/users", nil)
	req = req.WithContext(auth.WithPrincipal(req.Context(), auth.Principal{Subject: "admin", Roles: []string{auth.RoleFlowGoAdmin}}))
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
	if len(response.Users[0].Roles) != 1 || response.Users[0].Roles[0] != auth.RoleFlowGoAdmin {
		t.Fatalf("Unexpected user roles %#v", response.Users[0].Roles)
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
			BaseURL:            zitadel.URL,
			OwnerPATFile:       tokenFile,
			BootstrapStateFile: stateFile,
		},
	})
	r := mux.NewRouter()
	h.RegisterRoutes(r)
	body := `{"username":"sdk-orders","name":"Orders SDK","description":"Order system","environment":"production","owner_email":"platform@example.com","purpose":"Order worker","token_expires_at":"2027-01-01T00:00:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/identity/management/clients", strings.NewReader(body))
	req = req.WithContext(auth.WithPrincipal(req.Context(), auth.Principal{Subject: "admin", Roles: []string{auth.RoleFlowGoAdmin}}))
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("Expected 201 for client token creation, got %d: %s", rec.Code, rec.Body.String())
	}
	var response dto.IdentityManagementClientTokenResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode client token response: %v", err)
	}
	if response.ClientID != "client-user-1" || response.TokenID != "pat-1" || response.Token != "sdk-token" {
		t.Fatalf("Unexpected client token response: %#v", response)
	}
	if response.Role != auth.RoleFlowGoClient {
		t.Fatalf("Expected client role %q, got %q", auth.RoleFlowGoClient, response.Role)
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
	if !ok || len(roleKeys) != 1 || roleKeys[0] != auth.RoleFlowGoClient {
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
			BaseURL:            zitadel.URL,
			OwnerPATFile:       tokenFile,
			BootstrapStateFile: stateFile,
		},
	})
	r := mux.NewRouter()
	h.RegisterRoutes(r)

	req := httptest.NewRequest(http.MethodGet, "/identity/management/clients", nil)
	req = req.WithContext(auth.WithPrincipal(req.Context(), auth.Principal{Subject: "admin", Roles: []string{auth.RoleFlowGoAdmin}}))
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
	req = req.WithContext(auth.WithPrincipal(req.Context(), auth.Principal{Subject: "admin", Roles: []string{auth.RoleFlowGoAdmin}}))
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("Expected 201 for client token rotation, got %d: %s", rec.Code, rec.Body.String())
	}
	var rotateResponse dto.IdentityManagementClientTokenResponse
	if err := json.NewDecoder(rec.Body).Decode(&rotateResponse); err != nil {
		t.Fatalf("Failed to decode rotated token: %v", err)
	}
	if rotateResponse.TokenID != "pat-2" || rotateResponse.Token != "rotated-token" || rotatePayload["userId"] != "client-user-1" {
		t.Fatalf("Unexpected rotate response or payload: %#v %#v", rotateResponse, rotatePayload)
	}

	req = httptest.NewRequest(http.MethodDelete, "/identity/management/clients/client-user-1/tokens/pat-1", nil)
	req = req.WithContext(auth.WithPrincipal(req.Context(), auth.Principal{Subject: "admin", Roles: []string{auth.RoleFlowGoAdmin}}))
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("Expected 204 for client token revoke, got %d: %s", rec.Code, rec.Body.String())
	}
	if revokePayload["userId"] != "client-user-1" || revokePayload["tokenId"] != "pat-1" {
		t.Fatalf("Unexpected revoke payload: %#v", revokePayload)
	}

	req = httptest.NewRequest(http.MethodDelete, "/identity/management/clients/client-user-1", nil)
	req = req.WithContext(auth.WithPrincipal(req.Context(), auth.Principal{Subject: "admin", Roles: []string{auth.RoleFlowGoAdmin}}))
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
	h.RegisterRoutes(r)

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
	h.RegisterRoutes(r)
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
	h.RegisterRoutes(r)
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
	h.RegisterRoutes(r)
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

func TestPublishSignalAPI(t *testing.T) {
	h := setupTestHandler(t)
	r := mux.NewRouter()
	h.RegisterRoutes(r)
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
	h.RegisterRoutes(r)
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
	h.RegisterRoutes(r)
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
	h.RegisterRoutes(r)
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

func TestExternalWorkerFailJobAPI(t *testing.T) {
	h := setupTestHandler(t)
	r := mux.NewRouter()
	h.RegisterRoutes(r)
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
	h.RegisterRoutes(r)
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
	h.RegisterRoutes(r)
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
	h.RegisterRoutes(r)
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
	h.RegisterRoutes(r)
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
	h.RegisterRoutes(r)
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
	h.RegisterRoutes(r)
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
	h.RegisterRoutes(r)
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
	h.RegisterRoutes(r)
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
	h.RegisterRoutes(r)
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
	h.RegisterRoutes(r)
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
	h.RegisterRoutes(r)
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
	h.RegisterRoutes(r)
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
	h.RegisterRoutes(r)
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
