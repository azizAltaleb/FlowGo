package iam

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func testRSASPKI(t *testing.T, bits int) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatalf("marshal RSA public key: %v", err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
}

func TestParseClientPublicKeyRequiresRSA2048SPKI(t *testing.T) {
	if _, err := parseClientPublicKey(testRSASPKI(t, 2048)); err != nil {
		t.Fatalf("expected RSA-2048 SPKI to be accepted: %v", err)
	}
	if _, err := parseClientPublicKey(testRSASPKI(t, 1024)); !errors.Is(err, ErrInvalidClientPublicKey) {
		t.Fatalf("expected RSA-1024 key to be rejected, got %v", err)
	}
}

func TestClientCredentialExpirationAppliesDefaultAndMaximum(t *testing.T) {
	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	got, err := clientCredentialExpiration("", now)
	if err != nil {
		t.Fatalf("default expiration: %v", err)
	}
	if got != now.Add(defaultClientCredentialLifetime).Format(time.RFC3339) {
		t.Fatalf("unexpected default expiration %q", got)
	}
	tooLate := now.Add(maxClientCredentialLifetime + time.Second).Format(time.RFC3339)
	if _, err := clientCredentialExpiration(tooLate, now); !errors.Is(err, ErrInvalidClientCredentialExpiry) {
		t.Fatalf("expected over-maximum expiration rejection, got %v", err)
	}
}

func TestZITADELClientKeyLifecycleChecksOwnership(t *testing.T) {
	tokenFile := t.TempDir() + "/owner.pat"
	stateFile := t.TempDir() + "/state.json"
	if err := os.WriteFile(tokenFile, []byte("owner-token"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stateFile, []byte(`{"org_id":"org-1","project_id":"project-1"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var addPayload map[string]any
	var removePayload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/zitadel.user.v2.UserService/GetUserByID":
			_, _ = w.Write([]byte(`{"user":{"userId":"client-1","username":"sdk","state":"USER_STATE_ACTIVE","machine":{"name":"SDK"}}}`))
		case "/zitadel.authorization.v2.AuthorizationService/ListAuthorizations":
			_, _ = w.Write([]byte(`{"authorizations":[{"id":"auth-1","state":"STATE_ACTIVE","project":{"id":"project-1"},"user":{"id":"client-1"},"roles":[{"key":"flowgo client"}]}]}`))
		case "/zitadel.user.v2.UserService/AddKey":
			if err := json.NewDecoder(r.Body).Decode(&addPayload); err != nil {
				t.Fatal(err)
			}
			_, _ = w.Write([]byte(`{"keyId":"key-2","creationDate":"2026-07-19T12:00:00Z"}`))
		case "/zitadel.user.v2.UserService/ListKeys":
			_, _ = w.Write([]byte(`{"keys":[{"id":"key-2","userId":"client-1","creationDate":"2026-07-19T12:00:00Z","expirationDate":"2026-10-01T00:00:00Z"}]}`))
		case "/zitadel.user.v2.UserService/RemoveKey":
			if err := json.NewDecoder(r.Body).Decode(&removePayload); err != nil {
				t.Fatal(err)
			}
			_, _ = w.Write([]byte(`{}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewZITADELManagementClient(ZITADELManagementConfig{
		BaseURL: server.URL, OwnerPATFile: tokenFile, BootstrapStateFile: stateFile,
	})
	expiresAt := time.Now().UTC().Add(30 * 24 * time.Hour).Format(time.RFC3339)
	credential, err := client.AddClientKey(t.Context(), "client-1", ManagedClientKeyAdd{
		PublicKey: testRSASPKI(t, 2048), ExpiresAt: expiresAt,
	})
	if err != nil {
		t.Fatalf("add client key: %v", err)
	}
	if credential.ID != "key-2" || credential.Type != "private_key_jwt" {
		t.Fatalf("unexpected credential: %#v", credential)
	}
	if addPayload["userId"] != "client-1" || addPayload["expirationDate"] != expiresAt {
		t.Fatalf("unexpected AddKey payload: %#v", addPayload)
	}
	if err := client.RemoveClientKey(t.Context(), "client-1", "other-client-key"); !errors.Is(err, ErrZITADELManagedClientNotFound) {
		t.Fatalf("expected foreign key rejection, got %v", err)
	}
	if removePayload != nil {
		t.Fatalf("foreign key must not be removed: %#v", removePayload)
	}
	if err := client.RemoveClientKey(t.Context(), "client-1", "key-2"); err != nil {
		t.Fatalf("remove owned key: %v", err)
	}
	if removePayload["userId"] != "client-1" || removePayload["keyId"] != "key-2" {
		t.Fatalf("unexpected RemoveKey payload: %#v", removePayload)
	}
}

func TestLegacyPATIssuanceDisabledByDefault(t *testing.T) {
	client := NewZITADELManagementClient(ZITADELManagementConfig{})
	if _, err := client.CreateClientToken(t.Context(), ManagedClientTokenCreate{}); !errors.Is(err, ErrLegacyPATCreationDisabled) {
		t.Fatalf("expected PAT creation to be disabled, got %v", err)
	}
	if _, err := client.RotateClientToken(t.Context(), "client-1", ManagedClientTokenRotate{}); !errors.Is(err, ErrLegacyPATRotationDisabled) {
		t.Fatalf("expected PAT rotation to be disabled, got %v", err)
	}
}
