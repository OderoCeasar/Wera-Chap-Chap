package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	db "wera-chap-chap/backend/db/sqlc"
	"wera-chap-chap/backend/token"
	"wera-chap-chap/backend/utils"
)

var (
	errEmailTaken         = errors.New("email already registered")
	errInvalidCredentials = errors.New("invalid credentials")
	errInvalidRefresh     = errors.New("invalid refresh token")
	errInvalidRole        = errors.New("role must be client or tasker")
)

type registerRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	FullName string `json:"full_name" binding:"required"`
	Phone    string `json:"phone"`
	Role     string `json:"role" binding:"required"`
}

type authResponse struct {
	User         userResponse `json:"user"`
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
}

func (server *Server) register(ctx *gin.Context) {
	var req registerRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	if !db.IsValidRole(req.Role) {
		ctx.JSON(http.StatusBadRequest, errorResponse(errInvalidRole))
		return
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	result, err := server.store.RegisterUserTx(ctx, db.RegisterUserTxParams{
		CreateUserParams: db.CreateUserParams{
			Email:        req.Email,
			PasswordHash: hashedPassword,
			FullName:     req.FullName,
			Phone:        req.Phone,
			Role:         req.Role,
			// A client is usable immediately; a tasker still has to pass
			// verification before they are trusted in the directory.
			IsVerified: req.Role == db.RoleClient,
		},
	})
	if err != nil {
		if db.IsUniqueViolation(err) {
			ctx.JSON(http.StatusConflict, errorResponse(errEmailTaken))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	server.respondWithTokens(ctx, result.User, http.StatusCreated)
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func (server *Server) login(ctx *gin.Context) {
	var req loginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	user, err := server.store.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if db.IsNotFound(err) {
			// Deliberately the same answer as a wrong password: distinguishing
			// them turns this endpoint into a way to enumerate registered
			// addresses.
			ctx.JSON(http.StatusUnauthorized, errorResponse(errInvalidCredentials))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	if err := utils.CheckPassword(req.Password, user.PasswordHash); err != nil {
		ctx.JSON(http.StatusUnauthorized, errorResponse(errInvalidCredentials))
		return
	}

	server.respondWithTokens(ctx, user, http.StatusOK)
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func (server *Server) refreshAccessToken(ctx *gin.Context) {
	var req refreshRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	payload, err := server.tokenMaker.VerifyToken(req.RefreshToken)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, errorResponse(errInvalidRefresh))
		return
	}
	if err := payload.RequireType(token.TypeRefresh); err != nil {
		ctx.JSON(http.StatusUnauthorized, errorResponse(errInvalidRefresh))
		return
	}

	// The role is re-read from the account rather than carried over from the
	// refresh token, so a role changed since the token was issued takes effect
	// on the next refresh instead of persisting for the token's whole life.
	user, err := server.store.GetUser(ctx, payload.UserID)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, errorResponse(errInvalidRefresh))
		return
	}

	accessToken, _, err := server.tokenMaker.CreateToken(
		user.ID, user.Role, token.TypeAccess, server.config.AccessTokenDuration)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"access_token": accessToken})
}

// logout is a client-side operation: the tokens are stateless, so there is
// nothing to revoke server-side. It exists so the client has one endpoint to
// call and so adding a revocation list later needs no client change.
func (server *Server) logout(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

func (server *Server) respondWithTokens(ctx *gin.Context, user db.User, status int) {
	accessToken, _, err := server.tokenMaker.CreateToken(
		user.ID, user.Role, token.TypeAccess, server.config.AccessTokenDuration)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	refreshToken, _, err := server.tokenMaker.CreateToken(
		user.ID, user.Role, token.TypeRefresh, server.config.RefreshTokenDuration)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(status, authResponse{
		User:         newUserResponse(user),
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}
