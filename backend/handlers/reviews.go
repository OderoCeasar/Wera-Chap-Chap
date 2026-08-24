package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"wera-chap-chap/backend/middleware"
	"wera-chap-chap/backend/models"
)

type ReviewHandler struct{ db *gorm.DB }

func NewReviewHandler(db *gorm.DB) *ReviewHandler { return &ReviewHandler{db: db} }

func (h *ReviewHandler) Create(c *gin.Context) {
	scope, ok := loadBooking(h.db, c, "booking_id", false)
	if !ok {
		return
	}
	booking := scope.Booking
	if booking.Status != models.BookingCompleted {
		c.JSON(http.StatusConflict, gin.H{"error": "you can only review a completed booking"})
		return
	}

	var input struct {
		Rating  int    `json:"rating" binding:"required,min=1,max=5"`
		Comment string `json:"comment"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := middleware.CurrentUserID(c)

	// Each side reviews the other: the client rates the tasker's user account,
	// the tasker rates the client.
	var revieweeID uint
	if scope.IsClient {
		var tasker models.TaskerProfile
		if err := h.db.First(&tasker, booking.TaskerID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "tasker not found"})
			return
		}
		revieweeID = tasker.UserID
	} else {
		revieweeID = booking.ClientID
	}

	var existing int64
	h.db.Model(&models.Review{}).Where("booking_id = ? AND reviewer_id = ?", booking.ID, userID).Count(&existing)
	if existing > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "you already reviewed this booking"})
		return
	}

	review := models.Review{
		BookingID: booking.ID, ReviewerID: userID, RevieweeID: revieweeID,
		Rating: input.Rating, Comment: input.Comment,
	}
	if err := h.db.Create(&review).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "could not save review"})
		return
	}
	h.recalculateTaskerRating(revieweeID)
	h.db.Preload("Reviewer").First(&review, review.ID)
	c.JSON(http.StatusCreated, review)
}

// ForBooking returns the reviews attached to a booking so the UI can tell
// whether the caller has already left one.
func (h *ReviewHandler) ForBooking(c *gin.Context) {
	scope, ok := loadBooking(h.db, c, "booking_id", false)
	if !ok {
		return
	}
	reviews := []models.Review{}
	h.db.Preload("Reviewer").Where("booking_id = ?", scope.Booking.ID).Order("created_at desc").Find(&reviews)
	c.JSON(http.StatusOK, reviews)
}

func (h *ReviewHandler) ForTasker(c *gin.Context) {
	var tasker models.TaskerProfile
	if err := h.db.First(&tasker, c.Param("tasker_id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tasker not found"})
		return
	}
	reviews := []models.Review{}
	h.db.Preload("Reviewer").Where("reviewee_id = ?", tasker.UserID).Order("created_at desc").Find(&reviews)
	c.JSON(http.StatusOK, reviews)
}

// recalculateTaskerRating refreshes the denormalised rating columns. It is a
// no-op when the reviewee is a client, who has no tasker profile.
func (h *ReviewHandler) recalculateTaskerRating(userID uint) {
	var stats struct {
		Average float64
		Count   int
	}
	h.db.Model(&models.Review{}).
		Select("COALESCE(AVG(rating), 0) as average, COUNT(*) as count").
		Where("reviewee_id = ?", userID).Scan(&stats)
	h.db.Model(&models.TaskerProfile{}).Where("user_id = ?", userID).
		Updates(map[string]interface{}{"avg_rating": stats.Average, "total_reviews": stats.Count})
}
