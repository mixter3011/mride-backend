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
		&models.RideSubscription{},
		&models.SubscriptionSubscriber{},
		&models.SubscriptionNotification{},
		&services.UserConnection{},
		&services.ConnectionLog{},

		&services.DeviceToken{},
		&services.PushNotificationLog{},
		&services.NotificationPreference{},
	)
	if err != nil {
		log.Printf("AutoMigrate error: %v", err)
		return nil, err
	}

	db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_user_fcm_tokens_user_device ON user_fcm_tokens(user_id, device_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_ride_subscriptions_recurring_days ON ride_subscriptions USING gin(recurring_days)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_subscription_subscribers_unique ON subscription_subscribers(subscription_id, subscriber_id) WHERE status = 'active'")

	if !db.Migrator().HasTable(&services.UserConnection{}) {
		log.Println("ERROR: user_connections table was not created")
		return nil, err
	}
	if !db.Migrator().HasTable(&services.ConnectionLog{}) {
		log.Println("ERROR: connection_logs table was not created")
		return nil, err
	}

	if !db.Migrator().HasTable(&services.DeviceToken{}) {
		log.Println("ERROR: device_tokens table was not created")
	} else {
		log.Println("✓ device_tokens table verified")
	}
	if !db.Migrator().HasTable(&services.PushNotificationLog{}) {
		log.Println("ERROR: push_notification_logs table was not created")
	} else {
		log.Println("✓ push_notification_logs table verified")
	}
	if !db.Migrator().HasTable(&services.NotificationPreference{}) {
		log.Println("ERROR: notification_preferences table was not created")
	} else {
		log.Println("✓ notification_preferences table verified")
	}

	log.Println("Connected to database with GORM - all tables migrated successfully")
	return db, nil
}
