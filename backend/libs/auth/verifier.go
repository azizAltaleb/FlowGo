package auth

import "context"

type TokenVerifier interface {
	Verify(ctx context.Context, rawToken string) (*Principal, error)
}
