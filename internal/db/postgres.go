package db

import (
	"log"
	"mride-backend/internal/models"
	"mride-backend/internal/services"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func New(dbUrl string) (*gorm.DB, error) {
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		if testUrl := os.Getenv("DATABASE_URL"); testUrl != "" {
			dbUrl = testUrl
			log.Println("Using CI test database")
		}
	}

	db, err := gorm.Open(postgres.Open(dbUrl), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
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

	db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"")

	err = db.AutoMigrate(
		&models.User{},
		&models.OTP{},
		&models.PendingPhoneUpdate{},
		&models.Ride{},
		&models.RidePassenger{},
		&models.Notification{},
		&services.UserConnection{},
		&services.ConnectionLog{},
	)
	if err != nil {
		log.Printf("AutoMigrate error: %v", err)
		return nil, err
	}

	db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_user_fcm_tokens_user_device ON user_fcm_tokens(user_id, device_id)")

	if !db.Migrator().HasTable(&services.UserConnection{}) {
		log.Println("ERROR: user_connections table was not created")
		return nil, err
	}
	if !db.Migrator().HasTable(&services.ConnectionLog{}) {
		log.Println("ERROR: connection_logs table was not created")
		return nil, err
	}

	log.Println("Connected to database with GORM - all tables migrated successfully")
	return db, nil
}
