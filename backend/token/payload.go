package token

import (
	"errors"
	"time"
)

var (
	ErrInvalidToken = errors.New("token is invalid")
	ErrExpiredToken = errors.New("token has expired")
	// ErrWrongTokenType is returned when a token is structurally valid but is
	// the other kind -- a refresh token offered where an access token belongs,
	// or the reverse.
	ErrWrongTokenType = errors.New("token is of the wrong type")
)

// Payload is the decoded contents of a token.
type Payload struct {
	UserID    int64     `json:"user_id"`
	Role      string    `json:"role"`
	TokenType TokenType `json:"token_type"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiredAt time.Time `json:"expired_at"`
}

// NewPayload creates a token payload for a specific user, role and duration.
func NewPayload(userID int64, role string, tokenType TokenType, duration time.Duration) *Payload {
	now := time.Now()
	return &Payload{
		UserID:    userID,
		Role:      role,
		TokenType: tokenType,
		IssuedAt:  now,
		ExpiredAt: now.Add(duration),
	}
}

// Valid checks whether the payload has expired.
func (payload *Payload) Valid() error {
	if time.Now().After(payload.ExpiredAt) {
		return ErrExpiredToken
	}
	return nil
}

// RequireType reports whether the payload is the token kind the caller expects.
func (payload *Payload) RequireType(expected TokenType) error {
	if payload.TokenType != expected {
		return ErrWrongTokenType
	}
	return nil
}
