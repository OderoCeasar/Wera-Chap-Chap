package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	db "wera-chap-chap/backend/db/sqlc"
	"wera-chap-chap/backend/utils"
)

var (
	errUserNotFound    = errors.New("user not found")
	errWrongPassword   = errors.New("current password is incorrect")
	errNothingToUpdate = errors.New("no fields to update")
)

func (server *Server) getMe(ctx *gin.Context) {
	server.respondWithMe(ctx, http.StatusOK)
}

type updateMeRequest struct {
	// Pointers so "not supplied" is distinguishable from a deliberate empty
	// string; the query COALESCEs a nil back to the stored value.
	FullName  *string `json:"full_name"`
	Phone     *string `json:"phone"`
	AvatarURL *string `json:"avatar_url"`
}

func (server *Server) updateMe(ctx *gin.Context) {
	var req updateMeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	if _, err := server.store.UpdateUser(ctx, db.UpdateUserParams{
		ID:        currentUserID(ctx),
		FullName:  req.FullName,
		Phone:     req.Phone,
		AvatarUrl: req.AvatarURL,
	}); err != nil {
		if db.IsNotFound(err) {
			ctx.JSON(http.StatusNotFound, errorResponse(errUserNotFound))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	server.respondWithMe(ctx, http.StatusOK)
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}

func (server *Server) changePassword(ctx *gin.Context) {
	var req changePasswordRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	userID := currentUserID(ctx)
	user, err := server.store.GetUser(ctx, userID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, errorResponse(errUserNotFound))
		return
	}

	if err := utils.CheckPassword(req.CurrentPassword, user.PasswordHash); err != nil {
		ctx.JSON(http.StatusForbidden, errorResponse(errWrongPassword))
		return
	}

	hashedPassword, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	if err := server.store.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{
		ID:           userID,
		PasswordHash: hashedPassword,
	}); err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "password updated"})
}

// respondWithMe renders the caller's account, nesting their tasker profile when
// they have one so the client can read role-specific fields from a single call.
func (server *Server) respondWithMe(ctx *gin.Context, status int) {
	user, err := server.store.GetUser(ctx, currentUserID(ctx))
	if err != nil {
		if db.IsNotFound(err) {
			ctx.JSON(http.StatusNotFound, errorResponse(errUserNotFound))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	response := newUserResponse(user)
	if profile, found := server.taskerProfile(ctx, user.ID); found {
		nested := newTaskerProfileResponse(profile)
		response.TaskerProfile = &nested
	}

	ctx.JSON(status, response)
}
