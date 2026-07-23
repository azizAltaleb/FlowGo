package iam

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/artificialflow/artificialflow/backend/libs/auth"
)

var ErrZITADELManagementNotConfigured = errors.New("zitadel management is not configured")
var ErrZITADELManagedClientNotFound = errors.New("zitadel managed client not found")
var ErrInvalidClientPublicKey = errors.New("client public key must be an RSA SPKI public key of at least 2048 bits")
var ErrInvalidClientCredentialExpiry = errors.New("client credential expiry must be in the future and no more than 365 days away")
var ErrLegacyPATCreationDisabled = errors.New("legacy PAT creation is disabled")
var ErrLegacyPATRotationDisabled = errors.New("legacy PAT rotation is disabled")

const (
	defaultClientCredentialLifetime = 90 * 24 * time.Hour
	maxClientCredentialLifetime     = 365 * 24 * time.Hour
)

type ZITADELManagementConfig struct {
	BaseURL                  string
	PublicHost               string
	OwnerPATFile             string
	BootstrapStateFile       string
	ClientKeyDefaultLifetime time.Duration
	ClientKeyMaxLifetime     time.Duration
	EnableLegacyPATCreation  bool
	EnableLegacyPATRotation  bool
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

type ManagedClientCredentialSummary struct {
	ID        string
	Type      string
	CreatedAt string
	ChangedAt string
	ExpiresAt string
	Status    string
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
	Credentials []ManagedClientCredentialSummary
}

type ManagedClientKeyCreate struct {
	Username    string
	Name        string
	Description string
	Environment string
	OwnerEmail  string
	Purpose     string
	PublicKey   string
	ExpiresAt   string
}

type ManagedClientKeyAdd struct {
	PublicKey string
	ExpiresAt string
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
}

func (e *ZITADELError) Error() string {
	if code := strings.TrimSpace(e.Code); code != "" {
		return fmt.Sprintf("ZITADEL request failed with status %d (%s)", e.StatusCode, code)
	}
	return fmt.Sprintf("ZITADEL request failed with status %d", e.StatusCode)
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
	bootstrapStateFile := envValue("ARTIFICIALFLOW_ZITADEL_BOOTSTRAP_STATE_FILE")
	if bootstrapStateFile == "" {
		bootstrapStateFile = "/artificialflow/bootstrap/artificialflow-zitadel.json"
	}
	defaultKeyLifetime := boundedDurationEnv(
		"ARTIFICIALFLOW_IAM_CLIENT_KEY_DEFAULT_LIFETIME",
		defaultClientCredentialLifetime,
		time.Hour,
		maxClientCredentialLifetime,
	)
	maxKeyLifetime := boundedDurationEnv(
		"ARTIFICIALFLOW_IAM_CLIENT_KEY_MAX_LIFETIME",
		maxClientCredentialLifetime,
		defaultKeyLifetime,
		maxClientCredentialLifetime,
	)
	return ZITADELManagementConfig{
		BaseURL:                  strings.TrimRight(baseURL, "/"),
		PublicHost:               publicHost,
		OwnerPATFile:             ownerPATFile,
		BootstrapStateFile:       bootstrapStateFile,
		ClientKeyDefaultLifetime: defaultKeyLifetime,
		ClientKeyMaxLifetime:     maxKeyLifetime,
		EnableLegacyPATCreation:  envBool("ARTIFICIALFLOW_IAM_ENABLE_LEGACY_PAT_CREATION"),
		EnableLegacyPATRotation:  envBool("ARTIFICIALFLOW_IAM_ENABLE_LEGACY_PAT_ROTATION"),
	}
}

func envValue(name string) string {
	return strings.TrimSpace(os.Getenv(name))
}

func boundedDurationEnv(name string, fallback, minimum, maximum time.Duration) time.Duration {
	value := envValue(name)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed < minimum || parsed > maximum {
		return fallback
	}
	return parsed
}

func envBool(name string) bool {
	value := envValue(name)
	return strings.EqualFold(value, "true") || value == "1"
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
	content, err := readTrustedConfigFile(path)
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
	content, err := readTrustedConfigFile(path)
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

func parseClientPublicKey(value string) ([]byte, error) {
	raw := []byte(strings.TrimSpace(value))
	if block, _ := pem.Decode(raw); block != nil {
		if block.Type != "PUBLIC KEY" {
			return nil, ErrInvalidClientPublicKey
		}
		raw = block.Bytes
	} else {
		decoded, err := base64.StdEncoding.DecodeString(string(raw))
		if err != nil {
			return nil, ErrInvalidClientPublicKey
		}
		raw = decoded
	}
	parsed, err := x509.ParsePKIXPublicKey(raw)
	if err != nil {
		return nil, ErrInvalidClientPublicKey
	}
	publicKey, ok := parsed.(*rsa.PublicKey)
	if !ok || publicKey.N.BitLen() < 2048 {
		return nil, ErrInvalidClientPublicKey
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: raw}), nil
}

func clientCredentialExpiration(value string, now time.Time) (string, error) {
	return clientCredentialExpirationWithLimits(
		value,
		now,
		defaultClientCredentialLifetime,
		maxClientCredentialLifetime,
	)
}

func clientCredentialExpirationWithLimits(value string, now time.Time, defaultLifetime, maxLifetime time.Duration) (string, error) {
	if defaultLifetime == 0 {
		defaultLifetime = defaultClientCredentialLifetime
	}
	if maxLifetime == 0 {
		maxLifetime = maxClientCredentialLifetime
	}
	if defaultLifetime <= 0 || maxLifetime <= 0 || defaultLifetime > maxLifetime {
		return "", ErrInvalidClientCredentialExpiry
	}
	expiresAt := now.UTC().Add(defaultLifetime)
	if strings.TrimSpace(value) != "" {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
		if err != nil {
			return "", ErrInvalidClientCredentialExpiry
		}
		expiresAt = parsed.UTC()
	}
	if !expiresAt.After(now.UTC()) || expiresAt.After(now.UTC().Add(maxLifetime)) {
		return "", ErrInvalidClientCredentialExpiry
	}
	return expiresAt.Format(time.RFC3339), nil
}

func hostFromURL(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" {
		return ""
	}
	return parsed.Host
}
