package auth

import "context"

type Principal struct {
	Subject   string         `json:"subject"`
	Issuer    string         `json:"issuer,omitempty"`
	Audience  []string       `json:"audience,omitempty"`
	Email     string         `json:"email,omitempty"`
	Name      string         `json:"name,omitempty"`
	TenantID  string         `json:"tenant_id,omitempty"`
	Roles     []string       `json:"roles,omitempty"`
	Scopes    []string       `json:"scopes,omitempty"`
	TokenMode string         `json:"token_mode,omitempty"`
	Claims    map[string]any `json:"claims,omitempty"`
}

type principalContextKey struct{}

func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalContextKey{}, principal)
}

func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	value := ctx.Value(principalContextKey{})
	if value == nil {
		return Principal{}, false
	}
	principal, ok := value.(Principal)
	if !ok {
		return Principal{}, false
	}
	return principal, true
}
