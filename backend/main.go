package main

import (
	"log"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"wera-chap-chap/backend/config"
	"wera-chap-chap/backend/handlers"
	"wera-chap-chap/backend/middleware"
	"wera-chap-chap/backend/models"
	chat "wera-chap-chap/backend/websocket"
)

func main() {
	cfg := config.Load()
	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{})
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{}, &models.TaskerProfile{}, &models.Category{}, &models.TaskerSkill{},
		&models.TaskerAvailability{}, &models.Task{}, &models.TaskApplication{},
		&models.Booking{}, &models.Message{}, &models.Review{}, &models.Payment{},
	); err != nil {
		log.Fatalf("migrate database: %v", err)
	}
	seedCategories(db)
	seedDemoData(db)

	hub := chat.NewHub()
	go hub.Run()

	router := gin.Default()
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{cfg.FrontendOrigin},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	auth := handlers.NewAuthHandler(db, cfg)
	users := handlers.NewUserHandler(db)
	taskers := handlers.NewTaskerHandler(db)
	tasks := handlers.NewTaskHandler(db)
	bookings := handlers.NewBookingHandler(db)
	messages := handlers.NewMessageHandler(db, hub)
	reviews := handlers.NewReviewHandler(db)
	payments := handlers.NewPaymentHandler(db, cfg)

	api := router.Group("/api")
	api.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })
	authGroup := api.Group("/auth")
	authGroup.Use(middleware.AuthRateLimit())
	authGroup.POST("/register", auth.Register)
	authGroup.POST("/login", auth.Login)
	authGroup.POST("/refresh", auth.Refresh)
	authGroup.POST("/logout", auth.Logout)

	api.GET("/categories", handlers.ListCategories(db))
	api.GET("/taskers", taskers.List)
	api.GET("/taskers/:id", taskers.Get)
	// Provider webhook: authenticated by the checkout id it echoes back.
	api.POST("/payments/mpesa/callback", payments.MpesaCallback)

	protected := api.Group("")
	protected.Use(middleware.JWT(cfg.JWTSecret))
	protected.GET("/users/me", users.Me)
	protected.PUT("/users/me", users.UpdateMe)
	protected.PUT("/users/me/password", users.ChangePassword)
	protected.GET("/taskers/me", middleware.RequireRole(models.RoleTasker), taskers.Me)
	protected.PUT("/taskers/profile", middleware.RequireRole(models.RoleTasker), taskers.UpdateProfile)
	protected.POST("/taskers/availability", middleware.RequireRole(models.RoleTasker), taskers.SetAvailability)
	protected.GET("/taskers/me/bookings", middleware.RequireRole(models.RoleTasker), taskers.MyBookings)
	protected.GET("/taskers/me/applications", middleware.RequireRole(models.RoleTasker), taskers.MyApplications)

	protected.POST("/tasks", middleware.RequireRole(models.RoleClient), tasks.Create)
	protected.GET("/tasks", tasks.List)
	protected.GET("/tasks/my", middleware.RequireRole(models.RoleClient), tasks.MyTasks)
	protected.GET("/tasks/:id", tasks.Get)
	protected.PUT("/tasks/:id", middleware.RequireRole(models.RoleClient), tasks.Update)
	protected.DELETE("/tasks/:id", middleware.RequireRole(models.RoleClient), tasks.Cancel)
	protected.POST("/tasks/:id/apply", middleware.RequireRole(models.RoleTasker), tasks.Apply)
	protected.PUT("/tasks/:id/applications/:app_id/accept", middleware.RequireRole(models.RoleClient), tasks.AcceptApplication)
	protected.PUT("/tasks/:id/applications/:app_id/reject", middleware.RequireRole(models.RoleClient), tasks.RejectApplication)
	protected.POST("/tasks/:id/matches", middleware.RequireRole(models.RoleClient), tasks.Matches)

	protected.GET("/bookings", bookings.List)
	protected.GET("/bookings/:id", bookings.Get)
	protected.PUT("/bookings/:id/start", middleware.RequireRole(models.RoleTasker), bookings.Start)
	protected.PUT("/bookings/:id/complete", middleware.RequireRole(models.RoleTasker), bookings.Complete)
	protected.PUT("/bookings/:id/cancel", bookings.Cancel)

	protected.GET("/messages/booking/:booking_id", messages.History)
	protected.POST("/messages/booking/:booking_id", messages.Send)
	protected.GET("/reviews/tasker/:tasker_id", reviews.ForTasker)
	protected.GET("/reviews/booking/:booking_id", reviews.ForBooking)
	protected.POST("/reviews/booking/:booking_id", reviews.Create)
	protected.POST("/payments/booking/:booking_id/initiate", payments.Initiate)
	protected.POST("/payments/booking/:booking_id/confirm", payments.Confirm)
	protected.POST("/payments/booking/:booking_id/tip", payments.Tip)
	protected.GET("/payments/booking/:booking_id", payments.Get)

	router.GET("/ws/booking/:booking_id", middleware.WSJWT(cfg.JWTSecret), messages.WebSocket)

	log.Printf("Wera Chap Chap backend listening on %s", cfg.Addr())
	if err := router.Run(cfg.Addr()); err != nil {
		log.Fatal(err)
	}
}

func seedCategories(db *gorm.DB) {
	categories := []models.Category{
		{Name: "Home Repairs", IconURL: "🛠️", Description: "Fixes, maintenance and odd jobs around the home."},
		{Name: "Furniture Assembly", IconURL: "🪑", Description: "Flat-pack and custom furniture assembly."},
		{Name: "Cleaning", IconURL: "🧽", Description: "Home, office and post-event cleaning."},
		{Name: "Moving", IconURL: "🚚", Description: "Packing, lifting and relocation help."},
		{Name: "Delivery & Errands", IconURL: "🛵", Description: "Fast errands and item delivery."},
		{Name: "Yard Work", IconURL: "🌿", Description: "Gardening, mowing and outdoor cleanup."},
		{Name: "Personal Assistant", IconURL: "📋", Description: "Admin, scheduling and personal support."},
		{Name: "Handyman", IconURL: "🔧", Description: "General repairs and skilled help."},
	}
	for _, category := range categories {
		db.FirstOrCreate(&category, models.Category{Name: category.Name})
	}
}
