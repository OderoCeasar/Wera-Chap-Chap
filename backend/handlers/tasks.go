package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"wera-chap-chap/backend/middleware"
	"wera-chap-chap/backend/models"
	"wera-chap-chap/backend/services"
)

type TaskHandler struct{ db *gorm.DB }

func NewTaskHandler(db *gorm.DB) *TaskHandler { return &TaskHandler{db: db} }

// taskInput is the client-writable surface of a task. Binding the model
// directly would let a caller set client_id, status or nested records.
type taskInput struct {
	CategoryID      uint              `json:"category_id" binding:"required"`
	Title           string            `json:"title" binding:"required"`
	Description     string            `json:"description"`
	LocationAddress string            `json:"location_address"`
	LocationLat     float64           `json:"location_lat"`
	LocationLng     float64           `json:"location_lng"`
	BudgetType      models.BudgetType `json:"budget_type" binding:"required"`
	BudgetAmount    float64           `json:"budget_amount" binding:"gte=0"`
	ScheduledAt     time.Time         `json:"scheduled_at"`
}

func (i taskInput) validate() error {
	if i.BudgetType != models.BudgetFixed && i.BudgetType != models.BudgetHourly {
		return errors.New("budget_type must be fixed or hourly")
	}
	return nil
}

func (h *TaskHandler) Create(c *gin.Context) {
	var input taskInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := input.validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var category models.Category
	if err := h.db.First(&category, input.CategoryID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown category"})
		return
	}

	task := models.Task{
		ClientID:        middleware.CurrentUserID(c),
		CategoryID:      input.CategoryID,
		Title:           input.Title,
		Description:     input.Description,
		LocationAddress: input.LocationAddress,
		LocationLat:     input.LocationLat,
		LocationLng:     input.LocationLng,
		BudgetType:      input.BudgetType,
		BudgetAmount:    input.BudgetAmount,
		ScheduledAt:     input.ScheduledAt,
		Status:          models.TaskOpen,
	}
	if err := h.db.Create(&task).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "could not create task"})
		return
	}
	h.db.Preload("Category").Preload("Client").First(&task, task.ID)
	c.JSON(http.StatusCreated, task)
}

// List is the open marketplace feed taskers browse.
func (h *TaskHandler) List(c *gin.Context) {
	query := h.db.Preload("Client").Preload("Category").Where("status = ?", models.TaskOpen)
	if categoryID := c.Query("category_id"); categoryID != "" {
		query = query.Where("category_id = ?", categoryID)
	}
	if minBudget := c.Query("min_budget"); minBudget != "" {
		query = query.Where("budget_amount >= ?", minBudget)
	}
	if search := c.Query("q"); search != "" {
		query = query.Where("title ILIKE ? OR description ILIKE ?", "%"+search+"%", "%"+search+"%")
	}
	tasks := []models.Task{}
	query.Order("created_at desc").Find(&tasks)
	c.JSON(http.StatusOK, tasks)
}

func (h *TaskHandler) MyTasks(c *gin.Context) {
	tasks := []models.Task{}
	h.db.Preload("Category").Preload("Applications.Tasker.User").
		Where("client_id = ?", middleware.CurrentUserID(c)).
		Order("created_at desc").Find(&tasks)
	c.JSON(http.StatusOK, tasks)
}

func (h *TaskHandler) Get(c *gin.Context) {
	var task models.Task
	if err := h.db.Preload("Client").Preload("Category").Preload("Applications.Tasker.User").
		First(&task, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}
	c.JSON(http.StatusOK, task)
}

func (h *TaskHandler) Update(c *gin.Context) {
	var task models.Task
	if !h.ownedOpenTask(c, &task) {
		return
	}
	var input taskInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := input.validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.db.Model(&task).Updates(map[string]interface{}{
		"category_id": input.CategoryID, "title": input.Title, "description": input.Description,
		"location_address": input.LocationAddress, "location_lat": input.LocationLat,
		"location_lng": input.LocationLng, "budget_type": input.BudgetType,
		"budget_amount": input.BudgetAmount, "scheduled_at": input.ScheduledAt,
	})
	h.Get(c)
}

func (h *TaskHandler) Cancel(c *gin.Context) {
	var task models.Task
	if !h.ownedOpenTask(c, &task) {
		return
	}
	h.db.Model(&task).Update("status", models.TaskCancelled)
	c.JSON(http.StatusOK, gin.H{"message": "task cancelled"})
}

func (h *TaskHandler) Apply(c *gin.Context) {
	profile, ok := currentTaskerProfile(h.db, c)
	if !ok {
		return
	}
	taskID, valid := parseUintParam(c, "id")
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}
	var task models.Task
	if err := h.db.First(&task, taskID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}
	if task.Status != models.TaskOpen {
		c.JSON(http.StatusConflict, gin.H{"error": "this task is no longer open"})
		return
	}

	var input struct {
		ProposedRate float64 `json:"proposed_rate" binding:"required,gt=0"`
		CoverNote    string  `json:"cover_note"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var existing int64
	h.db.Model(&models.TaskApplication{}).Where("task_id = ? AND tasker_id = ?", task.ID, profile.ID).Count(&existing)
	if existing > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "you already applied for this task"})
		return
	}

	app := models.TaskApplication{
		TaskID: task.ID, TaskerID: profile.ID, ProposedRate: input.ProposedRate,
		CoverNote: input.CoverNote, Status: models.ApplicationPending,
	}
	if err := h.db.Create(&app).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "could not apply"})
		return
	}
	h.db.Preload("Tasker.User").First(&app, app.ID)
	c.JSON(http.StatusCreated, app)
}

func (h *TaskHandler) AcceptApplication(c *gin.Context) {
	h.decideApplication(c, models.ApplicationAccepted)
}

func (h *TaskHandler) RejectApplication(c *gin.Context) {
	h.decideApplication(c, models.ApplicationRejected)
}

func (h *TaskHandler) Matches(c *gin.Context) {
	var task models.Task
	if err := h.db.First(&task, c.Param("id")).Error; err != nil || task.ClientID != middleware.CurrentUserID(c) {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}
	matches, err := services.RankedTaskers(h.db, task, 10)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not rank taskers"})
		return
	}
	c.JSON(http.StatusOK, matches)
}

// decideApplication accepts or rejects an application. Accepting is the moment
// a task becomes a booking, so it rejects the rival applications atomically.
func (h *TaskHandler) decideApplication(c *gin.Context, status models.ApplicationStatus) {
	var task models.Task
	if err := h.db.First(&task, c.Param("id")).Error; err != nil || task.ClientID != middleware.CurrentUserID(c) {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}
	var app models.TaskApplication
	if err := h.db.First(&app, "id = ? AND task_id = ?", c.Param("app_id"), task.ID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "application not found"})
		return
	}
	if app.Status != models.ApplicationPending {
		c.JSON(http.StatusConflict, gin.H{"error": "application already " + string(app.Status)})
		return
	}

	if status == models.ApplicationRejected {
		h.db.Model(&app).Update("status", status)
		c.JSON(http.StatusOK, gin.H{"status": status})
		return
	}

	if task.Status != models.TaskOpen {
		c.JSON(http.StatusConflict, gin.H{"error": "this task already has a tasker"})
		return
	}

	var booking models.Booking
	err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&app).Update("status", models.ApplicationAccepted).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.TaskApplication{}).
			Where("task_id = ? AND id <> ?", task.ID, app.ID).
			Update("status", models.ApplicationRejected).Error; err != nil {
			return err
		}
		if err := tx.Model(&task).Update("status", models.TaskMatched).Error; err != nil {
			return err
		}
		booking = models.Booking{
			TaskID: task.ID, ClientID: task.ClientID, TaskerID: app.TaskerID,
			AgreedRate: app.ProposedRate, Status: models.BookingConfirmed,
		}
		return tx.Create(&booking).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not accept application"})
		return
	}

	h.db.Preload("Task.Category").Preload("Client").Preload("Tasker.User").First(&booking, booking.ID)
	c.JSON(http.StatusOK, gin.H{"status": status, "booking": booking})
}

func (h *TaskHandler) ownedOpenTask(c *gin.Context, task *models.Task) bool {
	if err := h.db.First(task, c.Param("id")).Error; err != nil || task.ClientID != middleware.CurrentUserID(c) {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return false
	}
	if task.Status != models.TaskOpen {
		c.JSON(http.StatusConflict, gin.H{"error": "only open tasks can be changed"})
		return false
	}
	return true
}
