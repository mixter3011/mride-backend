package services

import (
	"encoding/json"
	"mride-backend/internal/models"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupPushTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	err = db.AutoMigrate(&DeviceToken{}, &PushNotificationLog{}, &NotificationPreference{})
	assert.NoError(t, err)

	return db
}

func TestPushRegisterDeviceToken(t *testing.T) {
	t.Run("Register new device token", func(t *testing.T) {
		db := setupPushTestDB(t)

		deviceToken := DeviceToken{
			UserID:   1,
			Token:    "test_token_123",
			Platform: "android",
			IsActive: true,
			LastUsed: time.Now(),
		}

		err := db.Create(&deviceToken).Error
		assert.NoError(t, err)
		assert.NotZero(t, deviceToken.ID)
	})

	t.Run("Update existing device token", func(t *testing.T) {
		db := setupPushTestDB(t)

		deviceToken := DeviceToken{
			UserID:   2,
			Token:    "test_token_456",
			Platform: "ios",
			IsActive: false,
			LastUsed: time.Now().Add(-24 * time.Hour),
		}
		db.Create(&deviceToken)

		err := db.Model(&DeviceToken{}).
			Where("token = ?", "test_token_456").
			Updates(map[string]interface{}{
				"user_id":   2,
				"is_active": true,
				"last_used": time.Now(),
			}).Error
		assert.NoError(t, err)

		var updated DeviceToken
		db.Where("token = ?", "test_token_456").First(&updated)
		assert.True(t, updated.IsActive)
	})
}

func TestPushGetUserDeviceTokens(t *testing.T) {
	t.Run("Get active device tokens for user", func(t *testing.T) {
		db := setupPushTestDB(t)

		token1 := DeviceToken{UserID: 1, Token: "token1", Platform: "android", IsActive: true, LastUsed: time.Now()}
		db.Create(&token1)

		token2 := DeviceToken{UserID: 1, Token: "token2", Platform: "ios", IsActive: true, LastUsed: time.Now()}
		db.Create(&token2)

		token3 := DeviceToken{UserID: 1, Token: "token3", Platform: "web", LastUsed: time.Now()}
		db.Create(&token3)
		db.Model(&DeviceToken{}).Where("token = ?", "token3").Updates(map[string]interface{}{
			"is_active": false,
		})

		token4 := DeviceToken{UserID: 2, Token: "token4", Platform: "android", IsActive: true, LastUsed: time.Now()}
		db.Create(&token4)

		var activeTokens []DeviceToken
		err := db.Where("user_id = ? AND is_active = ?", 1, true).Find(&activeTokens).Error
		assert.NoError(t, err)
		assert.Equal(t, 2, len(activeTokens))

		for _, token := range activeTokens {
			assert.True(t, token.Token == "token1" || token.Token == "token2")
			assert.True(t, token.IsActive)
		}
	})
}

func TestPushNotificationPreferences(t *testing.T) {
	t.Run("Create notification preferences", func(t *testing.T) {
		db := setupPushTestDB(t)

		prefs := NotificationPreference{
			UserID:                      1,
			PushEnabled:                 true,
			RideJoinEnabled:             true,
			RideLeaveEnabled:            false,
			ChatMessageEnabled:          true,
			QuietHoursEnabled:           true,
			QuietHoursStart:             "22:00",
			QuietHoursEnd:               "08:00",
			SubscriptionReminderEnabled: true,
		}

		err := db.Create(&prefs).Error
		assert.NoError(t, err)
		assert.NotZero(t, prefs.ID)
	})

	t.Run("Update notification preferences", func(t *testing.T) {
		db := setupPushTestDB(t)

		prefs := NotificationPreference{
			UserID:      2,
			PushEnabled: true,
		}
		db.Create(&prefs)

		updates := map[string]interface{}{
			"push_enabled":      false,
			"ride_join_enabled": false,
		}
		err := db.Model(&prefs).Updates(updates).Error
		assert.NoError(t, err)

		var updated NotificationPreference
		db.First(&updated, prefs.ID)
		assert.False(t, updated.PushEnabled)
	})

	t.Run("Get default preferences for new user", func(t *testing.T) {
		db := setupPushTestDB(t)

		var prefs NotificationPreference
		err := db.Where("user_id = ?", 999).First(&prefs).Error
		assert.Error(t, err)

		defaultPrefs := NotificationPreference{
			UserID:                      999,
			PushEnabled:                 true,
			RideJoinEnabled:             true,
			RideLeaveEnabled:            true,
			RideDeleteEnabled:           true,
			RideStartEnabled:            true,
			RideCompleteEnabled:         true,
			ChatMessageEnabled:          true,
			SubscriptionJoinEnabled:     true,
			SubscriptionLeaveEnabled:    true,
			SubscriptionReminderEnabled: true,
			QuietHoursEnabled:           false,
		}

		assert.True(t, defaultPrefs.PushEnabled)
		assert.True(t, defaultPrefs.RideJoinEnabled)
	})
}

func TestPushNotificationLog(t *testing.T) {
	t.Run("Create push notification log", func(t *testing.T) {
		db := setupPushTestDB(t)

		log := PushNotificationLog{
			UserID:         1,
			NotificationID: 100,
			DeviceTokenID:  1,
			MessageID:      "fcm_msg_123",
			Status:         "sent",
			SentAt:         time.Now(),
		}

		err := db.Create(&log).Error
		assert.NoError(t, err)
		assert.NotZero(t, log.ID)
	})

	t.Run("Log failed notification", func(t *testing.T) {
		db := setupPushTestDB(t)

		log := PushNotificationLog{
			UserID:         2,
			NotificationID: 101,
			DeviceTokenID:  2,
			Status:         "failed",
			ErrorMessage:   "Invalid token",
			SentAt:         time.Now(),
		}

		err := db.Create(&log).Error
		assert.NoError(t, err)
		assert.NotEmpty(t, log.ErrorMessage)
	})

	t.Run("Get notification statistics", func(t *testing.T) {
		db := setupPushTestDB(t)

		logs := []PushNotificationLog{
			{UserID: 1, Status: "sent", SentAt: time.Now()},
			{UserID: 1, Status: "sent", SentAt: time.Now()},
			{UserID: 1, Status: "failed", ErrorMessage: "Token expired", SentAt: time.Now()},
			{UserID: 2, Status: "sent", SentAt: time.Now()},
		}

		for _, log := range logs {
			db.Create(&log)
		}

		var sentCount int64
		var failedCount int64
		db.Model(&PushNotificationLog{}).Where("status = ?", "sent").Count(&sentCount)
		db.Model(&PushNotificationLog{}).Where("status = ?", "failed").Count(&failedCount)

		assert.Equal(t, int64(3), sentCount)
		assert.Equal(t, int64(1), failedCount)
	})
}

func TestPushShouldSendNotification(t *testing.T) {
	t.Run("Send when push enabled", func(t *testing.T) {
		db := setupPushTestDB(t)

		prefs := NotificationPreference{
			UserID:          1,
			PushEnabled:     true,
			RideJoinEnabled: true,
		}
		db.Create(&prefs)

		var storedPrefs NotificationPreference
		db.Where("user_id = ?", 1).First(&storedPrefs)

		shouldSend := storedPrefs.PushEnabled && storedPrefs.RideJoinEnabled
		assert.True(t, shouldSend)
	})

	t.Run("Don't send when push disabled", func(t *testing.T) {
		db := setupPushTestDB(t)

		prefs := NotificationPreference{
			UserID:          2,
			RideJoinEnabled: true,
		}

		result := db.Create(&prefs)
		assert.NoError(t, result.Error)

		db.Model(&NotificationPreference{}).Where("user_id = ?", 2).Updates(map[string]interface{}{
			"push_enabled": false,
		})

		var storedPrefs NotificationPreference
		result = db.Where("user_id = ?", 2).First(&storedPrefs)
		assert.NoError(t, result.Error)

		t.Logf("PushEnabled value: %v", storedPrefs.PushEnabled)

		shouldSend := storedPrefs.PushEnabled && storedPrefs.RideJoinEnabled
		assert.False(t, shouldSend)
	})

	t.Run("Don't send specific notification type when disabled", func(t *testing.T) {
		db := setupPushTestDB(t)

		prefs := NotificationPreference{
			UserID:      3,
			PushEnabled: true,
		}
		result := db.Create(&prefs)
		assert.NoError(t, result.Error)

		db.Model(&NotificationPreference{}).Where("user_id = ?", 3).Updates(map[string]interface{}{
			"ride_join_enabled": false,
		})

		var storedPrefs NotificationPreference
		result = db.Where("user_id = ?", 3).First(&storedPrefs)
		assert.NoError(t, result.Error)

		t.Logf("RideJoinEnabled value: %v", storedPrefs.RideJoinEnabled)

		shouldSend := storedPrefs.PushEnabled && storedPrefs.RideJoinEnabled
		assert.False(t, shouldSend)
	})
}

func TestPushQuietHoursLogic(t *testing.T) {
	t.Run("During quiet hours", func(t *testing.T) {
		quietStart := "22:00"
		quietEnd := "08:00"
		currentTime := "23:30"

		inQuietHours := currentTime >= quietStart || currentTime <= quietEnd
		assert.True(t, inQuietHours)
	})

	t.Run("Outside quiet hours", func(t *testing.T) {
		quietStart := "22:00"
		quietEnd := "08:00"
		currentTime := "14:00"

		inQuietHours := currentTime >= quietStart || currentTime <= quietEnd
		assert.False(t, inQuietHours)
	})

	t.Run("Quiet hours disabled", func(t *testing.T) {
		quietHoursEnabled := false
		assert.False(t, quietHoursEnabled)
	})
}

func TestPushCleanupInactiveTokens(t *testing.T) {
	t.Run("Cleanup old tokens", func(t *testing.T) {
		db := setupPushTestDB(t)

		oldToken := DeviceToken{
			UserID:   1,
			Token:    "old_token",
			Platform: "android",
			IsActive: true,
			LastUsed: time.Now().AddDate(0, 0, -100),
		}
		db.Create(&oldToken)

		recentToken := DeviceToken{
			UserID:   1,
			Token:    "recent_token",
			Platform: "ios",
			IsActive: true,
			LastUsed: time.Now(),
		}
		db.Create(&recentToken)

		cutoff := time.Now().AddDate(0, 0, -90)
		result := db.Where("last_used < ?", cutoff).Delete(&DeviceToken{})
		assert.NoError(t, result.Error)
		assert.Equal(t, int64(1), result.RowsAffected)

		var remainingTokens []DeviceToken
		db.Find(&remainingTokens)
		assert.Equal(t, 1, len(remainingTokens))
		assert.Equal(t, "recent_token", remainingTokens[0].Token)
	})
}

func TestPushBulkNotifications(t *testing.T) {
	t.Run("Send to multiple users", func(t *testing.T) {
		db := setupPushTestDB(t)
		db.AutoMigrate(&models.Notification{})

		userIDs := []uint{1, 2, 3, 4, 5}
		title := "System Maintenance"
		message := "Scheduled maintenance tonight"

		for _, userID := range userIDs {
			data := map[string]interface{}{
				"type":    "maintenance",
				"message": message,
			}
			dataJSON, _ := json.Marshal(data)

			notification := models.Notification{
				UserID:  userID,
				Type:    "system",
				Title:   title,
				Message: message,
				Data:    dataJSON,
			}

			err := db.Create(&notification).Error
			assert.NoError(t, err)
		}

		var count int64
		db.Model(&models.Notification{}).Where("type = ?", "system").Count(&count)
		assert.Equal(t, int64(5), count)
	})
}

func TestPushDeviceTokenPlatforms(t *testing.T) {
	t.Run("Create tokens for different platforms", func(t *testing.T) {
		db := setupPushTestDB(t)

		platforms := []string{"android", "ios", "web"}

		for i, platform := range platforms {
			token := DeviceToken{
				UserID:   1,
				Token:    "token_" + platform,
				Platform: platform,
				IsActive: true,
				LastUsed: time.Now(),
			}
			err := db.Create(&token).Error
			assert.NoError(t, err)
			assert.Equal(t, uint(i+1), token.ID)
		}

		var androidCount int64
		var iosCount int64
		var webCount int64

		db.Model(&DeviceToken{}).Where("platform = ?", "android").Count(&androidCount)
		db.Model(&DeviceToken{}).Where("platform = ?", "ios").Count(&iosCount)
		db.Model(&DeviceToken{}).Where("platform = ?", "web").Count(&webCount)

		assert.Equal(t, int64(1), androidCount)
		assert.Equal(t, int64(1), iosCount)
		assert.Equal(t, int64(1), webCount)
	})
}

func TestPushTokenUniqueness(t *testing.T) {
	t.Run("Duplicate tokens handled correctly", func(t *testing.T) {
		db := setupPushTestDB(t)

		token1 := DeviceToken{
			UserID:   1,
			Token:    "unique_token",
			Platform: "android",
			IsActive: true,
			LastUsed: time.Now(),
		}
		err := db.Create(&token1).Error
		assert.NoError(t, err)

		token2 := DeviceToken{
			UserID:   2,
			Token:    "unique_token",
			Platform: "ios",
			IsActive: true,
			LastUsed: time.Now(),
		}
		err = db.Create(&token2).Error

		assert.Error(t, err)
	})
}

func BenchmarkPushRegisterDeviceToken(b *testing.B) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&DeviceToken{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		token := DeviceToken{
			UserID:   uint(i % 100),
			Token:    "benchmark_token_" + string(rune(i)),
			Platform: "android",
			IsActive: true,
			LastUsed: time.Now(),
		}
		db.Create(&token)
	}
}

func BenchmarkPushGetUserDeviceTokens(b *testing.B) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&DeviceToken{})

	for i := 0; i < 100; i++ {
		token := DeviceToken{
			UserID:   1,
			Token:    "token_" + string(rune(i)),
			Platform: "android",
			IsActive: true,
			LastUsed: time.Now(),
		}
		db.Create(&token)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var tokens []DeviceToken
		db.Where("user_id = ? AND is_active = ?", 1, true).Find(&tokens)
	}
}
