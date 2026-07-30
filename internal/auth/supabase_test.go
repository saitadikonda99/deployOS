package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSupabaseAuthenticatorAuthenticateSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Path, "/auth/v1/user"; got != want {
			t.Errorf("path = %q, want %q", got, want)
		}
		if got, want := r.Header.Get("Authorization"), "Bearer valid-token"; got != want {
			t.Errorf("Authorization header = %q, want %q", got, want)
		}
		if got, want := r.Header.Get("apikey"), "anon-key"; got != want {
			t.Errorf("apikey header = %q, want %q", got, want)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(supabaseUserResponse{ID: "user-1", Email: "operator@example.com"})
	}))
	defer srv.Close()

	authr := NewSupabaseAuthenticator(srv.URL, "anon-key")

	user, err := authr.Authenticate(context.Background(), "valid-token")
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if user.ID != "user-1" || user.Email != "operator@example.com" {
		t.Errorf("user = %+v, want {ID: user-1, Email: operator@example.com}", user)
	}
}

func TestSupabaseAuthenticatorAuthenticateRejectsInvalidToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	authr := NewSupabaseAuthenticator(srv.URL, "anon-key")

	_, err := authr.Authenticate(context.Background(), "bad-token")
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("Authenticate() error = %v, want ErrInvalidToken", err)
	}
}

func TestBearerToken(t *testing.T) {
	tests := []struct {
		name    string
		header  string
		want    string
		wantOK  bool
		comment string
	}{
		{name: "valid", header: "Bearer abc123", want: "abc123", wantOK: true},
		{name: "missing", header: "", want: "", wantOK: false},
		{name: "wrong scheme", header: "Basic abc123", want: "", wantOK: false},
		{name: "empty token", header: "Bearer ", want: "", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}

			got, ok := BearerToken(req)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("BearerToken() = (%q, %v), want (%q, %v)", got, ok, tt.want, tt.wantOK)
			}
		})
	}
}
