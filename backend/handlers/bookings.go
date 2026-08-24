package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"wera-chap-chap/backend/middleware"
	"wera-chap-chap/backend/models"
)

type BookingHandler struct{ db *gorm.DB }

func NewBookingHandler(db *gorm.DB) *BookingHandler { return &BookingHandler{db: db} }

// List returns every booking the caller takes part in, as client or as tasker.
func (h *BookingHandler) List(c *gin.Context) {
	userID := middleware.CurrentUserID(c)
	query := h.db.Preload("Task.Category").Preload("Client").Preload("Tasker.User")

	if profile, ok := taskerProfile(h.db, userID); ok {
		query = query.Where("client_id = ? OR tasker_id = ?", userID, profile.ID)
	} else {
		query = query.Where("client_id = ?", userID)
	}

	bookings := []models.Booking{}
	query.Order("created_at desc").Find(&bookings)
	c.JSON(http.StatusOK, bookings)
}

func (h *BookingHandler) Get(c *gin.Context) {
	scope, ok := loadBooking(h.db, c, "id", true)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, scope.Booking)
}

func (h *BookingHandler) Start(c *gin.Context) {
	h.transition(c, models.BookingStarted)
}

func (h *BookingHandler) Complete(c *gin.Context) {
	h.transition(c, models.BookingCompleted)
}

func (h *BookingHandler) Cancel(c *gin.Context) {
	h.transition(c, models.BookingCancelled)
}

// allowedTransitions is the booking lifecycle: a booking is confirmed on
// creation, started by the tasker, then completed. Either party may cancel
// before the work is finished.
var allowedTransitions = map[models.BookingStatus][]models.BookingStatus{
	models.BookingConfirmed: {models.BookingStarted, models.BookingCancelled},
	models.BookingStarted:   {models.BookingCompleted, models.BookingCancelled},
	models.BookingCompleted: {},
	models.BookingCancelled: {},
}

func canTransition(from, to models.BookingStatus) bool {
	for _, candidate := range allowedTransitions[from] {
		if candidate == to {
			return true
		}
	}
	return false
}

func (h *BookingHandler) transition(c *gin.Context, status models.BookingStatus) {
	scope, ok := loadBooking(h.db, c, "id", false)
	if !ok {
		return
	}
	booking := scope.Booking

	// Only the assigned tasker drives work forward; cancelling is open to both.
	if status != models.BookingCancelled && !scope.IsTasker {
		c.JSON(http.StatusForbidden, gin.H{"error": "only the assigned tasker can do that"})
		return
	}
	if !canTransition(booking.Status, status) {
		c.JSON(http.StatusConflict, gin.H{
			"error": "cannot move booking from " + string(booking.Status) + " to " + string(status),
		})
		return
	}

	now := time.Now()
	updates := map[string]interface{}{"status": status}
	taskStatus := models.TaskInProgress
	switch status {
	case models.BookingStarted:
		updates["started_at"] = &now
	case models.BookingCompleted:
		updates["completed_at"] = &now
		taskStatus = models.TaskCompleted
	case models.BookingCancelled:
		taskStatus = models.TaskCancelled
	}

	err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Booking{}).Where("id = ?", booking.ID).Updates(updates).Error; err != nil {
			return err
		}
		return tx.Model(&models.Task{}).Where("id = ?", booking.TaskID).Update("status", taskStatus).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not update booking"})
		return
	}

	var updated models.Booking
	h.db.Preload("Task.Category").Preload("Client").Preload("Tasker.User").First(&updated, booking.ID)
	c.JSON(http.StatusOK, updated)
}
