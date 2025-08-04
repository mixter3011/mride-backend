package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBUrl               string
	JWTSecret           string
	Port                string
	TwilioSID           string
	TwilioToken         string
	TwilioPhone         string
	ResendAPIKey        string
	FirebaseCredentials string
	FirebaseProjectID   string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	return &Config{
		DBUrl:               getEnv("DB_URL", ""),
		JWTSecret:           getEnv("JWT_SECRET", "default-secret"),
		Port:                getEnv("PORT", "8080"),
		TwilioSID:           getEnv("TWILIO_SID", ""),
		TwilioToken:         getEnv("TWILIO_TOKEN", ""),
		TwilioPhone:         getEnv("TWILIO_PHONE", ""),
		ResendAPIKey:        getEnv("RESEND_API_KEY", ""),
		FirebaseCredentials: getEnv("FIREBASE_CREDENTIALS", ""),
		FirebaseProjectID:   getEnv("FIREBASE_PROJECT_ID", ""),
	}
}

func getEnv(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}
