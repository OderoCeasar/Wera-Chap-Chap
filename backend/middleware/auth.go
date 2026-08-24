package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/time/rate"

	"wera-chap-chap/backend/models"
)

type Claims struct {
	UserID    uint        `json:"user_id"`
	Role      models.Role `json:"role"`
	TokenType string      `json:"token_type"`
	jwt.RegisteredClaims
}

func JWT(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		claims, err := ParseToken(tokenString, secret)
		if err != nil || claims.TokenType != "access" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		c.Set("user_id", claims.UserID)
		c.Set("role", claims.Role)
		c.Next()
	}
}

func WSJWT(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.Query("token")
		if tokenString == "" {
			tokenString = strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		}
		claims, err := ParseToken(tokenString, secret)
		if err != nil || claims.TokenType != "access" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid websocket token"})
			return
		}
		c.Set("user_id", claims.UserID)
		c.Set("role", claims.Role)
		c.Next()
	}
}

func RequireRole(role models.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		if CurrentRole(c) != role {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient role"})
			return
		}
		c.Next()
	}
}

func SignToken(userID uint, role models.Role, tokenType string, ttl time.Duration, secret string) (string, error) {
	claims := Claims{
		UserID:           userID,
		Role:             role,
		TokenType:        tokenType,
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)), IssuedAt: jwt.NewNumericDate(time.Now())},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

func ParseToken(tokenString, secret string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return nil, err
	}
	return claims, nil
}

func AuthRateLimit() gin.HandlerFunc {
	limiter := rate.NewLimiter(rate.Every(time.Minute), 30)
	return func(c *gin.Context) {
		if !limiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many auth requests"})
			return
		}
		c.Next()
	}
}

func CurrentUserID(c *gin.Context) uint {
	value, _ := c.Get("user_id")
	id, _ := value.(uint)
	return id
}

// CurrentRole reads the role stored by the JWT middleware. The context holds a
// models.Role, so c.GetString would always come back empty.
func CurrentRole(c *gin.Context) models.Role {
	value, _ := c.Get("role")
	role, _ := value.(models.Role)
	return role
}
