package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"wera-chap-chap/backend/models"
)

const testSecret = "test-secret"

func init() { gin.SetMode(gin.TestMode) }

// request builds a router with the real JWT + RequireRole chain and returns the status.
func request(t *testing.T, required models.Role, token string) int {
	t.Helper()
	router := gin.New()
	router.GET("/guarded", JWT(testSecret), RequireRole(required), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/guarded", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	router.ServeHTTP(recorder, req)
	return recorder.Code
}

func accessToken(t *testing.T, role models.Role) string {
	t.Helper()
	token, err := SignToken(7, role, "access", time.Minute, testSecret)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return token
}

// Regression: the role is stored as models.Role, so a plain string type
// assertion (c.GetString) came back empty and rejected every matching role.
func TestRequireRoleAllowsMatchingRole(t *testing.T) {
	for _, role := range []models.Role{models.RoleClient, models.RoleTasker} {
		if code := request(t, role, accessToken(t, role)); code != http.StatusOK {
			t.Fatalf("role %s: got %d, want 200", role, code)
		}
	}
}

func TestRequireRoleRejectsOtherRole(t *testing.T) {
	if code := request(t, models.RoleClient, accessToken(t, models.RoleTasker)); code != http.StatusForbidden {
		t.Fatalf("got %d, want 403", code)
	}
}

func TestJWTRejectsMissingAndRefreshTokens(t *testing.T) {
	if code := request(t, models.RoleClient, ""); code != http.StatusUnauthorized {
		t.Fatalf("missing token: got %d, want 401", code)
	}
	refresh, err := SignToken(7, models.RoleClient, "refresh", time.Minute, testSecret)
	if err != nil {
		t.Fatalf("sign refresh: %v", err)
	}
	if code := request(t, models.RoleClient, refresh); code != http.StatusUnauthorized {
		t.Fatalf("refresh token on access route: got %d, want 401", code)
	}
}

func TestJWTRejectsWrongSecretAndExpiredToken(t *testing.T) {
	foreign, err := SignToken(7, models.RoleClient, "access", time.Minute, "other-secret")
	if err != nil {
		t.Fatalf("sign foreign: %v", err)
	}
	if code := request(t, models.RoleClient, foreign); code != http.StatusUnauthorized {
		t.Fatalf("foreign secret: got %d, want 401", code)
	}
	expired, err := SignToken(7, models.RoleClient, "access", -time.Minute, testSecret)
	if err != nil {
		t.Fatalf("sign expired: %v", err)
	}
	if code := request(t, models.RoleClient, expired); code != http.StatusUnauthorized {
		t.Fatalf("expired token: got %d, want 401", code)
	}
}

func TestCurrentUserIDAndRole(t *testing.T) {
	router := gin.New()
	var gotID uint
	var gotRole models.Role
	router.GET("/me", JWT(testSecret), func(c *gin.Context) {
		gotID, gotRole = CurrentUserID(c), CurrentRole(c)
		c.Status(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken(t, models.RoleTasker))
	router.ServeHTTP(httptest.NewRecorder(), req)

	if gotID != 7 || gotRole != models.RoleTasker {
		t.Fatalf("got id=%d role=%q, want id=7 role=tasker", gotID, gotRole)
	}
}
