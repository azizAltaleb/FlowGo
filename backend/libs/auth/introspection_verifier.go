package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type introspectionVerifier struct {
	client *http.Client
	config Config
}

func newIntrospectionVerifier(cfg Config) (TokenVerifier, error) {
	introspectionURL := strings.TrimSpace(cfg.IntrospectionURL)
	if introspectionURL == "" {
		return nil, fmt.Errorf("AUTH_INTROSPECTION_URL is required when AUTH_TOKEN_MODE=introspection")
	}
	normalizedURL, err := validateIntrospectionURL(introspectionURL)
	if err != nil {
		return nil, err
	}
	cfg.IntrospectionURL = normalizedURL
	return &introspectionVerifier{
		client: &http.Client{Timeout: 10 * time.Second},
		config: cfg,
	}, nil
}

func validateIntrospectionURL(raw string) (string, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parse AUTH_INTROSPECTION_URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("AUTH_INTROSPECTION_URL must use http or https")
	}
	if strings.TrimSpace(parsed.Host) == "" {
		return "", fmt.Errorf("AUTH_INTROSPECTION_URL must include a host")
	}
	if parsed.User != nil {
		return "", fmt.Errorf("AUTH_INTROSPECTION_URL must not include userinfo")
	}
	return parsed.String(), nil
}

func (v *introspectionVerifier) Verify(ctx context.Context, rawToken string) (*Principal, error) {
	body := url.Values{}
	body.Set("token", rawToken)
	if strings.EqualFold(v.config.IntrospectionAuthMethod, "post") {
		if v.config.IntrospectionClientID != "" {
			body.Set("client_id", v.config.IntrospectionClientID)
		}
		if v.config.IntrospectionClientSecret != "" {
			body.Set("client_secret", v.config.IntrospectionClientSecret)
		}
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, v.config.IntrospectionURL, strings.NewReader(body.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build introspection request: %w", err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Accept", "application/json")
	if !strings.EqualFold(v.config.IntrospectionAuthMethod, "post") && v.config.IntrospectionClientID != "" {
		request.SetBasicAuth(v.config.IntrospectionClientID, v.config.IntrospectionClientSecret)
	}

	response, err := v.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("token introspection request failed: %w", err)
	}
	defer response.Body.Close()

	payloadRaw, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("read introspection response: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token introspection failed with status=%d body=%s", response.StatusCode, strings.TrimSpace(string(payloadRaw)))
	}

	claims := map[string]any{}
	if err := json.Unmarshal(payloadRaw, &claims); err != nil {
		return nil, fmt.Errorf("decode introspection response: %w", err)
	}
	if !valueAsBool(claims["active"]) {
		return nil, fmt.Errorf("inactive token")
	}

	subject := firstNonEmpty(
		valueAsString(claims["sub"]),
		valueAsString(claims["username"]),
	)
	principal := principalFromClaims(claims, subject, v.config, TokenModeIntrospection)
	if issuer := valueAsString(claims["iss"]); issuer != "" {
		principal.Issuer = issuer
	}
	if len(principal.Audience) == 0 {
		principal.Audience = claimFirstStringSlice(claims, "aud,client_id,azp")
	}

	if v.config.EnforceAudience && strings.TrimSpace(v.config.ClientID) != "" {
		audienceMatch := false
		for _, aud := range principal.Audience {
			if strings.EqualFold(strings.TrimSpace(aud), strings.TrimSpace(v.config.ClientID)) {
				audienceMatch = true
				break
			}
		}
		if !audienceMatch {
			clientIDClaim := valueAsString(claims["client_id"])
			azpClaim := valueAsString(claims["azp"])
			if !strings.EqualFold(strings.TrimSpace(clientIDClaim), strings.TrimSpace(v.config.ClientID)) &&
				!strings.EqualFold(strings.TrimSpace(azpClaim), strings.TrimSpace(v.config.ClientID)) {
				return nil, fmt.Errorf("token audience/client mismatch")
			}
		}
	}

	return principal, nil
}

func valueAsBool(value any) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		normalized := strings.TrimSpace(strings.ToLower(v))
		return normalized == "true" || normalized == "1" || normalized == "yes" || normalized == "on"
	case float64:
		return v != 0
	default:
		return false
	}
}
