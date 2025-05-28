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

	authHandler := handlers.NewAuthHandler(authSvc)
	otpHandler := handlers.NewOTPHandler(otpSvc, authSvc)

	r := gin.Default()

	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	api := r.Group("/api")
	{
		api.POST("/signup", authHandler.SignUp)
		api.POST("/login", authHandler.SignIn)
	}

	protected := api.Group("/")
	protected.Use(middleware.AuthMiddleware(jwtSvc))
	{
		protected.GET("/profile", authHandler.GetProfile)
		protected.POST("/send-otp", otpHandler.SendOTP)
		protected.POST("/verify-otp", otpHandler.VerifyOTP)
	}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	log.Printf("Server starting on port %s", cfg.Port)
	log.Fatal(r.Run(":" + cfg.Port))
}
