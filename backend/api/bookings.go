package api

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	db "wera-chap-chap/backend/db/sqlc"
)

var errOnlyAssignedTasker = errors.New("only the assigned tasker can do that")

// listBookings returns every booking the caller takes part in, on either side.
func (server *Server) listBookings(ctx *gin.Context) {
	userID := currentUserID(ctx)

	params := db.ListBookingsForUserParams{ClientID: userID}
	if profile, found := server.taskerProfile(ctx, userID); found {
		params.TaskerID = &profile.ID
	}

	rows, err := server.store.ListBookingsForUser(ctx, params)
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

func (server *Server) getBooking(ctx *gin.Context) {
	scope, ok := server.loadBooking(ctx, "id")
	if !ok {
		return
	}
	server.respondWithBooking(ctx, scope.Booking.ID, http.StatusOK)
}

func (server *Server) startBooking(ctx *gin.Context) {
	server.transitionBooking(ctx, db.BookingStarted)
}

func (server *Server) completeBooking(ctx *gin.Context) {
	server.transitionBooking(ctx, db.BookingCompleted)
}

func (server *Server) cancelBooking(ctx *gin.Context) {
	server.transitionBooking(ctx, db.BookingCancelled)
}

// allowedTransitions is the booking lifecycle: a booking is confirmed on
// creation, started by the tasker, then completed. Either party may cancel
// before the work is finished.
var allowedTransitions = map[string][]string{
	db.BookingConfirmed: {db.BookingStarted, db.BookingCancelled},
	db.BookingStarted:   {db.BookingCompleted, db.BookingCancelled},
	db.BookingCompleted: {},
	db.BookingCancelled: {},
}

func canTransition(from, to string) bool {
	for _, candidate := range allowedTransitions[from] {
		if candidate == to {
			return true
		}
	}
	return false
}

func (server *Server) transitionBooking(ctx *gin.Context, status string) {
	scope, ok := server.loadBooking(ctx, "id")
	if !ok {
		return
	}
	booking := scope.Booking

	// Only the assigned tasker drives the work forward; cancelling is open to
	// both parties.
	if status != db.BookingCancelled && !scope.IsTasker {
		ctx.JSON(http.StatusForbidden, errorResponse(errOnlyAssignedTasker))
		return
	}
	if !canTransition(booking.Status, status) {
		ctx.JSON(http.StatusConflict, errorResponse(
			fmt.Errorf("cannot move booking from %s to %s", booking.Status, status)))
		return
	}

	now := time.Now()
	params := db.TransitionBookingTxParams{
		BookingID:     booking.ID,
		TaskID:        booking.TaskID,
		BookingStatus: status,
		// A started booking means work is under way on the task; the two are
		// moved together so they cannot disagree.
		TaskStatus: db.TaskInProgress,
	}

	switch status {
	case db.BookingStarted:
		params.StartedAt = &now
	case db.BookingCompleted:
		params.CompletedAt = &now
		params.TaskStatus = db.TaskCompleted
	case db.BookingCancelled:
		params.TaskStatus = db.TaskCancelled
	}

	if err := server.store.TransitionBookingTx(ctx, params); err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	server.respondWithBooking(ctx, booking.ID, http.StatusOK)
}

func (server *Server) respondWithBooking(ctx *gin.Context, id int64, status int) {
	row, err := server.store.GetBookingWithRelations(ctx, id)
	if err != nil {
		if db.IsNotFound(err) {
			ctx.JSON(http.StatusNotFound, errorResponse(errBookingNotFound))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(status, newBookingWithRelations(
		row.Booking, row.Task, row.Category, row.TaskerProfile, row.User, row.User_2))
}
