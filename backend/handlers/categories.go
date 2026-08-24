package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"wera-chap-chap/backend/models"
)

func ListCategories(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var categories []models.Category
		db.Order("name asc").Find(&categories)
		c.JSON(http.StatusOK, categories)
	}
}
