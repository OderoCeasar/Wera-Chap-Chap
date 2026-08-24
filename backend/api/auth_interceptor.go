package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"wera-chap-chap/backend/token"
)

const (
	authorizationHeaderKey  = "Authorization"
	authorizationTypeBearer = "bearer"
	// authorizationPayloadKey is where the verified payload is stashed for the
	// handlers. Read it through currentUserID/currentRole rather than directly.
	authorizationPayloadKey = "authorization_payload"
)

var (
	errMissingAuthHeader = errors.New("authorization header missing")
	errInvalidAuthHeader = errors.New("invalid authorization header format")
	errInvalidToken      = errors.New("invalid or expired token")
	errInsufficientRole  = errors.New("insufficient role")
)

// AuthMiddleware verifies the bearer access token and stores its payload.
func (server *Server) AuthMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		payload, err := server.payloadFromHeader(ctx)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		ctx.Set(authorizationPayloadKey, payload)
		ctx.Next()
	}
}

// WSAuthMiddleware is AuthMiddleware for the WebSocket upgrade, which cannot
// carry an Authorization header from the browser: the WebSocket constructor
// exposes no way to set one, so the token arrives as a query parameter. The
// header is still accepted for non-browser clients.
func (server *Server) WSAuthMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		accessToken := ctx.Query("token")
		if accessToken == "" {
			var err error
			if accessToken, err = bearerFromHeader(ctx); err != nil {
				ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
				return
			}
		}

		payload, err := server.verifyAccessToken(accessToken)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		ctx.Set(authorizationPayloadKey, payload)
		ctx.Next()
	}
}

// RequireRole additionally requires the caller to hold a specific role.
func (server *Server) RequireRole(role string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if currentRole(ctx) != role {
			ctx.AbortWithStatusJSON(http.StatusForbidden, errorResponse(errInsufficientRole))
			return
		}
		ctx.Next()
	}
}

func (server *Server) payloadFromHeader(ctx *gin.Context) (*token.Payload, error) {
	accessToken, err := bearerFromHeader(ctx)
	if err != nil {
		return nil, err
	}
	return server.verifyAccessToken(accessToken)
}

// verifyAccessToken rejects a structurally valid refresh token presented where
// an access token belongs. Without the type check a refresh token -- which
// lives for two weeks and is stored in the browser -- would work as a bearer
// credential everywhere.
func (server *Server) verifyAccessToken(accessToken string) (*token.Payload, error) {
	payload, err := server.tokenMaker.VerifyToken(accessToken)
	if err != nil {
		return nil, errInvalidToken
	}
	if err := payload.RequireType(token.TypeAccess); err != nil {
		return nil, errInvalidToken
	}
	return payload, nil
}

func bearerFromHeader(ctx *gin.Context) (string, error) {
	header := ctx.GetHeader(authorizationHeaderKey)
	if header == "" {
		return "", errMissingAuthHeader
	}

	fields := strings.SplitN(header, " ", 2)
	if len(fields) != 2 || strings.ToLower(fields[0]) != authorizationTypeBearer {
		return "", errInvalidAuthHeader
	}
	return fields[1], nil
}

// authPayload returns the verified payload the middleware stored. It is only
// called from handlers behind AuthMiddleware, so a missing payload is a routing
// mistake rather than a request the caller can cause.
func authPayload(ctx *gin.Context) *token.Payload {
	value, exists := ctx.Get(authorizationPayloadKey)
	if !exists {
		return nil
	}
	payload, _ := value.(*token.Payload)
	return payload
}

func currentUserID(ctx *gin.Context) int64 {
	if payload := authPayload(ctx); payload != nil {
		return payload.UserID
	}
	return 0
}

func currentRole(ctx *gin.Context) string {
	if payload := authPayload(ctx); payload != nil {
		return payload.Role
	}
	return ""
}
