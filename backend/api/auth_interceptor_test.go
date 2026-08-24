package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"wera-chap-chap/backend/config"
	db "wera-chap-chap/backend/db/sqlc"
	"wera-chap-chap/backend/token"
)

// testSecret is 32 characters, the minimum NewJWTMaker accepts.
const testSecret = "test-secret-test-secret-test-sec"

func init() { gin.SetMode(gin.TestMode) }

// newTestServer builds a Server with only the pieces the auth middleware
// touches. The store is nil on purpose: no middleware test reaches it, and a
// nil dereference would be a clearer failure than a silently-passing fake.
func newTestServer(t *testing.T) *Server {
	t.Helper()
	maker, err := token.NewJWTMaker(testSecret)
	if err != nil {
		t.Fatalf("new jwt maker: %v", err)
	}
	return &Server{
		config:     config.Config{JWTSecret: testSecret},
		tokenMaker: maker,
	}
}

// guardedRequest drives the real AuthMiddleware + RequireRole chain and returns
// the status code.
func guardedRequest(t *testing.T, required string, accessToken string) int {
	t.Helper()
	server := newTestServer(t)

	router := gin.New()
	router.GET("/guarded", server.AuthMiddleware(), server.RequireRole(required), func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"ok": true})
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/guarded", nil)
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}
	router.ServeHTTP(recorder, req)
	return recorder.Code
}

func mintToken(t *testing.T, role string, tokenType token.TokenType, duration time.Duration, secret string) string {
	t.Helper()
	maker, err := token.NewJWTMaker(secret)
	if err != nil {
		t.Fatalf("new jwt maker: %v", err)
	}
	signed, _, err := maker.CreateToken(7, role, tokenType, duration)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	return signed
}

func accessTokenFor(t *testing.T, role string) string {
	t.Helper()
	return mintToken(t, role, token.TypeAccess, time.Minute, testSecret)
}

func TestRequireRoleAllowsMatchingRole(t *testing.T) {
	for _, role := range []string{db.RoleClient, db.RoleTasker} {
		if code := guardedRequest(t, role, accessTokenFor(t, role)); code != http.StatusOK {
			t.Fatalf("role %s: got %d, want 200", role, code)
		}
	}
}

func TestRequireRoleRejectsOtherRole(t *testing.T) {
	if code := guardedRequest(t, db.RoleClient, accessTokenFor(t, db.RoleTasker)); code != http.StatusForbidden {
		t.Fatalf("got %d, want 403", code)
	}
}

// A refresh token lives for weeks and sits in browser storage. Accepting one as
// a bearer credential would make that storage equivalent to a long-lived
// session, which is what RequireType guards against.
func TestAuthMiddlewareRejectsMissingAndRefreshTokens(t *testing.T) {
	if code := guardedRequest(t, db.RoleClient, ""); code != http.StatusUnauthorized {
		t.Fatalf("missing token: got %d, want 401", code)
	}

	refresh := mintToken(t, db.RoleClient, token.TypeRefresh, time.Minute, testSecret)
	if code := guardedRequest(t, db.RoleClient, refresh); code != http.StatusUnauthorized {
		t.Fatalf("refresh token on access route: got %d, want 401", code)
	}
}

func TestAuthMiddlewareRejectsWrongSecretAndExpiredToken(t *testing.T) {
	foreign := mintToken(t, db.RoleClient, token.TypeAccess, time.Minute, "other-secret-other-secret-other!")
	if code := guardedRequest(t, db.RoleClient, foreign); code != http.StatusUnauthorized {
		t.Fatalf("foreign secret: got %d, want 401", code)
	}

	expired := mintToken(t, db.RoleClient, token.TypeAccess, -time.Minute, testSecret)
	if code := guardedRequest(t, db.RoleClient, expired); code != http.StatusUnauthorized {
		t.Fatalf("expired token: got %d, want 401", code)
	}
}

func TestAuthMiddlewareRejectsMalformedHeader(t *testing.T) {
	server := newTestServer(t)
	router := gin.New()
	router.GET("/guarded", server.AuthMiddleware(), func(ctx *gin.Context) {
		ctx.Status(http.StatusOK)
	})

	for _, header := range []string{"token-without-scheme", "Basic abc", "Bearer"} {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/guarded", nil)
		req.Header.Set("Authorization", header)
		router.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("header %q: got %d, want 401", header, recorder.Code)
		}
	}
}

func TestCurrentUserIDAndRole(t *testing.T) {
	server := newTestServer(t)

	var gotID int64
	var gotRole string
	router := gin.New()
	router.GET("/me", server.AuthMiddleware(), func(ctx *gin.Context) {
		gotID, gotRole = currentUserID(ctx), currentRole(ctx)
		ctx.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer "+accessTokenFor(t, db.RoleTasker))
	router.ServeHTTP(httptest.NewRecorder(), req)

	if gotID != 7 || gotRole != db.RoleTasker {
		t.Fatalf("got id=%d role=%q, want id=7 role=tasker", gotID, gotRole)
	}
}

// The WebSocket upgrade cannot carry an Authorization header from a browser, so
// it accepts the token as a query parameter -- but must still reject a refresh
// token and an absent one.
func TestWSAuthMiddleware(t *testing.T) {
	server := newTestServer(t)
	router := gin.New()
	router.GET("/ws", server.WSAuthMiddleware(), func(ctx *gin.Context) {
		ctx.Status(http.StatusOK)
	})

	cases := []struct {
		name  string
		query string
		want  int
	}{
		{"valid access token", "?token=" + accessTokenFor(t, db.RoleTasker), http.StatusOK},
		{"no token", "", http.StatusUnauthorized},
		{"refresh token", "?token=" + mintToken(t, db.RoleTasker, token.TypeRefresh, time.Minute, testSecret), http.StatusUnauthorized},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/ws"+testCase.query, nil))
			if recorder.Code != testCase.want {
				t.Fatalf("got %d, want %d", recorder.Code, testCase.want)
			}
		})
	}
}
