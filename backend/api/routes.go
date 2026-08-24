package api

import (
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	db "wera-chap-chap/backend/db/sqlc"
)

// setupRouter declares every route in one place.
//
// Keeping the routing table here rather than in main means the shape of the
// API is readable in one file, and main is left doing nothing but wiring.
func (server *Server) setupRouter() {
	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{server.config.FrontendOrigin},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	api := router.Group("/api")
	api.GET("/health", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// --- Public ---
	auth := api.Group("/auth", server.AuthRateLimit())
	auth.POST("/register", server.register)
	auth.POST("/login", server.login)
	auth.POST("/refresh", server.refreshAccessToken)
	auth.POST("/logout", server.logout)

	api.GET("/categories", server.listCategories)
	api.GET("/taskers", server.listTaskers)
	api.GET("/taskers/:id", server.getTasker)
	// Provider webhook: authenticated by the checkout id it echoes back, so it
	// sits outside the auth middleware by necessity.
	api.POST("/payments/mpesa/callback", server.mpesaCallback)

	// --- Authenticated ---
	protected := api.Group("", server.AuthMiddleware())

	protected.GET("/users/me", server.getMe)
	protected.PUT("/users/me", server.updateMe)
	protected.PUT("/users/me/password", server.changePassword)

	protected.GET("/taskers/me", server.RequireRole(db.RoleTasker), server.getMyTaskerProfile)
	protected.PUT("/taskers/profile", server.RequireRole(db.RoleTasker), server.updateTaskerProfile)
	protected.POST("/taskers/availability", server.RequireRole(db.RoleTasker), server.setAvailability)
	protected.GET("/taskers/me/bookings", server.RequireRole(db.RoleTasker), server.listMyTaskerBookings)
	protected.GET("/taskers/me/applications", server.RequireRole(db.RoleTasker), server.listMyApplications)

	protected.POST("/tasks", server.RequireRole(db.RoleClient), server.createTask)
	protected.GET("/tasks", server.listTasks)
	protected.GET("/tasks/my", server.RequireRole(db.RoleClient), server.listMyTasks)
	protected.GET("/tasks/:id", server.getTask)
	protected.PUT("/tasks/:id", server.RequireRole(db.RoleClient), server.updateTask)
	protected.DELETE("/tasks/:id", server.RequireRole(db.RoleClient), server.cancelTask)
	protected.POST("/tasks/:id/apply", server.RequireRole(db.RoleTasker), server.applyToTask)
	protected.PUT("/tasks/:id/applications/:app_id/accept", server.RequireRole(db.RoleClient), server.acceptApplication)
	protected.PUT("/tasks/:id/applications/:app_id/reject", server.RequireRole(db.RoleClient), server.rejectApplication)
	protected.POST("/tasks/:id/matches", server.RequireRole(db.RoleClient), server.listTaskMatches)

	protected.GET("/bookings", server.listBookings)
	protected.GET("/bookings/:id", server.getBooking)
	protected.PUT("/bookings/:id/start", server.RequireRole(db.RoleTasker), server.startBooking)
	protected.PUT("/bookings/:id/complete", server.RequireRole(db.RoleTasker), server.completeBooking)
	protected.PUT("/bookings/:id/cancel", server.cancelBooking)

	protected.GET("/messages/booking/:booking_id", server.listMessages)
	protected.POST("/messages/booking/:booking_id", server.sendMessage)

	protected.GET("/reviews/tasker/:tasker_id", server.listTaskerReviews)
	protected.GET("/reviews/booking/:booking_id", server.listBookingReviews)
	protected.POST("/reviews/booking/:booking_id", server.createReview)

	protected.POST("/payments/booking/:booking_id/initiate", server.initiatePayment)
	protected.POST("/payments/booking/:booking_id/confirm", server.confirmPayment)
	protected.POST("/payments/booking/:booking_id/tip", server.tipPayment)
	protected.GET("/payments/booking/:booking_id", server.getPayment)

	// The browser WebSocket API cannot set an Authorization header, so this one
	// takes its token from the query string instead -- see WSAuthMiddleware.
	router.GET("/ws/booking/:booking_id", server.WSAuthMiddleware(), server.streamMessages)

	server.router = router
}
