package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	fcmSvc, err := services.NewFCMSvc(database.DB, cfg.FirebaseCredentials, cfg.FirebaseProjectID)
	if err != nil {
		log.Printf("Failed to initialize FCM service: %v", err)
		fcmSvc = nil
	}

	webSocketSvc := services.NewWebSocketSvc()

	notificationSvc := services.NewNotificationSvc(database.DB, fcmSvc, webSocketSvc)
	rideSvc := services.NewRideSvc(database.DB, notificationSvc)

	authHandler := handlers.NewAuthHandler(authSvc, otpSvc)
	otpHandler := handlers.NewOTPHandler(otpSvc, authSvc)
	rideHandler := handlers.NewRideHandler(rideSvc)
	notificationHandler := handlers.NewNotificationHandler(notificationSvc)
	fcmHandler := handlers.NewFCMHandler(notificationSvc)
	webSocketHandler := handlers.NewWebSocketHandler(webSocketSvc, jwtSvc)

	r := gin.Default()

	r.POST("/auth/signup", authHandler.SignUp)
	r.POST("/auth/signin", authHandler.SignIn)

	protected := r.Group("/", middleware.AuthMiddleware(jwtSvc))
	{
		protected.GET("/auth/profile", authHandler.GetProfile)
		protected.POST("/auth/send-otp", otpHandler.SendOTP)
		protected.POST("/auth/verify-otp", otpHandler.VerifyOTP)

		protected.PUT("/update/profile", authHandler.UpdateProfile)
		protected.PUT("/update/password", authHandler.UpdatePassword)
		protected.POST("/update/phone/request-update", authHandler.RequestPhoneUpdate)
		protected.POST("/update/phone/confirm-update", authHandler.ConfirmPhoneUpdate)
		protected.PUT("/update/location", authHandler.UpdateLocation)

		protected.POST("/ride/create", rideHandler.CreateRide)
		protected.GET("/rides/my", rideHandler.GetMyRides)
		protected.POST("/ride/:id/join", rideHandler.JoinRide)
		protected.DELETE("/ride/:id/delete", rideHandler.DeleteRide)
		protected.DELETE("/ride/:id/leave", rideHandler.LeaveRide)
		protected.GET("/rides/search", rideHandler.SearchRides)
		protected.GET("/rides/nearby", rideHandler.GetNearbyRides)
		protected.GET("/rides/all", rideHandler.GetAllRides)

		protected.GET("/notifications", notificationHandler.GetNotifications)
		protected.PUT("/notifications/:id/read", notificationHandler.MarkAsRead)
		protected.PUT("/notifications/read-all", notificationHandler.MarkAllAsRead)
		protected.GET("/notifications/unread-count", notificationHandler.GetUnreadCount)

		protected.POST("/fcm/token", fcmHandler.SaveFCMToken)
		protected.DELETE("/fcm/token", fcmHandler.RemoveFCMToken)

		protected.GET("/users/online", webSocketHandler.GetOnlineUsers)
	}

	r.GET("/ws", middleware.WebSocketAuthMiddleware(jwtSvc), webSocketHandler.HandleWebSocket)

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	go func() {
		log.Printf("Server starting on port %s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Failed to start server:", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	webSocketSvc.Shutdown()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
}
