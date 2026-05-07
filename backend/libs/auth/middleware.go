package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/azizAltaleb/goflow/backend/libs/logger"
)

type Middleware struct {
	provider       ConfigProvider
	logger         *logger.Logger
	publicPaths    map[string]struct{}
	mu             sync.RWMutex
	verifier       TokenVerifier
	verifierConfig string
}

func NewMiddleware(ctx context.Context, cfg Config) (*Middleware, error) {
	return NewDynamicMiddleware(ctx, StaticConfigProvider{Config: cfg}, nil)
}

func NewDynamicMiddleware(ctx context.Context, provider ConfigProvider, publicPaths []string) (*Middleware, error) {
	log := logger.New("auth-middleware")
	if provider == nil {
		return nil, fmt.Errorf("auth middleware requires a config provider")
	}
	paths := map[string]struct{}{
		"/health": {},
	}
	for _, path := range publicPaths {
		trimmed := strings.TrimSpace(path)
		if trimmed == "" {
			continue
		}
		paths[trimmed] = struct{}{}
	}
	m := &Middleware{
		provider:    provider,
		logger:      log,
		publicPaths: paths,
	}
	initialConfig, err := provider.GetConfig(ctx)
	if err != nil {
		return nil, err
	}
	if initialConfig.Enabled() {
		verifier, err := m.buildVerifier(ctx, initialConfig)
		if err != nil {
			return nil, err
		}
		m.verifier = verifier
		m.verifierConfig = configFingerprint(initialConfig)
	}
	return m, nil
}

// Handler verifies the Bearer token and adds claims to the context
func (m *Middleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := m.publicPaths[r.URL.Path]; ok {
			next.ServeHTTP(w, r)
			return
		}

		cfg, err := m.provider.GetConfig(r.Context())
		if err != nil {
			m.logger.Error(r.Context(), "failed to resolve auth configuration", map[string]any{"error": err.Error()})
			http.Error(w, "Authentication configuration unavailable", http.StatusServiceUnavailable)
			return
		}
		if !cfg.Enabled() {
			next.ServeHTTP(w, r)
			return
		}
		verifier, err := m.getVerifier(r.Context(), cfg)
		if err != nil {
			m.logger.Error(r.Context(), "failed to initialize token verifier", map[string]any{"error": err.Error()})
			http.Error(w, "Authentication configuration unavailable", http.StatusServiceUnavailable)
			return
		}

		rawAccessToken := extractBearerToken(r.Header.Get("Authorization"))

		if rawAccessToken == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		principal, err := verifier.Verify(r.Context(), rawAccessToken)
		if err != nil {
			m.logger.Error(r.Context(), "token verification failed", map[string]any{"error": err.Error()})
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Add claims to context
		ctx := WithPrincipal(r.Context(), *principal)
		ctx = context.WithValue(ctx, "user_claims", principal.Claims)
		// Also standard sub/user_id
		ctx = context.WithValue(ctx, "user_id", principal.Subject)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *Middleware) getVerifier(ctx context.Context, cfg Config) (TokenVerifier, error) {
	fingerprint := configFingerprint(cfg)
	m.mu.RLock()
	if m.verifier != nil && m.verifierConfig == fingerprint {
		verifier := m.verifier
		m.mu.RUnlock()
		return verifier, nil
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.verifier != nil && m.verifierConfig == fingerprint {
		return m.verifier, nil
	}
	verifier, err := m.buildVerifier(ctx, cfg)
	if err != nil {
		return nil, err
	}
	m.verifier = verifier
	m.verifierConfig = fingerprint
	return verifier, nil
}

func (m *Middleware) buildVerifier(ctx context.Context, cfg Config) (TokenVerifier, error) {
	switch cfg.TokenValidationMode {
	case TokenModeIntrospection:
		return newIntrospectionVerifier(cfg)
	default:
		return newJWTVerifier(ctx, cfg, m.logger)
	}
}

func configFingerprint(cfg Config) string {
	parts := []string{
		strings.TrimSpace(cfg.InternalIssuerURL),
		strings.TrimSpace(cfg.ExternalIssuerURL),
		strings.TrimSpace(cfg.ClientID),
		strings.TrimSpace(cfg.TokenValidationMode),
		strings.TrimSpace(cfg.IntrospectionURL),
		strings.TrimSpace(cfg.IntrospectionClientID),
		cfg.IntrospectionClientSecret,
		strings.TrimSpace(cfg.IntrospectionAuthMethod),
		strings.TrimSpace(cfg.ClaimSubjectPath),
		strings.TrimSpace(cfg.ClaimRolesPath),
		strings.TrimSpace(cfg.ClaimScopesPath),
		strings.TrimSpace(cfg.ClaimTenantPath),
		strings.TrimSpace(cfg.ClaimEmailPath),
		strings.TrimSpace(cfg.ClaimNamePath),
	}
	if cfg.EnforceAudience {
		parts = append(parts, "audience:on")
	} else {
		parts = append(parts, "audience:off")
	}
	if cfg.AllowInsecureIssuer {
		parts = append(parts, "issuer:insecure")
	} else {
		parts = append(parts, "issuer:strict")
	}
	return strings.Join(parts, "|")
}

func extractBearerToken(rawAuthorization string) string {
	authorization := strings.TrimSpace(rawAuthorization)
	if authorization == "" {
		return ""
	}
	parts := strings.SplitN(authorization, " ", 2)
	if len(parts) != 2 {
		return ""
	}
	if !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}
