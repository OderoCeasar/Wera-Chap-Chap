package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"wera-chap-chap/backend/config"
	"wera-chap-chap/backend/middleware"
	"wera-chap-chap/backend/models"
)

type AuthHandler struct {
	db  *gorm.DB
	cfg config.Config
}

func NewAuthHandler(db *gorm.DB, cfg config.Config) *AuthHandler {
	return &AuthHandler{db: db, cfg: cfg}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var input struct {
		Email    string      `json:"email" binding:"required,email"`
		Password string      `json:"password" binding:"required,min=8"`
		FullName string      `json:"full_name" binding:"required"`
		Phone    string      `json:"phone"`
		Role     models.Role `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if input.Role != models.RoleClient && input.Role != models.RoleTasker {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role must be client or tasker"})
		return
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	user := models.User{Email: input.Email, PasswordHash: string(hash), FullName: input.FullName, Phone: input.Phone, Role: input.Role}
	if input.Role == models.RoleClient {
		user.IsVerified = true
	}
	err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		if user.Role == models.RoleTasker {
			return tx.Create(&models.TaskerProfile{UserID: user.ID, IsAvailable: true, ServiceRadiusKM: 10}).Error
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
		return
	}
	access, refresh := h.tokens(user)
	c.JSON(http.StatusCreated, gin.H{"user": user, "access_token": access, "refresh_token": refresh})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var input struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var user models.User
	if err := h.db.Where("email = ?", input.Email).First(&user).Error; err != nil ||
		bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	access, refresh := h.tokens(user)
	c.JSON(http.StatusOK, gin.H{"user": user, "access_token": access, "refresh_token": refresh})
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var input struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	claims, err := middleware.ParseToken(input.RefreshToken, h.cfg.JWTSecret)
	if err != nil || claims.TokenType != "refresh" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}
	access, err := middleware.SignToken(claims.UserID, claims.Role, "access", h.cfg.AccessTokenTTL, h.cfg.JWTSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not sign token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"access_token": access})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

func (h *AuthHandler) tokens(user models.User) (string, string) {
	access, _ := middleware.SignToken(user.ID, user.Role, "access", h.cfg.AccessTokenTTL, h.cfg.JWTSecret)
	refresh, _ := middleware.SignToken(user.ID, user.Role, "refresh", h.cfg.RefreshTokenTTL, h.cfg.JWTSecret)
	return access, refresh
}
