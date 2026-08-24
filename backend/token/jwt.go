package token

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// minSecretKeySize is the HMAC-SHA256 block size. A shorter key is not rejected
// by the library but weakens the signature, and a short key here is always a
// placeholder rather than a deliberate choice.
const minSecretKeySize = 32

// claims is the wire form of Payload. It stays unexported so the token package
// is the only thing that knows this is JWT.
type claims struct {
	UserID    int64     `json:"user_id"`
	Role      string    `json:"role"`
	TokenType TokenType `json:"token_type"`
	jwt.RegisteredClaims
}

// JWTMaker is a JSON Web Token maker using HMAC-SHA256.
type JWTMaker struct {
	secretKey string
}

// NewJWTMaker creates a new JWTMaker.
func NewJWTMaker(secretKey string) (*JWTMaker, error) {
	if len(secretKey) < minSecretKeySize {
		return nil, fmt.Errorf("invalid key size: must be at least %d characters", minSecretKeySize)
	}
	return &JWTMaker{secretKey: secretKey}, nil
}

// CreateToken creates a new token for a specific user, role and duration.
func (maker *JWTMaker) CreateToken(userID int64, role string, tokenType TokenType, duration time.Duration) (string, *Payload, error) {
	payload := NewPayload(userID, role, tokenType, duration)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		UserID:    payload.UserID,
		Role:      payload.Role,
		TokenType: payload.TokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(payload.IssuedAt),
			ExpiresAt: jwt.NewNumericDate(payload.ExpiredAt),
		},
	})

	signed, err := token.SignedString([]byte(maker.secretKey))
	if err != nil {
		return "", nil, err
	}
	return signed, payload, nil
}

// VerifyToken checks if the token is valid or not.
func (maker *JWTMaker) VerifyToken(token string) (*Payload, error) {
	keyFunc := func(t *jwt.Token) (interface{}, error) {
		// Without this check any token claiming alg:"none" -- or an asymmetric
		// alg verified against our secret as a public key -- would be accepted.
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(maker.secretKey), nil
	}

	parsed, err := jwt.ParseWithClaims(token, &claims{}, keyFunc)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	decoded, ok := parsed.Claims.(*claims)
	if !ok || !parsed.Valid {
		return nil, ErrInvalidToken
	}

	payload := &Payload{
		UserID:    decoded.UserID,
		Role:      decoded.Role,
		TokenType: decoded.TokenType,
	}
	if decoded.IssuedAt != nil {
		payload.IssuedAt = decoded.IssuedAt.Time
	}
	if decoded.ExpiresAt != nil {
		payload.ExpiredAt = decoded.ExpiresAt.Time
	}
	return payload, nil
}
