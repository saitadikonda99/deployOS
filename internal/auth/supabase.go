package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// SupabaseAuthenticator authenticates access tokens by asking Supabase
// Auth's GET /auth/v1/user endpoint who they belong to. This works
// regardless of the JWT signing algorithm Supabase uses internally, and
// keeps token verification logic on Supabase's side rather than
// duplicating it here.
type SupabaseAuthenticator struct {
	baseURL string
	anonKey string
	client  *http.Client
}

// NewSupabaseAuthenticator builds a SupabaseAuthenticator. baseURL is the
// Supabase project URL (e.g. "https://xyz.supabase.co") and anonKey is
// the project's anon/public API key.
func NewSupabaseAuthenticator(baseURL, anonKey string) *SupabaseAuthenticator {
	return &SupabaseAuthenticator{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		anonKey: anonKey,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

type supabaseUserResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

// Authenticate implements Authenticator.
func (a *SupabaseAuthenticator) Authenticate(ctx context.Context, accessToken string) (User, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.baseURL+"/auth/v1/user", nil)
	if err != nil {
		return User{}, fmt.Errorf("building supabase auth request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("apikey", a.anonKey)

	resp, err := a.client.Do(req)
	if err != nil {
		return User{}, fmt.Errorf("calling supabase auth: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return User{}, ErrInvalidToken
		}
		return User{}, fmt.Errorf("supabase auth returned status %d", resp.StatusCode)
	}

	var body supabaseUserResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return User{}, fmt.Errorf("decoding supabase auth response: %w", err)
	}
	if body.ID == "" {
		return User{}, ErrInvalidToken
	}

	return User(body), nil
}
