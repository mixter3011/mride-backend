package db

import (
	"log"
	"mride-backend/internal/models"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func New(dbUrl string) (*gorm.DB, error) {
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		if testUrl := os.Getenv("DATABASE_URL"); testUrl != "" {
			dbUrl = testUrl
			log.Println("Using CI test database")
		}
	}

	db, err := gorm.Open(postgres.Open(dbUrl), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	if err := sqlDB.Ping(); err != nil {
		return nil, err
	}

	err = db.AutoMigrate(
		&models.User{},
		&models.OTP{},
		&models.PendingPhoneUpdate{},
		&models.Ride{},
		&models.RidePassenger{},
		&models.Notification{},
	)
	if err != nil {
		return nil, err
	}

	db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_user_fcm_tokens_user_device ON user_fcm_tokens(user_id, device_id)")
	log.Println("Connected to database with GORM")
	return db, nil
}
