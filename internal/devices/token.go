package devices

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenIssuer issues the signed device token returned to an agent after
// successful registration. It's a separate interface from Repository so
// Service doesn't need to know how tokens are signed.
type TokenIssuer interface {
	Issue(ctx context.Context, device Device) (token string, expiresAt time.Time, err error)
}

// deviceClaims are the JWT claims embedded in a device token.
type deviceClaims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

// JWTTokenIssuer issues HMAC-signed (HS256) JWTs identifying a device
// and its owning user. Verifying these tokens is future work (it lands
// with the heartbeat feature, which needs to authenticate agents);
// Issue is all device registration requires today.
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
