package api

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/artificialflow/artificialflow/backend/libs/auth"
	"github.com/artificialflow/artificialflow/backend/libs/iam"
	"github.com/artificialflow/artificialflow/backend/services/workflow-command/internal/interfaces/http/dto"
	"github.com/gorilla/mux"
)

func apiTestPublicKey(t *testing.T) string {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
}

func TestIdentityManagementPrivateKeyJWTLifecycle(t *testing.T) {
	tokenFile := t.TempDir() + "/owner.pat"
	stateFile := t.TempDir() + "/state.json"
	if err := os.WriteFile(tokenFile, []byte("owner-token"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stateFile, []byte(`{"org_id":"org-1","project_id":"project-1"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	addCount := 0
	var removePayload map[string]any
	zitadel := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/zitadel.user.v2.UserService/CreateUser":
			_, _ = w.Write([]byte(`{"id":"client-1"}`))
		case "/zitadel.user.v2.UserService/GetUserByID":
			_, _ = w.Write([]byte(`{"user":{"userId":"client-1","username":"sdk","state":"USER_STATE_ACTIVE","machine":{"name":"SDK"}}}`))
		case "/zitadel.authorization.v2.AuthorizationService/ListAuthorizations":
			if addCount == 0 {
				_, _ = w.Write([]byte(`{"authorizations":[]}`))
			} else {
				_, _ = w.Write([]byte(`{"authorizations":[{"id":"auth-1","state":"STATE_ACTIVE","project":{"id":"project-1"},"user":{"id":"client-1"},"roles":[{"key":"flowgo client"}]}]}`))
			}
		case "/zitadel.authorization.v2.AuthorizationService/CreateAuthorization":
			_, _ = w.Write([]byte(`{"id":"auth-1"}`))
		case "/zitadel.user.v2.UserService/AddKey":
			addCount++
			_, _ = w.Write([]byte(`{"keyId":"key-` + string(rune('0'+addCount)) + `","creationDate":"2026-07-19T12:00:00Z"}`))
		case "/zitadel.user.v2.UserService/ListKeys":
			_, _ = w.Write([]byte(`{"keys":[{"id":"key-1","userId":"client-1"},{"id":"key-2","userId":"client-1"}]}`))
		case "/zitadel.user.v2.UserService/RemoveKey":
			if err := json.NewDecoder(r.Body).Decode(&removePayload); err != nil {
				t.Fatal(err)
			}
			_, _ = w.Write([]byte(`{}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer zitadel.Close()

	handler := setupTestHandlerWithIdentityConfig(t, iam.DeploymentConfig{
		Mode: iam.DeploymentModeZITADEL,
		ZITADELManagement: iam.ZITADELManagementConfig{
			BaseURL: zitadel.URL, OwnerPATFile: tokenFile, BootstrapStateFile: stateFile,
		},
	})
	router := mux.NewRouter()
	registerTestRoutes(router, handler)
	admin := auth.Principal{Subject: "admin", Roles: []string{auth.RoleArtificialFlowAdmin}}
	expiresAt := time.Now().UTC().Add(30 * 24 * time.Hour).Format(time.RFC3339)

	createBody, _ := json.Marshal(map[string]string{
		"name": "SDK", "username": "sdk", "public_key": apiTestPublicKey(t), "key_expires_at": expiresAt,
	})
	request := httptest.NewRequest(http.MethodPost, "/identity/management/clients", bytes.NewReader(createBody))
	request = request.WithContext(auth.WithPrincipal(request.Context(), admin))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("create key client: %d %s", response.Code, response.Body.String())
	}
	if response.Header().Get("Cache-Control") != "no-store" || response.Header().Get("Content-Disposition") == "" ||
		response.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("missing sensitive response headers: %#v", response.Header())
	}
	var created dto.IdentityManagementClientResponse
	if err := json.NewDecoder(response.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if len(created.Credentials) != 1 || created.Credentials[0].Type != "private_key_jwt" || created.Credentials[0].ID != "key-1" {
		t.Fatalf("unexpected created credentials: %#v", created.Credentials)
	}

	addBody, _ := json.Marshal(map[string]string{"public_key": apiTestPublicKey(t), "key_expires_at": expiresAt})
	request = httptest.NewRequest(http.MethodPost, "/identity/management/clients/client-1/keys", bytes.NewReader(addBody))
	request = request.WithContext(auth.WithPrincipal(request.Context(), admin))
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusCreated || addCount != 2 {
		t.Fatalf("add overlapping key: %d %s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodDelete, "/identity/management/clients/client-1/keys/key-1", nil)
	request = request.WithContext(auth.WithPrincipal(request.Context(), admin))
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("revoke key: %d %s", response.Code, response.Body.String())
	}
	if removePayload["userId"] != "client-1" || removePayload["keyId"] != "key-1" {
		t.Fatalf("unexpected RemoveKey payload: %#v", removePayload)
	}
}
