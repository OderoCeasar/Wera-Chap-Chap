package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	db "wera-chap-chap/backend/db/sqlc"
)

var (
	errTaskerNotFound  = errors.New("tasker not found")
	errNegativeRate    = errors.New("hourly rate cannot be negative")
	errBadDayOfWeek    = errors.New("day_of_week must be between 0 and 6")
	errBadTimeRange    = errors.New("start_time must be before end_time")
	errBadTimeFormat   = errors.New("start_time and end_time must be HH:MM")
	errNegativeRadius  = errors.New("service radius cannot be negative")
	errNegativeYears   = errors.New("years of experience cannot be negative")
	errBadFilterFormat = errors.New("invalid filter value")
)

func (server *Server) listTaskers(ctx *gin.Context) {
	// The directory only ever shows available taskers; the flag is a query
	// argument rather than a hardcoded predicate so the matching service can
	// reuse the same statement.
	availableOnly := true
	params := db.ListTaskerProfilesParams{AvailableOnly: &availableOnly}

	if raw := ctx.Query("category_id"); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, errorResponse(errBadFilterFormat))
			return
		}
		params.CategoryID = &value
	}
	if raw := ctx.Query("max_rate"); raw != "" {
		value, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, errorResponse(errBadFilterFormat))
			return
		}
		params.MaxRate = &value
	}
	if raw := ctx.Query("min_rating"); raw != "" {
		value, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, errorResponse(errBadFilterFormat))
			return
		}
		params.MinRating = &value
	}
	if raw := ctx.Query("q"); raw != "" {
		params.Search = &raw
	}

	rows, err := server.store.ListTaskerProfiles(ctx, params)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	taskers := make([]*taskerProfileResponse, 0, len(rows))
	for _, row := range rows {
		taskers = append(taskers, newTaskerProfileRef(row.TaskerProfile, row.User))
	}
	if err := server.attachTaskerRelations(ctx, taskers); err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, taskers)
}

func (server *Server) getTasker(ctx *gin.Context) {
	id, ok := parseInt64Param(ctx, "id")
	if !ok {
		ctx.JSON(http.StatusBadRequest, errorResponse(errInvalidID))
		return
	}

	row, err := server.store.GetTaskerProfileWithUser(ctx, id)
	if err != nil {
		if db.IsNotFound(err) {
			ctx.JSON(http.StatusNotFound, errorResponse(errTaskerNotFound))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	server.respondWithTaskerProfile(ctx, row.TaskerProfile, row.User, http.StatusOK)
}

// getMyTaskerProfile returns the calling tasker's own profile so the dashboard
// can prefill its forms.
func (server *Server) getMyTaskerProfile(ctx *gin.Context) {
	row, err := server.store.GetTaskerProfileWithUserByUserID(ctx, currentUserID(ctx))
	if err != nil {
		if db.IsNotFound(err) {
			ctx.JSON(http.StatusNotFound, errorResponse(errTaskerNotFound))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	server.respondWithTaskerProfile(ctx, row.TaskerProfile, row.User, http.StatusOK)
}

type updateTaskerProfileRequest struct {
	// Pointers let us tell "not supplied" apart from a deliberate zero, which a
	// value-typed request would silently discard.
	Bio             *string  `json:"bio"`
	HourlyRate      *float64 `json:"hourly_rate"`
	YearsExperience *int32   `json:"years_experience"`
	ServiceRadiusKM *float64 `json:"service_radius_km"`
	IsAvailable     *bool    `json:"is_available"`
	// SkillIDs nil leaves the skill set alone; a supplied list replaces it.
	SkillIDs []int64 `json:"skill_ids"`
}

func (server *Server) updateTaskerProfile(ctx *gin.Context) {
	var req updateTaskerProfileRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	if req.HourlyRate != nil && *req.HourlyRate < 0 {
		ctx.JSON(http.StatusBadRequest, errorResponse(errNegativeRate))
		return
	}
	if req.ServiceRadiusKM != nil && *req.ServiceRadiusKM < 0 {
		ctx.JSON(http.StatusBadRequest, errorResponse(errNegativeRadius))
		return
	}
	if req.YearsExperience != nil && *req.YearsExperience < 0 {
		ctx.JSON(http.StatusBadRequest, errorResponse(errNegativeYears))
		return
	}

	profile, ok := server.currentTaskerProfile(ctx)
	if !ok {
		return
	}

	if _, err := server.store.UpdateTaskerProfileTx(ctx, db.UpdateTaskerProfileTxParams{
		UpdateTaskerProfileParams: db.UpdateTaskerProfileParams{
			ID:              profile.ID,
			Bio:             req.Bio,
			HourlyRate:      req.HourlyRate,
			YearsExperience: req.YearsExperience,
			ServiceRadiusKm: req.ServiceRadiusKM,
			IsAvailable:     req.IsAvailable,
		},
		SkillIDs: req.SkillIDs,
	}); err != nil {
		if db.ErrorCode(err) == db.ForeignKeyViolation {
			ctx.JSON(http.StatusBadRequest, errorResponse(errors.New("unknown category in skill_ids")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	server.respondWithCurrentTaskerProfile(ctx)
}

type availabilitySlotRequest struct {
	DayOfWeek int32  `json:"day_of_week"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

func (server *Server) setAvailability(ctx *gin.Context) {
	var req []availabilitySlotRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	slots := make([]db.AvailabilitySlot, 0, len(req))
	for _, slot := range req {
		if slot.DayOfWeek < 0 || slot.DayOfWeek > 6 {
			ctx.JSON(http.StatusBadRequest, errorResponse(errBadDayOfWeek))
			return
		}
		// The column is TEXT with a CHECK on the format; validating here turns
		// a client mistake into a 400 rather than a constraint violation.
		if !isHourMinute(slot.StartTime) || !isHourMinute(slot.EndTime) {
			ctx.JSON(http.StatusBadRequest, errorResponse(errBadTimeFormat))
			return
		}
		if slot.StartTime >= slot.EndTime {
			ctx.JSON(http.StatusBadRequest, errorResponse(errBadTimeRange))
			return
		}
		slots = append(slots, db.AvailabilitySlot{
			DayOfWeek: slot.DayOfWeek,
			StartTime: slot.StartTime,
			EndTime:   slot.EndTime,
		})
	}

	profile, ok := server.currentTaskerProfile(ctx)
	if !ok {
		return
	}

	if _, err := server.store.SetAvailabilityTx(ctx, db.SetAvailabilityTxParams{
		TaskerID: profile.ID,
		Slots:    slots,
	}); err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	server.respondWithCurrentTaskerProfile(ctx)
}

func (server *Server) listMyTaskerBookings(ctx *gin.Context) {
	profile, ok := server.currentTaskerProfile(ctx)
	if !ok {
		return
	}

	rows, err := server.store.ListBookingsByTasker(ctx, profile.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	bookings := make([]bookingResponse, 0, len(rows))
	for _, row := range rows {
		bookings = append(bookings, newBookingWithRelations(
			row.Booking, row.Task, row.Category, row.TaskerProfile, row.User, row.User_2))
	}
	ctx.JSON(http.StatusOK, bookings)
}

// listMyApplications lets a tasker track what they have applied for.
func (server *Server) listMyApplications(ctx *gin.Context) {
	profile, ok := server.currentTaskerProfile(ctx)
	if !ok {
		return
	}

	rows, err := server.store.ListApplicationsByTasker(ctx, profile.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	applications := make([]applicationResponse, 0, len(rows))
	for _, row := range rows {
		application := newApplicationResponse(row.TaskApplication)
		application.Task = newTaskRef(row.Task, row.Category, row.User)
		applications = append(applications, application)
	}
	ctx.JSON(http.StatusOK, applications)
}

func (server *Server) respondWithCurrentTaskerProfile(ctx *gin.Context) {
	row, err := server.store.GetTaskerProfileWithUserByUserID(ctx, currentUserID(ctx))
	if err != nil {
		ctx.JSON(http.StatusNotFound, errorResponse(errTaskerNotFound))
		return
	}
	server.respondWithTaskerProfile(ctx, row.TaskerProfile, row.User, http.StatusOK)
}

// isHourMinute reports whether value is a zero-padded 24-hour "HH:MM".
func isHourMinute(value string) bool {
	if len(value) != 5 || value[2] != ':' {
		return false
	}
	for index, char := range value {
		if index == 2 {
			continue
		}
		if char < '0' || char > '9' {
			return false
		}
	}
	hours := int(value[0]-'0')*10 + int(value[1]-'0')
	minutes := int(value[3]-'0')*10 + int(value[4]-'0')
	return hours <= 23 && minutes <= 59
}
