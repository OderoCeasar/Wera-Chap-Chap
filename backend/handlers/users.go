package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"wera-chap-chap/backend/middleware"
	"wera-chap-chap/backend/models"
)

type UserHandler struct{ db *gorm.DB }

func NewUserHandler(db *gorm.DB) *UserHandler { return &UserHandler{db: db} }

func (h *UserHandler) Me(c *gin.Context) {
	var user models.User
	if err := h.db.Preload("TaskerProfile").First(&user, middleware.CurrentUserID(c)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, user)
}

func (h *UserHandler) UpdateMe(c *gin.Context) {
	var input struct {
		FullName  string `json:"full_name"`
		Phone     string `json:"phone"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	updates := map[string]interface{}{"full_name": input.FullName, "phone": input.Phone, "avatar_url": input.AvatarURL}
	h.db.Model(&models.User{}).Where("id = ?", middleware.CurrentUserID(c)).Updates(updates)
	h.Me(c)
}

func (h *UserHandler) ChangePassword(c *gin.Context) {
	var input struct {
		CurrentPassword string `json:"current_password" binding:"required"`
		NewPassword     string `json:"new_password" binding:"required,min=8"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var user models.User
	h.db.First(&user, middleware.CurrentUserID(c))
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.CurrentPassword)) != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "current password is incorrect"})
		return
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	h.db.Model(&user).Update("password_hash", string(hash))
	c.JSON(http.StatusOK, gin.H{"message": "password updated"})
}
