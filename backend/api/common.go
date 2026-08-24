package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	db "wera-chap-chap/backend/db/sqlc"
)

var (
	errBookingNotFound  = errors.New("booking not found")
	errNotYourBooking   = errors.New("not your booking")
	errTaskerProfileReq = errors.New("tasker profile required")
	errInvalidID        = errors.New("invalid id")
)

// parseInt64Param reads a positive path parameter. Zero is rejected along with
// non-numeric input because no row ever has id 0, so a 0 could only come from a
// malformed request.
func parseInt64Param(ctx *gin.Context, name string) (int64, bool) {
	value, err := strconv.ParseInt(ctx.Param(name), 10, 64)
	if err != nil || value <= 0 {
		return 0, false
	}
	return value, true
}

// bookingScope describes how the current caller relates to a booking.
type bookingScope struct {
	Booking  db.Booking
	IsClient bool
	IsTasker bool
	// TaskerProfileID is the caller's own profile id when IsTasker, not the
	// booking's.
	TaskerProfileID int64
}

// loadBooking fetches the booking named by param and confirms the caller takes
// part in it. It writes the error response itself and reports whether the
// caller may continue.
func (server *Server) loadBooking(ctx *gin.Context, param string) (bookingScope, bool) {
	var scope bookingScope

	id, ok := parseInt64Param(ctx, param)
	if !ok {
		ctx.JSON(http.StatusBadRequest, errorResponse(errInvalidID))
		return scope, false
	}

	booking, err := server.store.GetBooking(ctx, id)
	if err != nil {
		if db.IsNotFound(err) {
			ctx.JSON(http.StatusNotFound, errorResponse(errBookingNotFound))
			return scope, false
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return scope, false
	}
	scope.Booking = booking

	userID := currentUserID(ctx)
	scope.IsClient = booking.ClientID == userID
	if !scope.IsClient {
		if profile, found := server.taskerProfile(ctx, userID); found && profile.ID == booking.TaskerID {
			scope.IsTasker = true
			scope.TaskerProfileID = profile.ID
		}
	}

	if !scope.IsClient && !scope.IsTasker {
		ctx.JSON(http.StatusForbidden, errorResponse(errNotYourBooking))
		return scope, false
	}
	return scope, true
}

// taskerProfile loads the tasker profile owned by userID, reporting whether one
// exists. A client legitimately has none, so absence is not an error here.
func (server *Server) taskerProfile(ctx *gin.Context, userID int64) (db.TaskerProfile, bool) {
	profile, err := server.store.GetTaskerProfileByUserID(ctx, userID)
	if err != nil {
		return db.TaskerProfile{}, false
	}
	return profile, true
}

// currentTaskerProfile resolves the caller's tasker profile, answering 403 when
// they have none.
func (server *Server) currentTaskerProfile(ctx *gin.Context) (db.TaskerProfile, bool) {
	profile, found := server.taskerProfile(ctx, currentUserID(ctx))
	if !found {
		ctx.JSON(http.StatusForbidden, errorResponse(errTaskerProfileReq))
		return profile, false
	}
	return profile, true
}

// attachTaskerRelations fills in skills and availability for a page of tasker
// profiles using two queries for the whole set, rather than two per profile.
// This is the batching the Preload-based version did not do.
func (server *Server) attachTaskerRelations(ctx *gin.Context, profiles []*taskerProfileResponse) error {
	if len(profiles) == 0 {
		return nil
	}

	ids := make([]int64, 0, len(profiles))
	for _, profile := range profiles {
		ids = append(ids, profile.ID)
	}

	skillRows, err := server.store.ListTaskerSkillsByTaskerIDs(ctx, ids)
	if err != nil {
		return err
	}
	skillsByTasker := make(map[int64][]skillResponse, len(profiles))
	for _, row := range skillRows {
		taskerID := row.TaskerSkill.TaskerID
		skillsByTasker[taskerID] = append(skillsByTasker[taskerID], skillResponse{
			ID:         row.TaskerSkill.ID,
			TaskerID:   taskerID,
			CategoryID: row.TaskerSkill.CategoryID,
			Category:   row.Category,
		})
	}

	slots, err := server.store.ListTaskerAvailabilityByTaskerIDs(ctx, ids)
	if err != nil {
		return err
	}
	slotsByTasker := make(map[int64][]db.TaskerAvailability, len(profiles))
	for _, slot := range slots {
		slotsByTasker[slot.TaskerID] = append(slotsByTasker[slot.TaskerID], slot)
	}

	for _, profile := range profiles {
		if skills, found := skillsByTasker[profile.ID]; found {
			profile.Skills = skills
		}
		if availability, found := slotsByTasker[profile.ID]; found {
			profile.Availability = availability
		}
	}
	return nil
}

// respondWithTaskerProfile renders one profile with its skills and
// availability, which is the shape both the tasker dashboard and the public
// profile page read.
func (server *Server) respondWithTaskerProfile(ctx *gin.Context, profile db.TaskerProfile, user db.User, status int) {
	response := newTaskerProfileRef(profile, user)

	skills, err := server.store.ListTaskerSkills(ctx, profile.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	response.Skills = newSkillResponsesFromSingle(skills)

	availability, err := server.store.ListTaskerAvailability(ctx, profile.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	response.Availability = availability

	ctx.JSON(status, response)
}

// attachApplications loads the applications for a page of tasks in one query
// and nests them, matching what Preload("Applications.Tasker.User") produced.
func (server *Server) attachApplications(ctx *gin.Context, tasks []*taskResponse) error {
	if len(tasks) == 0 {
		return nil
	}

	ids := make([]int64, 0, len(tasks))
	for _, task := range tasks {
		ids = append(ids, task.ID)
	}

	rows, err := server.store.ListApplicationsByTaskIDs(ctx, ids)
	if err != nil {
		return err
	}

	byTask := make(map[int64][]applicationResponse, len(tasks))
	for _, row := range rows {
		application := newApplicationResponse(row.TaskApplication)
		application.Tasker = newTaskerProfileRef(row.TaskerProfile, row.User)
		byTask[row.TaskApplication.TaskID] = append(byTask[row.TaskApplication.TaskID], application)
	}

	for _, task := range tasks {
		// Assigned unconditionally, including the empty slice: a task the
		// client can act on needs to render "no applications yet" rather than
		// leave the key absent.
		applications := byTask[task.ID]
		if applications == nil {
			applications = []applicationResponse{}
		}
		task.Applications = applications
	}
	return nil
}
