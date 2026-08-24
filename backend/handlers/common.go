package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"wera-chap-chap/backend/middleware"
	"wera-chap-chap/backend/models"
)

// bookingScope describes how the current user relates to a booking.
type bookingScope struct {
	Booking  models.Booking
	IsClient bool
	IsTasker bool
}

// loadBooking fetches the booking named by param and confirms the caller is a
// participant. It writes the error response itself and reports whether the
// caller may continue.
func loadBooking(db *gorm.DB, c *gin.Context, param string, preload bool) (bookingScope, bool) {
	var scope bookingScope
	id, err := strconv.ParseUint(c.Param(param), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid booking id"})
		return scope, false
	}

	query := db.Session(&gorm.Session{})
	if preload {
		query = query.Preload("Task.Category").Preload("Client").Preload("Tasker.User")
	}
	if err := query.First(&scope.Booking, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "booking not found"})
		return scope, false
	}

	userID := middleware.CurrentUserID(c)
	scope.IsClient = scope.Booking.ClientID == userID
	if !scope.IsClient {
		if profile, ok := taskerProfile(db, userID); ok && profile.ID == scope.Booking.TaskerID {
			scope.IsTasker = true
		}
	}
	if !scope.IsClient && !scope.IsTasker {
		c.JSON(http.StatusForbidden, gin.H{"error": "not your booking"})
		return scope, false
	}
	return scope, true
}

// taskerProfile loads the tasker profile owned by userID.
func taskerProfile(db *gorm.DB, userID uint) (models.TaskerProfile, bool) {
	var profile models.TaskerProfile
	if err := db.Where("user_id = ?", userID).First(&profile).Error; err != nil {
		return profile, false
	}
	return profile, true
}

// currentTaskerProfile resolves the caller's tasker profile, responding with
// 403 when they do not have one.
func currentTaskerProfile(db *gorm.DB, c *gin.Context) (models.TaskerProfile, bool) {
	profile, ok := taskerProfile(db, middleware.CurrentUserID(c))
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "tasker profile required"})
		return profile, false
	}
	return profile, true
}

func parseUintParam(c *gin.Context, name string) (uint, bool) {
	value, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil || value == 0 {
		return 0, false
	}
	return uint(value), true
}
