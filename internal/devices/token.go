package devices

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/saitadikonda99/deployOS/pkg/types"
)

// TokenIssuer issues the signed device token returned to an agent after
// successful registration. It's a separate interface from Repository so
// Service doesn't need to know how tokens are signed.
type TokenIssuer interface {
	Issue(ctx context.Context, device Device) (token string, expiresAt time.Time, err error)
}

// ErrInvalidToken is returned when a device token is missing, malformed,
// expired, or signed with the wrong secret.
var ErrInvalidToken = errors.New("invalid or expired device token")

// deviceClaims are the JWT claims embedded in a device token.
type deviceClaims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

// JWTTokenIssuer issues and verifies HMAC-signed (HS256) JWTs identifying
// a device and its owning user.
type JWTTokenIssuer struct {
	secret []byte
	ttl    time.Duration
}

// NewJWTTokenIssuer builds a JWTTokenIssuer. secret must be non-empty.
func NewJWTTokenIssuer(secret string, ttl time.Duration) *JWTTokenIssuer {
	return &JWTTokenIssuer{secret: []byte(secret), ttl: ttl}
}

// Issue implements TokenIssuer.
func (j *JWTTokenIssuer) Issue(_ context.Context, device Device) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(j.ttl)

	claims := deviceClaims{
		UserID: device.UserID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   device.ID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signed, err := token.SignedString(j.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("signing device token: %w", err)
	}

	return signed, expiresAt, nil
}

// Verify checks a device token's signature and expiry, returning the
// device and user IDs embedded in it. It satisfies
// internal/connection.TokenVerifier, which the gRPC connection server
// uses to authenticate agents - see docs/connection.md.
func (j *JWTTokenIssuer) Verify(_ context.Context, tokenString string) (types.AgentID, string, error) {
	var claims deviceClaims

	token, err := jwt.ParseWithClaims(tokenString, &claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return j.secret, nil
	})
	if err != nil || !token.Valid {
		return "", "", ErrInvalidToken
	}

	if claims.Subject == "" || claims.UserID == "" {
		return "", "", ErrInvalidToken
	}

	return types.AgentID(claims.Subject), claims.UserID, nil
}
