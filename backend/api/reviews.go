package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	db "wera-chap-chap/backend/db/sqlc"
)

var (
	errReviewNotCompleted = errors.New("you can only review a completed booking")
	errAlreadyReviewed    = errors.New("you already reviewed this booking")
)

type createReviewRequest struct {
	Rating  int32  `json:"rating" binding:"required,min=1,max=5"`
	Comment string `json:"comment"`
}

func (server *Server) createReview(ctx *gin.Context) {
	scope, ok := server.loadBooking(ctx, "booking_id")
	if !ok {
		return
	}
	booking := scope.Booking

	if booking.Status != db.BookingCompleted {
		ctx.JSON(http.StatusConflict, errorResponse(errReviewNotCompleted))
		return
	}

	var req createReviewRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Each side reviews the other: the client rates the tasker's user account,
	// the tasker rates the client.
	var revieweeID int64
	if scope.IsClient {
		profile, err := server.store.GetTaskerProfile(ctx, booking.TaskerID)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, errorResponse(errTaskerNotFound))
			return
		}
		revieweeID = profile.UserID
	} else {
		revieweeID = booking.ClientID
	}

	review, err := server.store.CreateReviewTx(ctx, db.CreateReviewTxParams{
		CreateReviewParams: db.CreateReviewParams{
			BookingID:  booking.ID,
			ReviewerID: currentUserID(ctx),
			RevieweeID: revieweeID,
			Rating:     req.Rating,
			Comment:    req.Comment,
		},
	})
	if err != nil {
		// (booking_id, reviewer_id) is UNIQUE, so a second review is rejected
		// by the insert rather than by a prior count.
		if db.IsUniqueViolation(err) {
			ctx.JSON(http.StatusConflict, errorResponse(errAlreadyReviewed))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	row, err := server.store.GetReviewWithReviewer(ctx, review.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusCreated, newReviewResponse(row.Review, row.User))
}

// listBookingReviews returns the reviews attached to a booking so the UI can
// tell whether the caller has already left one.
func (server *Server) listBookingReviews(ctx *gin.Context) {
	scope, ok := server.loadBooking(ctx, "booking_id")
	if !ok {
		return
	}

	rows, err := server.store.ListReviewsByBooking(ctx, scope.Booking.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, newReviewResponses(rowsToReviewPairs(rows)))
}

func (server *Server) listTaskerReviews(ctx *gin.Context) {
	taskerID, ok := parseInt64Param(ctx, "tasker_id")
	if !ok {
		ctx.JSON(http.StatusBadRequest, errorResponse(errInvalidID))
		return
	}

	// The path carries a tasker profile id, but reviews are keyed on the user
	// account behind it -- a client's reviews live on the same table.
	profile, err := server.store.GetTaskerProfile(ctx, taskerID)
	if err != nil {
		if db.IsNotFound(err) {
			ctx.JSON(http.StatusNotFound, errorResponse(errTaskerNotFound))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	rows, err := server.store.ListReviewsByReviewee(ctx, profile.UserID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	reviews := make([]reviewResponse, 0, len(rows))
	for _, row := range rows {
		reviews = append(reviews, newReviewResponse(row.Review, row.User))
	}
	ctx.JSON(http.StatusOK, reviews)
}

// reviewPair is the (review, reviewer) shape both list queries return under
// different generated row types.
type reviewPair struct {
	Review   db.Review
	Reviewer db.User
}

func rowsToReviewPairs(rows []db.ListReviewsByBookingRow) []reviewPair {
	pairs := make([]reviewPair, 0, len(rows))
	for _, row := range rows {
		pairs = append(pairs, reviewPair{Review: row.Review, Reviewer: row.User})
	}
	return pairs
}

func newReviewResponses(pairs []reviewPair) []reviewResponse {
	reviews := make([]reviewResponse, 0, len(pairs))
	for _, pair := range pairs {
		reviews = append(reviews, newReviewResponse(pair.Review, pair.Reviewer))
	}
	return reviews
}
