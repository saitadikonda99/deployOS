// Package auth authenticates DeployOS operators against Supabase Auth.
// The control plane is the only DeployOS component that holds Supabase
// credentials; agents and the dashboard authenticate through it, never
// against Supabase directly.
package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"
)

// User is the authenticated operator behind a request.
type User struct {
	ID    string
	Email string
}

// ErrInvalidToken is returned when a token is missing, malformed, expired,
// or rejected by the upstream identity provider.
var ErrInvalidToken = errors.New("invalid or expired access token")

// Authenticator resolves a bearer access token to the User it belongs to.
type Authenticator interface {
	Authenticate(ctx context.Context, accessToken string) (User, error)
}

// BearerToken extracts the token from an "Authorization: Bearer <token>"
// header. It returns false if the header is missing or malformed.
func BearerToken(r *http.Request) (string, bool) {
	header := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return "", false
	}
	token := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	if token == "" {
		return "", false
	}
	return token, true
}
