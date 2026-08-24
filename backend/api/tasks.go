package api

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	db "wera-chap-chap/backend/db/sqlc"
)

var (
	errTaskNotFound        = errors.New("task not found")
	errUnknownCategory     = errors.New("unknown category")
	errTaskNotOpen         = errors.New("this task is no longer open")
	errOnlyOpenTasksChange = errors.New("only open tasks can be changed")
	errAlreadyApplied      = errors.New("you already applied for this task")
	errApplicationNotFound = errors.New("application not found")
	errTaskAlreadyMatched  = errors.New("this task already has a tasker")
	errBadBudgetType       = errors.New("budget_type must be fixed or hourly")
)

// maxMatches is how many ranked taskers the matching endpoint returns. The
// client shows a shortlist, and scoring is done in Go over every available
// tasker, so the cap keeps the response useful rather than exhaustive.
const maxMatches = 10

// taskRequest is the client-writable surface of a task. Binding the row type
// directly would let a caller set client_id or status.
type taskRequest struct {
	CategoryID      int64      `json:"category_id" binding:"required"`
	Title           string     `json:"title" binding:"required"`
	Description     string     `json:"description"`
	LocationAddress string     `json:"location_address"`
	LocationLat     float64    `json:"location_lat"`
	LocationLng     float64    `json:"location_lng"`
	BudgetType      string     `json:"budget_type" binding:"required"`
	BudgetAmount    float64    `json:"budget_amount" binding:"gte=0"`
	ScheduledAt     *time.Time `json:"scheduled_at"`
}

func (req taskRequest) validate() error {
	if !db.IsValidBudgetType(req.BudgetType) {
		return errBadBudgetType
	}
	return nil
}

func (server *Server) createTask(ctx *gin.Context) {
	var req taskRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	if err := req.validate(); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	task, err := server.store.CreateTask(ctx, db.CreateTaskParams{
		ClientID:        currentUserID(ctx),
		CategoryID:      req.CategoryID,
		Title:           req.Title,
		Description:     req.Description,
		LocationAddress: req.LocationAddress,
		LocationLat:     req.LocationLat,
		LocationLng:     req.LocationLng,
		BudgetType:      req.BudgetType,
		BudgetAmount:    req.BudgetAmount,
		ScheduledAt:     req.ScheduledAt,
	})
	if err != nil {
		// The category is a foreign key, so an unknown one is caught by the
		// database rather than by a lookup query before the insert.
		if db.ErrorCode(err) == db.ForeignKeyViolation {
			ctx.JSON(http.StatusBadRequest, errorResponse(errUnknownCategory))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	server.respondWithTask(ctx, task.ID, http.StatusCreated, false)
}

// listTasks is the open marketplace feed taskers browse.
func (server *Server) listTasks(ctx *gin.Context) {
	var params db.ListOpenTasksParams

	if raw := ctx.Query("category_id"); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, errorResponse(errBadFilterFormat))
			return
		}
		params.CategoryID = &value
	}
	if raw := ctx.Query("min_budget"); raw != "" {
		value, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, errorResponse(errBadFilterFormat))
			return
		}
		params.MinBudget = &value
	}
	if raw := ctx.Query("q"); raw != "" {
		params.Search = &raw
	}

	rows, err := server.store.ListOpenTasks(ctx, params)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	tasks := make([]taskResponse, 0, len(rows))
	for _, row := range rows {
		tasks = append(tasks, newTaskWithRelations(row.Task, row.Category, row.User))
	}
	ctx.JSON(http.StatusOK, tasks)
}

func (server *Server) listMyTasks(ctx *gin.Context) {
	rows, err := server.store.ListTasksByClient(ctx, currentUserID(ctx))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	tasks := make([]taskResponse, 0, len(rows))
	refs := make([]*taskResponse, 0, len(rows))
	for _, row := range rows {
		tasks = append(tasks, newTaskWithRelations(row.Task, row.Category, row.User))
	}
	for index := range tasks {
		refs = append(refs, &tasks[index])
	}

	// The client dashboard decides on applications, so they are nested here.
	if err := server.attachApplications(ctx, refs); err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, tasks)
}

func (server *Server) getTask(ctx *gin.Context) {
	id, ok := parseInt64Param(ctx, "id")
	if !ok {
		ctx.JSON(http.StatusBadRequest, errorResponse(errInvalidID))
		return
	}
	server.respondWithTask(ctx, id, http.StatusOK, true)
}

func (server *Server) updateTask(ctx *gin.Context) {
	task, ok := server.ownedOpenTask(ctx)
	if !ok {
		return
	}

	var req taskRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	if err := req.validate(); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	if _, err := server.store.UpdateTask(ctx, db.UpdateTaskParams{
		ID:              task.ID,
		CategoryID:      req.CategoryID,
		Title:           req.Title,
		Description:     req.Description,
		LocationAddress: req.LocationAddress,
		LocationLat:     req.LocationLat,
		LocationLng:     req.LocationLng,
		BudgetType:      req.BudgetType,
		BudgetAmount:    req.BudgetAmount,
		ScheduledAt:     req.ScheduledAt,
	}); err != nil {
		if db.ErrorCode(err) == db.ForeignKeyViolation {
			ctx.JSON(http.StatusBadRequest, errorResponse(errUnknownCategory))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	server.respondWithTask(ctx, task.ID, http.StatusOK, true)
}

func (server *Server) cancelTask(ctx *gin.Context) {
	task, ok := server.ownedOpenTask(ctx)
	if !ok {
		return
	}

	if err := server.store.UpdateTaskStatus(ctx, db.UpdateTaskStatusParams{
		ID:     task.ID,
		Status: db.TaskCancelled,
	}); err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "task cancelled"})
}

type applyRequest struct {
	ProposedRate float64 `json:"proposed_rate" binding:"required,gt=0"`
	CoverNote    string  `json:"cover_note"`
}

func (server *Server) applyToTask(ctx *gin.Context) {
	profile, ok := server.currentTaskerProfile(ctx)
	if !ok {
		return
	}

	taskID, ok := parseInt64Param(ctx, "id")
	if !ok {
		ctx.JSON(http.StatusBadRequest, errorResponse(errInvalidID))
		return
	}

	task, err := server.store.GetTask(ctx, taskID)
	if err != nil {
		if db.IsNotFound(err) {
			ctx.JSON(http.StatusNotFound, errorResponse(errTaskNotFound))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	if task.Status != db.TaskOpen {
		ctx.JSON(http.StatusConflict, errorResponse(errTaskNotOpen))
		return
	}

	var req applyRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	application, err := server.store.CreateTaskApplication(ctx, db.CreateTaskApplicationParams{
		TaskID:       task.ID,
		TaskerID:     profile.ID,
		ProposedRate: req.ProposedRate,
		CoverNote:    req.CoverNote,
	})
	if err != nil {
		// (task_id, tasker_id) is UNIQUE, so the duplicate is caught by the
		// insert rather than by a count-then-insert, which could let two
		// concurrent requests both pass the check.
		if db.IsUniqueViolation(err) {
			ctx.JSON(http.StatusConflict, errorResponse(errAlreadyApplied))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	row, err := server.store.GetApplicationWithTasker(ctx, application.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	response := newApplicationResponse(row.TaskApplication)
	response.Tasker = newTaskerProfileRef(row.TaskerProfile, row.User)
	ctx.JSON(http.StatusCreated, response)
}

func (server *Server) acceptApplication(ctx *gin.Context) {
	server.decideApplication(ctx, db.ApplicationAccepted)
}

func (server *Server) rejectApplication(ctx *gin.Context) {
	server.decideApplication(ctx, db.ApplicationRejected)
}

func (server *Server) listTaskMatches(ctx *gin.Context) {
	id, ok := parseInt64Param(ctx, "id")
	if !ok {
		ctx.JSON(http.StatusBadRequest, errorResponse(errInvalidID))
		return
	}

	task, err := server.store.GetTask(ctx, id)
	if err != nil || task.ClientID != currentUserID(ctx) {
		// Answering 404 rather than 403 for someone else's task keeps this
		// endpoint from confirming that a given task id exists.
		ctx.JSON(http.StatusNotFound, errorResponse(errTaskNotFound))
		return
	}

	results, err := server.matcher.RankedTaskers(ctx, task, maxMatches)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, newMatchResponses(results))
}

// decideApplication accepts or rejects an application. Accepting is the moment
// a task becomes a booking, which is why it goes through AcceptApplicationTx.
func (server *Server) decideApplication(ctx *gin.Context, status string) {
	taskID, ok := parseInt64Param(ctx, "id")
	if !ok {
		ctx.JSON(http.StatusBadRequest, errorResponse(errInvalidID))
		return
	}
	applicationID, ok := parseInt64Param(ctx, "app_id")
	if !ok {
		ctx.JSON(http.StatusBadRequest, errorResponse(errInvalidID))
		return
	}

	task, err := server.store.GetTask(ctx, taskID)
	if err != nil || task.ClientID != currentUserID(ctx) {
		ctx.JSON(http.StatusNotFound, errorResponse(errTaskNotFound))
		return
	}

	application, err := server.store.GetTaskApplication(ctx, db.GetTaskApplicationParams{
		ID:     applicationID,
		TaskID: task.ID,
	})
	if err != nil {
		if db.IsNotFound(err) {
			ctx.JSON(http.StatusNotFound, errorResponse(errApplicationNotFound))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	if application.Status != db.ApplicationPending {
		ctx.JSON(http.StatusConflict, errorResponse(
			errors.New("application already "+application.Status)))
		return
	}

	if status == db.ApplicationRejected {
		if err := server.store.UpdateApplicationStatus(ctx, db.UpdateApplicationStatusParams{
			ID:     application.ID,
			Status: db.ApplicationRejected,
		}); err != nil {
			ctx.JSON(http.StatusInternalServerError, errorResponse(err))
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"status": status})
		return
	}

	if task.Status != db.TaskOpen {
		ctx.JSON(http.StatusConflict, errorResponse(errTaskAlreadyMatched))
		return
	}

	result, err := server.store.AcceptApplicationTx(ctx, db.AcceptApplicationTxParams{
		Task:        task,
		Application: application,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	row, err := server.store.GetBookingWithRelations(ctx, result.Booking.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status": status,
		"booking": newBookingWithRelations(
			row.Booking, row.Task, row.Category, row.TaskerProfile, row.User, row.User_2),
	})
}

// ownedOpenTask loads the task named by :id and confirms the caller owns it and
// that it is still open.
func (server *Server) ownedOpenTask(ctx *gin.Context) (db.Task, bool) {
	id, ok := parseInt64Param(ctx, "id")
	if !ok {
		ctx.JSON(http.StatusBadRequest, errorResponse(errInvalidID))
		return db.Task{}, false
	}

	task, err := server.store.GetTask(ctx, id)
	if err != nil || task.ClientID != currentUserID(ctx) {
		ctx.JSON(http.StatusNotFound, errorResponse(errTaskNotFound))
		return db.Task{}, false
	}
	if task.Status != db.TaskOpen {
		ctx.JSON(http.StatusConflict, errorResponse(errOnlyOpenTasksChange))
		return db.Task{}, false
	}
	return task, true
}

// respondWithTask renders one task, optionally with its applications nested.
func (server *Server) respondWithTask(ctx *gin.Context, id int64, status int, withApplications bool) {
	row, err := server.store.GetTaskWithRelations(ctx, id)
	if err != nil {
		if db.IsNotFound(err) {
			ctx.JSON(http.StatusNotFound, errorResponse(errTaskNotFound))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	response := newTaskWithRelations(row.Task, row.Category, row.User)
	if withApplications {
		if err := server.attachApplications(ctx, []*taskResponse{&response}); err != nil {
			ctx.JSON(http.StatusInternalServerError, errorResponse(err))
			return
		}
	}

	ctx.JSON(status, response)
}
