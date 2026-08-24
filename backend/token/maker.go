package token

import "time"

// TokenType distinguishes the two tokens this service mints. It is carried
// inside the token itself so a refresh token cannot be presented as an access
// token: the auth middleware accepts only TypeAccess, and the refresh endpoint
// only TypeRefresh.
type TokenType string

const (
	TypeAccess  TokenType = "access"
	TypeRefresh TokenType = "refresh"
)

// Maker is an interface for managing tokens.
//
// The API layer depends on this rather than on a signing library, so the
// algorithm is one package's business. Swapping JWT for PASETO means adding a
// file here and changing the constructor call in main.
type Maker interface {
	// CreateToken creates a new token for a specific user, role and duration.
	CreateToken(userID int64, role string, tokenType TokenType, duration time.Duration) (string, *Payload, error)

	// VerifyToken checks if the token is valid or not.
	VerifyToken(token string) (*Payload, error)
}
