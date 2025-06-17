package main

import (
	"log"

	"mride-backend/internal/config"
	"mride-backend/internal/db"
	"mride-backend/internal/handlers"
	"mride-backend/internal/middleware"
	"mride-backend/internal/services"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	database, err := db.New(cfg.DBUrl)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer database.Close()

	if err := database.Migrate(); err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	jwtSvc := services.NewJWTSvc(cfg.JWTSecret)
	authSvc := services.NewAuthSvc(database.DB, jwtSvc)
	otpSvc := services.NewOTPSvc(database.DB, cfg.TwilioSID, cfg.TwilioToken, cfg.TwilioPhone)

	fcmSvc, err := services.NewFCMSvc(database.DB, cfg.FirebaseCredentials)
	if err != nil {
		log.Printf("Failed to initialize FCM service: %v", err)
		fcmSvc = nil
	}

	notificationSvc := services.NewNotificationSvc(database.DB, fcmSvc)
	rideSvc := services.NewRideSvc(database.DB, notificationSvc)

	authHandler := handlers.NewAuthHandler(authSvc)
	otpHandler := handlers.NewOTPHandler(otpSvc, authSvc)
	rideHandler := handlers.NewRideHandler(rideSvc)
	notificationHandler := handlers.NewNotificationHandler(notificationSvc)
	fcmHandler := handlers.NewFCMHandler(notificationSvc)

	r := gin.Default()

	r.POST("/auth/signup", authHandler.SignUp)
	r.POST("/auth/signin", authHandler.SignIn)

	protected := r.Group("/", middleware.AuthMiddleware(jwtSvc))
	{
		protected.GET("/auth/profile", authHandler.GetProfile)
		protected.POST("/auth/send-otp", otpHandler.SendOTP)
		protected.POST("/auth/verify-otp", otpHandler.VerifyOTP)

		protected.POST("/ride/create", rideHandler.CreateRide)
		protected.GET("/rides/my", rideHandler.GetMyRides)
		protected.POST("/ride/:id/join", rideHandler.JoinRide)
		protected.GET("/rides/search", rideHandler.SearchRides)
		protected.GET("/rides/nearby", rideHandler.GetNearbyRides)
		protected.GET("/rides/all", rideHandler.GetAllRides)

		protected.GET("/notifications", notificationHandler.GetNotifications)
		protected.PUT("/notifications/:id/read", notificationHandler.MarkAsRead)
		protected.PUT("/notifications/read-all", notificationHandler.MarkAllAsRead)
		protected.GET("/notifications/unread-count", notificationHandler.GetUnreadCount)

		protected.POST("/fcm/token", fcmHandler.SaveFCMToken)
		protected.DELETE("/fcm/token", fcmHandler.RemoveFCMToken)
	}

	log.Printf("Server starting on port %s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
