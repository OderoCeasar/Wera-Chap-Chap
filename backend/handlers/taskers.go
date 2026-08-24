package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"wera-chap-chap/backend/middleware"
	"wera-chap-chap/backend/models"
)

type TaskerHandler struct{ db *gorm.DB }

func NewTaskerHandler(db *gorm.DB) *TaskerHandler { return &TaskerHandler{db: db} }

func (h *TaskerHandler) List(c *gin.Context) {
	query := h.db.Preload("User").Preload("Skills.Category").Where("is_available = ?", true)
	if category := c.Query("category_id"); category != "" {
		query = query.Joins("JOIN tasker_skills ON tasker_skills.tasker_id = tasker_profiles.id").
			Where("tasker_skills.category_id = ?", category)
	}
	if maxRate := c.Query("max_rate"); maxRate != "" {
		query = query.Where("hourly_rate <= ?", maxRate)
	}
	if minRating := c.Query("min_rating"); minRating != "" {
		query = query.Where("avg_rating >= ?", minRating)
	}
	if search := c.Query("q"); search != "" {
		query = query.Joins("JOIN users ON users.id = tasker_profiles.user_id").
			Where("users.full_name ILIKE ? OR tasker_profiles.bio ILIKE ?", "%"+search+"%", "%"+search+"%")
	}
	taskers := []models.TaskerProfile{}
	query.Order("avg_rating desc, total_reviews desc").Find(&taskers)
	c.JSON(http.StatusOK, taskers)
}

func (h *TaskerHandler) Get(c *gin.Context) {
	var tasker models.TaskerProfile
	if err := h.db.Preload("User").Preload("Skills.Category").Preload("Availability").
		First(&tasker, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tasker not found"})
		return
	}
	c.JSON(http.StatusOK, tasker)
}

// Me returns the calling tasker's own profile so the dashboard can prefill.
func (h *TaskerHandler) Me(c *gin.Context) {
	h.respondWithProfile(c, middleware.CurrentUserID(c))
}

func (h *TaskerHandler) UpdateProfile(c *gin.Context) {
	// Pointers let us tell "not supplied" apart from a deliberate zero, which a
	// struct-based Updates would silently discard.
	var input struct {
		Bio             *string  `json:"bio"`
		HourlyRate      *float64 `json:"hourly_rate"`
		YearsExperience *int     `json:"years_experience"`
		ServiceRadiusKM *float64 `json:"service_radius_km"`
		IsAvailable     *bool    `json:"is_available"`
		SkillIDs        []uint   `json:"skill_ids"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	profile, ok := currentTaskerProfile(h.db, c)
	if !ok {
		return
	}

	updates := map[string]interface{}{}
	if input.Bio != nil {
		updates["bio"] = *input.Bio
	}
	if input.HourlyRate != nil {
		if *input.HourlyRate < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "hourly rate cannot be negative"})
			return
		}
		updates["hourly_rate"] = *input.HourlyRate
	}
	if input.YearsExperience != nil {
		updates["years_experience"] = *input.YearsExperience
	}
	if input.ServiceRadiusKM != nil {
		updates["service_radius_km"] = *input.ServiceRadiusKM
	}
	if input.IsAvailable != nil {
		updates["is_available"] = *input.IsAvailable
	}

	err := h.db.Transaction(func(tx *gorm.DB) error {
		if len(updates) > 0 {
			if err := tx.Model(&models.TaskerProfile{}).Where("id = ?", profile.ID).Updates(updates).Error; err != nil {
				return err
			}
		}
		if input.SkillIDs == nil {
			return nil
		}
		if err := tx.Where("tasker_id = ?", profile.ID).Delete(&models.TaskerSkill{}).Error; err != nil {
			return err
		}
		seen := map[uint]bool{}
		for _, categoryID := range input.SkillIDs {
			if seen[categoryID] {
				continue
			}
			seen[categoryID] = true
			if err := tx.Create(&models.TaskerSkill{TaskerID: profile.ID, CategoryID: categoryID}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "could not update profile"})
		return
	}

	h.respondWithProfile(c, middleware.CurrentUserID(c))
}

func (h *TaskerHandler) SetAvailability(c *gin.Context) {
	var input []models.TaskerAvailability
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	for _, slot := range input {
		if slot.DayOfWeek < 0 || slot.DayOfWeek > 6 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "day_of_week must be between 0 and 6"})
			return
		}
		if slot.StartTime >= slot.EndTime {
			c.JSON(http.StatusBadRequest, gin.H{"error": "start_time must be before end_time"})
			return
		}
	}
	profile, ok := currentTaskerProfile(h.db, c)
	if !ok {
		return
	}

	err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("tasker_id = ?", profile.ID).Delete(&models.TaskerAvailability{}).Error; err != nil {
			return err
		}
		for _, slot := range input {
			slot.ID = 0
			slot.TaskerID = profile.ID
			if err := tx.Create(&slot).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "could not save availability"})
		return
	}

	h.respondWithProfile(c, middleware.CurrentUserID(c))
}

func (h *TaskerHandler) MyBookings(c *gin.Context) {
	profile, ok := currentTaskerProfile(h.db, c)
	if !ok {
		return
	}
	bookings := []models.Booking{}
	h.db.Preload("Task.Category").Preload("Client").Preload("Tasker.User").
		Where("tasker_id = ?", profile.ID).Order("created_at desc").Find(&bookings)
	c.JSON(http.StatusOK, bookings)
}

// MyApplications lets a tasker track what they have applied for.
func (h *TaskerHandler) MyApplications(c *gin.Context) {
	profile, ok := currentTaskerProfile(h.db, c)
	if !ok {
		return
	}
	applications := []models.TaskApplication{}
	h.db.Preload("Task.Category").Preload("Task.Client").
		Where("tasker_id = ?", profile.ID).Order("created_at desc").Find(&applications)
	c.JSON(http.StatusOK, applications)
}

func (h *TaskerHandler) respondWithProfile(c *gin.Context, userID uint) {
	var profile models.TaskerProfile
	if err := h.db.Preload("User").Preload("Skills.Category").Preload("Availability").
		Where("user_id = ?", userID).First(&profile).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tasker profile not found"})
		return
	}
	c.JSON(http.StatusOK, profile)
}
