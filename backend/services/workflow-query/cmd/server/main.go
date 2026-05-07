package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/azizAltaleb/goflow/backend/libs/auth"
	esadapter "github.com/azizAltaleb/goflow/backend/libs/elasticsearch"
	"github.com/azizAltaleb/goflow/backend/libs/logger"
	"github.com/azizAltaleb/goflow/backend/services/workflow-query/internal/application"
	"github.com/azizAltaleb/goflow/backend/services/workflow-query/internal/infrastructure/persistence"
	api "github.com/azizAltaleb/goflow/backend/services/workflow-query/internal/interfaces/http"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

func main() {
	log := logger.New("workflow-query")
	ctx := context.Background()

	esAddr := os.Getenv("ES_ADDR")
	if esAddr == "" {
		esAddr = "http://localhost:9200"
	}
	indexPrefix := os.Getenv("ES_INDEX_PREFIX")
	rawSearchBackend := os.Getenv("SEARCH_BACKEND")
	searchBackend, fallbackToDefault := parseSearchBackend(rawSearchBackend)
	if fallbackToDefault {
		log.Warn(ctx, "unsupported search backend configured; falling back to elasticsearch", map[string]any{
			"configured_backend": strings.TrimSpace(rawSearchBackend),
		})
	}

	addr := os.Getenv("QUERY_SERVER_ADDR")
	if addr == "" {
		addr = "0.0.0.0:8081"
	}

	// Auth Configuration
	authConfig := auth.ResolveConfigFromEnv()
	internalIssuer := authConfig.InternalIssuerURL
	externalIssuer := authConfig.ExternalIssuerURL
	clientID := authConfig.ClientID

	log.Info(ctx, "initializing workflow-query service", map[string]any{
		"es_addr":         esAddr,
		"search_backend":  searchBackend,
		"index_prefix":    indexPrefix,
		"server_addr":     addr,
		"internal_issuer": internalIssuer,
		"external_issuer": externalIssuer,
		"auth_client_id":  clientID,
		"auth_mode":       authConfig.TokenValidationMode,
		"auth_enabled":    authConfig.Enabled(),
	})

	// Initialize Auth Middleware
	var authMiddleware *auth.Middleware
	authMiddleware, err := auth.NewMiddleware(ctx, authConfig)
	if err != nil {
		log.Error(ctx, "failed to initialize auth middleware", map[string]any{"error": err.Error()})
		os.Exit(1)
	}
	log.Info(ctx, "auth middleware initialized", nil)

	if searchBackend == "opensearch" {
		log.Info(ctx, "opensearch compatibility mode enabled", nil)
	}

	client, err := esadapter.NewClient(esadapter.Config{Addresses: []string{esAddr}})
	if err != nil {
		log.Error(ctx, "failed to init search client", map[string]any{"error": err.Error(), "backend": searchBackend})
		os.Exit(1)
	}
	esRepo := esadapter.NewRepository(client)
	log.Info(ctx, "search client initialized", map[string]any{"backend": searchBackend})

	// Initialize Domain Repository & Application Service
	queryRepo := persistence.NewESRepository(esRepo, indexPrefix)
	queryService := application.NewQueryService(queryRepo)

	// Initialize Handler
	handler := api.NewHandler(queryService)

	r := mux.NewRouter()

	// Apply Auth Middleware if enabled
	if authMiddleware != nil {
		r.Use(authMiddleware.Handler)
		log.Info(ctx, "auth middleware enabled on router", nil)
	}

	r.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
	}).Methods("GET")
	r.HandleFunc("/identity/me", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		principal, ok := auth.PrincipalFromContext(r.Context())
		if !ok {
			_ = json.NewEncoder(w).Encode(map[string]any{"authenticated": false})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"authenticated": true,
			"principal":     principal,
		})
	}).Methods("GET")

	// Domain-specific endpoints
	handler.RegisterRoutes(r)

	// Add logging middleware
	httpMiddleware := logger.NewHTTPMiddleware("http")
	allowedOrigins := parseAllowedOrigins(os.Getenv("ALLOWED_ORIGINS"))
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{"http://localhost:9100", "http://localhost:5173"}
	}
	allowCredentials := envBool("CORS_ALLOW_CREDENTIALS", false)
	if len(allowedOrigins) == 1 && allowedOrigins[0] == "*" && allowCredentials {
		log.Warn(ctx, "cors credentials disabled for wildcard origin", nil)
		allowCredentials = false
	}

	c := cors.New(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type", "X-Correlation-ID"},
		AllowCredentials: allowCredentials,
	})

	srv := &http.Server{
		Handler:      httpMiddleware.Middleware(c.Handler(r)),
		Addr:         addr,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigCh
		log.Info(ctx, "received shutdown signal", map[string]any{"signal": sig.String()})
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Error(ctx, "shutdown error", map[string]any{"error": err.Error()})
		}
	}()

	log.Info(ctx, "server starting", map[string]any{"addr": addr})
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Error(ctx, "server error", map[string]any{"error": err.Error()})
		os.Exit(1)
	}
	log.Info(ctx, "server stopped gracefully", nil)
}

func parseAllowedOrigins(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))
	for _, p := range parts {
		v := strings.TrimSpace(p)
		if v == "" {
			continue
		}
		origins = append(origins, v)
	}
	return origins
}

func envBool(key string, defaultValue bool) bool {
	val := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	switch val {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return defaultValue
	}
}

func parseSearchBackend(raw string) (backend string, fallbackToDefault bool) {
	parsed := strings.ToLower(strings.TrimSpace(raw))
	if parsed == "" {
		return "elasticsearch", false
	}
	if parsed == "elasticsearch" || parsed == "opensearch" {
		return parsed, false
	}
	return "elasticsearch", true
}
