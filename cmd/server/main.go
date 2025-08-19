package main

import (
	"context"
	"fmt"
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
	gin.SetMode(gin.ReleaseMode)

	cfg := config.Load()

	database, err := db.New(cfg.DBUrl)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	sqlDB, err := database.DB()
	if err != nil {
		log.Fatal("Failed to get underlying sql.DB:", err)
	}
	defer sqlDB.Close()

	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(25)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	jwtSvc := services.NewJWTSvc(cfg.JWTSecret, database)
	authSvc := services.NewAuthSvc(database, jwtSvc)
	fmt.Println("Creating services...")
	otpSvc := services.NewOTPSvc(database, cfg.TwilioSID, cfg.TwilioToken, cfg.TwilioPhone)
	fmt.Println("OTP service created")

	webSocketSvc := services.NewWebSocketSvc(database)

	notificationSvc := services.NewNotificationSvc(database, webSocketSvc)
	rideSvc := services.NewRideSvc(database, notificationSvc)

	if err := rideSvc.CleanupExpiredRides(); err != nil {
		log.Printf("Warning: Failed to cleanup expired rides: %v", err)
	}

	authHandler := handlers.NewAuthHandler(authSvc, otpSvc)
	fmt.Println("Creating handlers...")
	otpHandler := handlers.NewOTPHandler(otpSvc, authSvc)
	fmt.Println("OTP handler created")
	rideHandler := handlers.NewRideHandler(rideSvc)
	notificationHandler := handlers.NewNotificationHandler(notificationSvc)
	webSocketHandler := handlers.NewWebSocketHandler(webSocketSvc, jwtSvc)

	r := gin.New()

	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(middleware.CORSMiddleware())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.RateLimitMiddleware())

	r.GET("/health", func(c *gin.Context) {
		if err := sqlDB.Ping(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":   "unhealthy",
				"database": "disconnected",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":   "healthy",
			"database": "connected",
		})
	})

	r.HEAD("/health", func(c *gin.Context) {
		if err := sqlDB.Ping(); err != nil {
			c.Status(http.StatusServiceUnavailable)
			return
		}
		c.Status(http.StatusOK)
	})

	r.POST("/auth/signup", authHandler.SignUp)
	r.POST("/auth/signin", authHandler.SignIn)
	r.POST("/auth/refresh", authHandler.RefreshToken)

	protected := r.Group("/", middleware.AuthMiddleware(jwtSvc))
	{
		protected.POST("/auth/logout", authHandler.Logout)

		protected.POST("/auth/send-otp", otpHandler.SendOTP)
		protected.POST("/auth/verify-otp", otpHandler.VerifyOTP)
		protected.POST("/auth/send-email-otp", otpHandler.SendEmailOTP)
		protected.POST("/auth/verify-email-otp", otpHandler.VerifyEmailOTP)
		protected.GET("/auth/profile", authHandler.GetProfile)

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
		protected.POST("/ride/:id/start", rideHandler.StartRide)
		protected.POST("/ride/:id/complete", rideHandler.CompleteRide)
		protected.GET("/ride/:id/progress", rideHandler.GetRideProgress)
		protected.GET("/rides/active", rideHandler.GetActiveRides)

		protected.GET("/notifications", notificationHandler.GetNotifications)
		protected.PUT("/notifications/:id/read", notificationHandler.MarkAsRead)
		protected.PUT("/notifications/read-all", notificationHandler.MarkAllAsRead)
		protected.GET("/notifications/unread-count", notificationHandler.GetUnreadCount)

		protected.GET("/users/online", webSocketHandler.GetOnlineUsers)
		protected.GET("/ws/online-db", webSocketHandler.GetOnlineUsersFromDB)
		protected.GET("/ws/history/:user_id", webSocketHandler.GetUserConnectionHistory)
		protected.GET("/ws/stats", webSocketHandler.GetConnectionStats)
		protected.POST("/ws/cleanup", webSocketHandler.CleanupStaleConnections)
	}

	r.GET("/ws", middleware.WebSocketAuthMiddleware(jwtSvc), webSocketHandler.HandleWebSocket)

	srv := &http.Server{
		Addr:           ":" + cfg.Port,
		Handler:        r,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20,
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
