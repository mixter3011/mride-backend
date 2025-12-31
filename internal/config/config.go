package config

import (
	"fmt"
	"log"
	"os"
	"strings"

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
		FirebaseCredentials: loadFirebaseCredentials(),
		FirebaseProjectID:   getEnv("FIREBASE_PROJECT_ID", "mride-51861"),
	}
}

func loadFirebaseCredentials() string {

	if creds := os.Getenv("FIREBASE_CREDENTIALS"); creds != "" {
		if strings.HasPrefix(strings.TrimSpace(creds), "{") {
			return creds
		}
	}

	if credFile := os.Getenv("FIREBASE_CREDENTIALS_FILE"); credFile != "" {
		data, err := os.ReadFile(credFile)
		if err != nil {
			fmt.Printf("Warning: Failed to read Firebase credentials file %s: %v\n", credFile, err)
			return ""
		}
		return string(data)
	}

	defaultPath := "./mride-51861-firebase-adminsdk-fbsvc-08c24a73d0.json"
	if _, err := os.Stat(defaultPath); err == nil {
		data, err := os.ReadFile(defaultPath)
		if err != nil {
			fmt.Printf("Warning: Failed to read default Firebase credentials file: %v\n", err)
			return ""
		}
		log.Println("✓ Loaded Firebase credentials from default location")
		return string(data)
	}

	log.Println("Warning: No Firebase credentials found. Push notifications will be disabled.")
	return ""
}

func getEnv(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}
