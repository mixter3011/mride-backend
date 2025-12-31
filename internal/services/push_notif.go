package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
	"gorm.io/gorm"

	"mride-backend/internal/models"
)

type PushNotificationSvc struct {
	db              *gorm.DB
	fcmClient       *messaging.Client
	webSocketSvc    WebSocketInterface
	notificationSvc *NotificationSvc
}

type DeviceToken struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    uint      `gorm:"not null;index" json:"user_id"`
	Token     string    `gorm:"not null;uniqueIndex;size:500" json:"token"`
	Platform  string    `gorm:"not null;size:20" json:"platform"`
	IsActive  bool      `gorm:"default:true;index" json:"is_active"`
	LastUsed  time.Time `gorm:"not null" json:"last_used"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (DeviceToken) TableName() string {
	return "device_tokens"
}

type PushNotificationLog struct {
	ID             uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID         uint      `gorm:"not null;index" json:"user_id"`
	NotificationID uint      `gorm:"index" json:"notification_id"`
	DeviceTokenID  uint      `gorm:"index" json:"device_token_id"`
	MessageID      string    `gorm:"size:200" json:"message_id"`
	Status         string    `gorm:"not null;size:20;index" json:"status"`
	ErrorMessage   string    `gorm:"type:text" json:"error_message,omitempty"`
	SentAt         time.Time `gorm:"not null" json:"sent_at"`
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (PushNotificationLog) TableName() string {
	return "push_notification_logs"
}

type NotificationPreference struct {
	ID                          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID                      uint      `gorm:"not null;uniqueIndex" json:"user_id"`
	PushEnabled                 bool      `gorm:"default:true" json:"push_enabled"`
	RideJoinEnabled             bool      `gorm:"default:true" json:"ride_join_enabled"`
	RideLeaveEnabled            bool      `gorm:"default:true" json:"ride_leave_enabled"`
	RideDeleteEnabled           bool      `gorm:"default:true" json:"ride_delete_enabled"`
	RideStartEnabled            bool      `gorm:"default:true" json:"ride_start_enabled"`
	RideCompleteEnabled         bool      `gorm:"default:true" json:"ride_complete_enabled"`
	ChatMessageEnabled          bool      `gorm:"default:true" json:"chat_message_enabled"`
	SubscriptionJoinEnabled     bool      `gorm:"default:true" json:"subscription_join_enabled"`
	SubscriptionLeaveEnabled    bool      `gorm:"default:true" json:"subscription_leave_enabled"`
	SubscriptionReminderEnabled bool      `gorm:"default:true" json:"subscription_reminder_enabled"`
	QuietHoursEnabled           bool      `gorm:"default:false" json:"quiet_hours_enabled"`
	QuietHoursStart             string    `gorm:"size:5" json:"quiet_hours_start"`
	QuietHoursEnd               string    `gorm:"size:5" json:"quiet_hours_end"`
	CreatedAt                   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt                   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (NotificationPreference) TableName() string {
	return "notification_preferences"
}

func NewPushNotificationSvc(db *gorm.DB, firebaseCredentials string, webSocketSvc WebSocketInterface, notificationSvc *NotificationSvc) (*PushNotificationSvc, error) {

	if err := db.AutoMigrate(&DeviceToken{}, &PushNotificationLog{}, &NotificationPreference{}); err != nil {
		return nil, fmt.Errorf("failed to migrate push notification tables: %w", err)
	}

	opt := option.WithCredentialsJSON([]byte(firebaseCredentials))
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Firebase app: %w", err)
	}

	fcmClient, err := app.Messaging(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize FCM client: %w", err)
	}

	log.Println("Push notification service initialized successfully")

	return &PushNotificationSvc{
		db:              db,
		fcmClient:       fcmClient,
		webSocketSvc:    webSocketSvc,
		notificationSvc: notificationSvc,
	}, nil
}

func (p *PushNotificationSvc) RegisterDeviceToken(userID uint, token, platform string) error {
	now := time.Now()

	var existingToken DeviceToken
	err := p.db.Where("token = ?", token).First(&existingToken).Error

	if err == gorm.ErrRecordNotFound {

		deviceToken := DeviceToken{
			UserID:   userID,
			Token:    token,
			Platform: platform,
			IsActive: true,
			LastUsed: now,
		}
		return p.db.Create(&deviceToken).Error
	} else if err != nil {
		return err
	}

	return p.db.Model(&existingToken).Updates(map[string]interface{}{
		"user_id":   userID,
		"is_active": true,
		"last_used": now,
		"platform":  platform,
	}).Error
}

func (p *PushNotificationSvc) UnregisterDeviceToken(userID uint, token string) error {
	return p.db.Model(&DeviceToken{}).
		Where("user_id = ? AND token = ?", userID, token).
		Update("is_active", false).Error
}

func (p *PushNotificationSvc) GetUserDeviceTokens(userID uint) ([]DeviceToken, error) {
	var tokens []DeviceToken
	err := p.db.Where("user_id = ? AND is_active = ?", userID, true).Find(&tokens).Error
	return tokens, err
}

func (p *PushNotificationSvc) SendPushNotification(userID uint, notification *models.Notification) error {

	if !p.ShouldSendPushNotification(userID, notification.Type) {
		log.Printf("Push notifications disabled for user %d, type: %s", userID, notification.Type)
		return nil
	}

	if p.webSocketSvc != nil && p.webSocketSvc.IsUserOnline(int(userID)) {
		log.Printf("User %d is online via WebSocket, skipping push notification", userID)
		return nil
	}

	tokens, err := p.GetUserDeviceTokens(userID)
	if err != nil {
		return fmt.Errorf("failed to get device tokens: %w", err)
	}

	if len(tokens) == 0 {
		log.Printf("No device tokens found for user %d", userID)
		return nil
	}

	var notificationData map[string]interface{}
	if err := json.Unmarshal(notification.Data, &notificationData); err != nil {
		notificationData = make(map[string]interface{})
	}

	notificationData["notification_id"] = notification.ID

	dataMap := make(map[string]string)
	for key, value := range notificationData {
		dataMap[key] = fmt.Sprintf("%v", value)
	}

	for _, deviceToken := range tokens {
		go p.sendToDevice(deviceToken, notification, dataMap)
	}

	return nil
}

func (p *PushNotificationSvc) sendToDevice(deviceToken DeviceToken, notification *models.Notification, data map[string]string) {
	message := &messaging.Message{
		Token: deviceToken.Token,
		Notification: &messaging.Notification{
			Title: notification.Title,
			Body:  notification.Message,
		},
		Data: data,
		Android: &messaging.AndroidConfig{
			Priority: "high",
			Notification: &messaging.AndroidNotification{
				Sound:        "default",
				ChannelID:    "ride_notifications",
				Priority:     messaging.PriorityHigh,
				DefaultSound: true,
				Tag:          fmt.Sprintf("notification_%d", notification.ID),
			},
		},
		APNS: &messaging.APNSConfig{
			Headers: map[string]string{
				"apns-priority": "10",
			},
			Payload: &messaging.APNSPayload{
				Aps: &messaging.Aps{
					Sound:            "default",
					Badge:            nil,
					ContentAvailable: true,
					MutableContent:   true,
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	messageID, err := p.fcmClient.Send(ctx, message)

	logEntry := PushNotificationLog{
		UserID:         deviceToken.UserID,
		NotificationID: notification.ID,
		DeviceTokenID:  deviceToken.ID,
		MessageID:      messageID,
		SentAt:         time.Now(),
	}

	if err != nil {
		logEntry.Status = "failed"
		logEntry.ErrorMessage = err.Error()
		log.Printf("Failed to send push notification to device %d: %v", deviceToken.ID, err)

		if messaging.IsRegistrationTokenNotRegistered(err) || messaging.IsInvalidArgument(err) {
			p.db.Model(&deviceToken).Update("is_active", false)
			log.Printf("Deactivated invalid device token %d", deviceToken.ID)
		}
	} else {
		logEntry.Status = "sent"
		log.Printf("Push notification sent successfully to device %d, message ID: %s", deviceToken.ID, messageID)
	}

	if err := p.db.Create(&logEntry).Error; err != nil {
		log.Printf("Failed to create push notification log: %v", err)
	}
}

func (p *PushNotificationSvc) ShouldSendPushNotification(userID uint, notificationType string) bool {
	var pref NotificationPreference
	err := p.db.Where("user_id = ?", userID).First(&pref).Error

	if err == gorm.ErrRecordNotFound {

		return true
	} else if err != nil {
		log.Printf("Error fetching notification preferences: %v", err)
		return true
	}

	if !pref.PushEnabled {
		return false
	}

	if pref.QuietHoursEnabled && p.isInQuietHours(pref.QuietHoursStart, pref.QuietHoursEnd) {
		return false
	}

	switch notificationType {
	case models.NotificationTypeRideJoin:
		return pref.RideJoinEnabled
	case models.NotificationTypeRideLeave:
		return pref.RideLeaveEnabled
	case models.NotificationTypeRideDelete:
		return pref.RideDeleteEnabled
	case models.NotificationTypeRideStarted:
		return pref.RideStartEnabled
	case models.NotificationTypeRideCompleted:
		return pref.RideCompleteEnabled
	case "chat_message":
		return pref.ChatMessageEnabled
	case "subscription_join":
		return pref.SubscriptionJoinEnabled
	case "subscription_leave":
		return pref.SubscriptionLeaveEnabled
	case "subscription_ride_reminder":
		return pref.SubscriptionReminderEnabled
	default:
		return true
	}
}

func (p *PushNotificationSvc) isInQuietHours(start, end string) bool {
	if start == "" || end == "" {
		return false
	}

	now := time.Now()
	currentTime := now.Format("15:04")

	if start <= end {
		return currentTime >= start && currentTime <= end
	}
	return currentTime >= start || currentTime <= end
}

func (p *PushNotificationSvc) UpdateNotificationPreferences(userID uint, prefs *NotificationPreference) error {
	var existing NotificationPreference
	err := p.db.Where("user_id = ?", userID).First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		prefs.UserID = userID
		return p.db.Create(prefs).Error
	} else if err != nil {
		return err
	}

	return p.db.Model(&existing).Updates(prefs).Error
}

func (p *PushNotificationSvc) GetNotificationPreferences(userID uint) (*NotificationPreference, error) {
	var prefs NotificationPreference
	err := p.db.Where("user_id = ?", userID).First(&prefs).Error

	if err == gorm.ErrRecordNotFound {

		return &NotificationPreference{
			UserID:                      userID,
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
		}, nil
	}

	return &prefs, err
}

func (p *PushNotificationSvc) SendBulkNotification(userIDs []uint, title, message string, data map[string]interface{}) error {
	for _, userID := range userIDs {

		dataJSON, _ := json.Marshal(data)
		notification := models.Notification{
			UserID:  userID,
			Type:    "bulk",
			Title:   title,
			Message: message,
			Data:    dataJSON,
			Read:    false,
		}

		if err := p.db.Create(&notification).Error; err != nil {
			log.Printf("Failed to create notification for user %d: %v", userID, err)
			continue
		}

		go p.SendPushNotification(userID, &notification)
	}

	return nil
}

func (p *PushNotificationSvc) CleanupInactiveTokens() error {
	cutoff := time.Now().AddDate(0, 0, -90)
	result := p.db.Where("last_used < ?", cutoff).Delete(&DeviceToken{})

	if result.Error != nil {
		return result.Error
	}

	log.Printf("Cleaned up %d inactive device tokens", result.RowsAffected)
	return nil
}

func (p *PushNotificationSvc) GetPushNotificationStats() (map[string]interface{}, error) {
	var totalTokens int64
	var activeTokens int64
	var totalSent int64
	var totalFailed int64

	p.db.Model(&DeviceToken{}).Count(&totalTokens)
	p.db.Model(&DeviceToken{}).Where("is_active = ?", true).Count(&activeTokens)
	p.db.Model(&PushNotificationLog{}).Where("status = ?", "sent").Count(&totalSent)
	p.db.Model(&PushNotificationLog{}).Where("status = ?", "failed").Count(&totalFailed)

	return map[string]interface{}{
		"total_tokens":  totalTokens,
		"active_tokens": activeTokens,
		"total_sent":    totalSent,
		"total_failed":  totalFailed,
		"success_rate":  float64(totalSent) / float64(totalSent+totalFailed) * 100,
	}, nil
}
