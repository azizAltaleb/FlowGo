package iam

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/azizAltaleb/goflow/backend/libs/auth"
)

var ErrZITADELManagementNotConfigured = errors.New("zitadel management is not configured")
var ErrZITADELManagedClientNotFound = errors.New("zitadel managed client not found")

type ZITADELManagementConfig struct {
	BaseURL            string
	PublicHost         string
	OwnerPATFile       string
	BootstrapStateFile string
}

type ZITADELBootstrapState struct {
	OrgID     string `json:"org_id"`
	ProjectID string `json:"project_id"`
}

type ManagedUser struct {
	ID                 string   `json:"id"`
	Username           string   `json:"username"`
	PreferredLoginName string   `json:"preferred_login_name"`
	DisplayName        string   `json:"display_name"`
	GivenName          string   `json:"given_name"`
	FamilyName         string   `json:"family_name"`
	Email              string   `json:"email"`
	EmailVerified      bool     `json:"email_verified"`
	State              string   `json:"state"`
	Type               string   `json:"type"`
	CreatedAt          string   `json:"created_at"`
	ChangedAt          string   `json:"changed_at"`
	Roles              []string `json:"roles"`
}

type ManagedRole struct {
	Key         string `json:"key"`
	DisplayName string `json:"display_name"`
	Group       string `json:"group"`
}

type ManagedClientTokenCreate struct {
	Username       string
	Name           string
	Description    string
	Environment    string
	OwnerEmail     string
	Purpose        string
	TokenExpiresAt string
}

type ManagedClientTokenRotate struct {
	TokenExpiresAt string
}

type ManagedClientTokenSummary struct {
	TokenID        string
	TokenCreatedAt string
	TokenChangedAt string
	TokenExpiresAt string
	Status         string
}

type ManagedClient struct {
	ClientID    string
	Username    string
	Name        string
	Description string
	Environment string
	OwnerEmail  string
	Purpose     string
	Role        string
	State       string
	CreatedAt   string
	ChangedAt   string
	Tokens      []ManagedClientTokenSummary
}

type ManagedClientToken struct {
	ClientID       string
	Username       string
	Name           string
	Description    string
	Environment    string
	OwnerEmail     string
	Purpose        string
	Role           string
	TokenID        string
	Token          string
	TokenCreatedAt string
	TokenExpiresAt string
}

type ManagedUserCreate struct {
	Username               string
	GivenName              string
	FamilyName             string
	Email                  string
	Password               string
	PasswordChangeRequired bool
	RoleKeys               []string
}

type ManagedUserUpdate struct {
	Username       string
	GivenName      string
	FamilyName     string
	DisplayName    string
	Email          string
	RoleKeys       []string
	UpdateRoleKeys bool
}

type ManagedRoleCreate struct {
	Key         string
	DisplayName string
	Group       string
}

type ManagedRoleUpdate struct {
	DisplayName string
	Group       string
}

type ZITADELError struct {
	StatusCode int
	Code       string
	Message    string
	Body       string
}

func (e *ZITADELError) Error() string {
	message := strings.TrimSpace(e.Message)
	if message == "" {
		message = strings.TrimSpace(e.Body)
	}
	if message == "" {
		message = http.StatusText(e.StatusCode)
	}
	return fmt.Sprintf("ZITADEL request failed with status %d: %s", e.StatusCode, message)
}

func ResolveZITADELManagementConfigFromEnv(authConfig auth.Config, frontendConfig FrontendAuthConfig) ZITADELManagementConfig {
	baseURL := strings.TrimSpace(os.Getenv("ZITADEL_MANAGEMENT_URL"))
	if baseURL == "" {
		baseURL = strings.TrimSpace(authConfig.InternalIssuerURL)
	}
	if baseURL == "" {
		baseURL = strings.TrimSpace(frontendConfig.OIDCAuthority)
	}
	if baseURL == "" {
		baseURL = strings.TrimSpace(authConfig.ExternalIssuerURL)
	}
	publicHost := strings.TrimSpace(os.Getenv("ZITADEL_PUBLIC_HOST"))
	if publicHost == "" {
		publicHost = hostFromURL(firstNonEmpty(authConfig.ExternalIssuerURL, frontendConfig.OIDCAuthority, baseURL))
	}
	ownerPATFile := strings.TrimSpace(os.Getenv("ZITADEL_OWNER_PAT_FILE"))
	if ownerPATFile == "" {
		ownerPATFile = "/zitadel/bootstrap/owner.pat"
	}
	bootstrapStateFile := strings.TrimSpace(os.Getenv("GOFLOW_ZITADEL_BOOTSTRAP_STATE_FILE"))
	if bootstrapStateFile == "" {
		bootstrapStateFile = "/goflow/bootstrap/goflow-zitadel.json"
	}
	return ZITADELManagementConfig{
		BaseURL:            strings.TrimRight(baseURL, "/"),
		PublicHost:         publicHost,
		OwnerPATFile:       ownerPATFile,
		BootstrapStateFile: bootstrapStateFile,
	}
}

type ZITADELManagementClient struct {
	cfg        ZITADELManagementConfig
	httpClient *http.Client
}

func NewZITADELManagementClient(cfg ZITADELManagementConfig) *ZITADELManagementClient {
	return &ZITADELManagementClient{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (c *ZITADELManagementClient) readOwnerToken() (string, error) {
	path := strings.TrimSpace(c.cfg.OwnerPATFile)
	if path == "" {
		return "", ErrZITADELManagementNotConfigured
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read ZITADEL owner PAT: %w", err)
	}
	token := strings.TrimSpace(string(content))
	if token == "" {
		return "", ErrZITADELManagementNotConfigured
	}
	return token, nil
}

func (c *ZITADELManagementClient) readBootstrapState() (ZITADELBootstrapState, error) {
	path := strings.TrimSpace(c.cfg.BootstrapStateFile)
	if path == "" {
		return ZITADELBootstrapState{}, ErrZITADELManagementNotConfigured
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return ZITADELBootstrapState{}, fmt.Errorf("read ZITADEL bootstrap state: %w", err)
	}
	var state ZITADELBootstrapState
	if err := json.Unmarshal(content, &state); err != nil {
		return ZITADELBootstrapState{}, fmt.Errorf("decode ZITADEL bootstrap state: %w", err)
	}
	if strings.TrimSpace(state.OrgID) == "" || strings.TrimSpace(state.ProjectID) == "" {
		return ZITADELBootstrapState{}, ErrZITADELManagementNotConfigured
	}
	return state, nil
}

func (c *ZITADELManagementClient) connect(ctx context.Context, path string, payload any, target any) error {
	token, err := c.readOwnerToken()
	if err != nil {
		return err
	}
	return c.requestJSON(ctx, http.MethodPost, path, token, payload, target)
}

func (c *ZITADELManagementClient) requestJSON(ctx context.Context, method string, requestPath string, token string, payload any, target any) error {
	baseURL := strings.TrimSpace(c.cfg.BaseURL)
	if baseURL == "" {
		return ErrZITADELManagementNotConfigured
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	requestURL := baseURL + requestPath
	req, err := http.NewRequestWithContext(ctx, method, requestURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if c.cfg.PublicHost != "" {
		req.Host = c.cfg.PublicHost
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var responseError struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}
		_ = json.Unmarshal(responseBody, &responseError)
		return &ZITADELError{
			StatusCode: resp.StatusCode,
			Code:       responseError.Code,
			Message:    responseError.Message,
			Body:       string(responseBody),
		}
	}
	if target == nil || len(responseBody) == 0 {
		return nil
	}
	if err := json.Unmarshal(responseBody, target); err != nil {
		return fmt.Errorf("decode ZITADEL response: %w", err)
	}
	return nil
}

func hostFromURL(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" {
		return ""
	}
	return parsed.Host
}
