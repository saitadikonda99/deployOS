package devices

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/saitadikonda99/deployOS/pkg/types"
)

func TestJWTTokenIssuerIssuesVerifiableToken(t *testing.T) {
	issuer := NewJWTTokenIssuer("test-secret", time.Hour)
	device := Device{ID: types.AgentID("11111111-1111-1111-1111-111111111111"), UserID: "user-1"}

	token, expiresAt, err := issuer.Issue(context.Background(), device)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	if token == "" {
		t.Fatal("Issue() returned empty token")
	}
	if !expiresAt.After(time.Now()) {
		t.Fatalf("expiresAt = %v, want a time in the future", expiresAt)
	}

	var claims deviceClaims
	parsed, err := jwt.ParseWithClaims(token, &claims, func(*jwt.Token) (any, error) {
		return []byte("test-secret"), nil
	})
	if err != nil {
		t.Fatalf("parsing issued token: %v", err)
	}
	if !parsed.Valid {
		t.Fatal("issued token is not valid")
	}
	if claims.Subject != device.ID.String() {
		t.Errorf("Subject = %q, want %q", claims.Subject, device.ID.String())
	}
	if claims.UserID != device.UserID {
		t.Errorf("UserID = %q, want %q", claims.UserID, device.UserID)
	}
}

func TestJWTTokenIssuerRejectsWrongSecret(t *testing.T) {
	issuer := NewJWTTokenIssuer("test-secret", time.Hour)
	device := Device{ID: types.AgentID("11111111-1111-1111-1111-111111111111"), UserID: "user-1"}

	token, _, err := issuer.Issue(context.Background(), device)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	var claims deviceClaims
	_, err = jwt.ParseWithClaims(token, &claims, func(*jwt.Token) (any, error) {
		return []byte("wrong-secret"), nil
	})
	if err == nil {
		t.Fatal("expected an error parsing a token with the wrong secret")
	}
}
