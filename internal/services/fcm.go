package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type FCMSvc struct {
	db        *gorm.DB
	projectID string
	client    *http.Client
}

type FCMTokenReq struct {
	FCMToken string `json:"fcm_token" binding:"required"`
	DeviceID string `json:"device_id"`
	Platform string `json:"platform" binding:"required,oneof=android ios web"`
}

type UserFCMToken struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    int       `gorm:"not null;index" json:"user_id"`
	FCMToken  string    `gorm:"not null" json:"fcm_token"`
	DeviceID  string    `gorm:"not null" json:"device_id"`
	Platform  string    `gorm:"not null" json:"platform"`
	IsActive  bool      `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (UserFCMToken) TableName() string {
	return "user_fcm_tokens"
}

type FCMNotificationData struct {
	RideID     string `json:"ride_id,omitempty"`
	UserID     string `json:"user_id,omitempty"`
	UserName   string `json:"user_name,omitempty"`
	ActionType string `json:"action_type,omitempty"`
	Type       string `json:"type,omitempty"`
}

type FCMMessage struct {
	Message FCMMessagePayload `json:"message"`
}

type FCMMessagePayload struct {
	Token        string            `json:"token,omitempty"`
	Tokens       []string          `json:"tokens,omitempty"`
	Notification *FCMNotification  `json:"notification,omitempty"`
	Data         map[string]string `json:"data,omitempty"`
	Android      *FCMAndroidConfig `json:"android,omitempty"`
	APNS         *FCMAPNSConfig    `json:"apns,omitempty"`
	Webpush      *FCMWebpushConfig `json:"webpush,omitempty"`
}

type FCMNotification struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

type FCMAndroidConfig struct {
	Priority     string                  `json:"priority,omitempty"`
	Notification *FCMAndroidNotification `json:"notification,omitempty"`
}

type FCMAndroidNotification struct {
	Sound                string `json:"sound,omitempty"`
	ChannelID            string `json:"channel_id,omitempty"`
	Icon                 string `json:"icon,omitempty"`
	Color                string `json:"color,omitempty"`
	NotificationPriority string `json:"notification_priority,omitempty"`
}

type FCMAPNSConfig struct {
	Payload *FCMAPNSPayload `json:"payload,omitempty"`
}

type FCMAPNSPayload struct {
	Aps *FCMAps `json:"aps,omitempty"`
}

type FCMAps struct {
	Alert *FCMAlert `json:"alert,omitempty"`
	Sound string    `json:"sound,omitempty"`
}

type FCMAlert struct {
	Title string `json:"title,omitempty"`
	Body  string `json:"body,omitempty"`
}

type FCMWebpushConfig struct {
	Notification *FCMWebpushNotification `json:"notification,omitempty"`
}

type FCMWebpushNotification struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Icon  string `json:"icon,omitempty"`
}

type FCMResponse struct {
	Name  string `json:"name,omitempty"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error,omitempty"`
}

func NewFCMSvc(db *gorm.DB, firebaseCredentialsPath, projectID string) (*FCMSvc, error) {
	if firebaseCredentialsPath == "" || projectID == "" {
		log.Println("Firebase credentials or project ID not provided, FCM service will be disabled")
		return &FCMSvc{db: db}, nil
	}

	ctx := context.Background()

	creds, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		credBytes, fileErr := os.ReadFile(firebaseCredentialsPath)
		if fileErr != nil {
			return nil, fmt.Errorf("error reading credentials file: %v (default: %v, file: %v)", err, err, fileErr)
		}

		creds, fileErr = google.CredentialsFromJSON(ctx, credBytes, "https://www.googleapis.com/auth/cloud-platform")
		if fileErr != nil {
			return nil, fmt.Errorf("error loading credentials from JSON: %v", fileErr)
		}
	}

	client := oauth2.NewClient(ctx, creds.TokenSource)

	return &FCMSvc{
		db:        db,
		projectID: projectID,
		client:    client,
	}, nil
}

func (f *FCMSvc) SaveFCMToken(userID int, req FCMTokenReq) error {
	if req.FCMToken == "" {
		log.Printf("Empty FCM token provided for user %d, skipping save", userID)
		return nil
	}

	token := UserFCMToken{
		UserID:   userID,
		FCMToken: req.FCMToken,
		DeviceID: req.DeviceID,
		Platform: req.Platform,
		IsActive: true,
	}

	return f.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "device_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"fcm_token":  req.FCMToken,
			"platform":   req.Platform,
			"is_active":  true,
			"updated_at": time.Now(),
		}),
	}).Create(&token).Error
}

func (f *FCMSvc) GetUserFCMTokens(userID int) ([]string, error) {
	var tokens []UserFCMToken
	err := f.db.Where("user_id = ? AND is_active = ?", userID, true).
		Select("fcm_token").
		Find(&tokens).Error

	if err != nil {
		return nil, err
	}

	var tokenStrings []string
	for _, token := range tokens {
		tokenStrings = append(tokenStrings, token.FCMToken)
	}

	return tokenStrings, nil
}

func (f *FCMSvc) SendNotification(userIDs []int, title, body string, data FCMNotificationData) error {
	if f.client == nil || f.projectID == "" {
		log.Println("FCM client not initialized, skipping notification")
		return nil
	}

	var allTokens []string
	for _, userID := range userIDs {
		tokens, err := f.GetUserFCMTokens(userID)
		if err != nil {
			log.Printf("Error getting FCM tokens for user %d: %v", userID, err)
			continue
		}
		allTokens = append(allTokens, tokens...)
	}

	if len(allTokens) == 0 {
		log.Println("No FCM tokens found for users")
		return nil
	}

	dataMap := make(map[string]string)
	dataBytes, _ := json.Marshal(data)
	var dataInterface map[string]interface{}
	json.Unmarshal(dataBytes, &dataInterface)

	for k, v := range dataInterface {
		if str, ok := v.(string); ok {
			dataMap[k] = str
		} else {
			dataMap[k] = fmt.Sprintf("%v", v)
		}
	}

	var successCount, failureCount int
	for _, token := range allTokens {
		if err := f.sendToToken(token, title, body, dataMap); err != nil {
			log.Printf("Failed to send notification to token: %v", err)
			failureCount++
			f.handleFailedToken(token, err)
		} else {
			successCount++
		}
	}

	log.Printf("FCM notification sent. Success: %d, Failure: %d", successCount, failureCount)
	return nil
}

func (f *FCMSvc) sendToToken(token, title, body string, data map[string]string) error {
	message := FCMMessage{
		Message: FCMMessagePayload{
			Token: token,
			Notification: &FCMNotification{
				Title: title,
				Body:  body,
			},
			Data: data,
			Android: &FCMAndroidConfig{
				Priority: "high",
				Notification: &FCMAndroidNotification{
					Sound:                "default",
					NotificationPriority: "PRIORITY_HIGH",
				},
			},
			APNS: &FCMAPNSConfig{
				Payload: &FCMAPNSPayload{
					Aps: &FCMAps{
						Alert: &FCMAlert{
							Title: title,
							Body:  body,
						},
						Sound: "default",
					},
				},
			},
			Webpush: &FCMWebpushConfig{
				Notification: &FCMWebpushNotification{
					Title: title,
					Body:  body,
					Icon:  "/icon-192x192.png",
				},
			},
		},
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("error marshaling message: %v", err)
	}

	url := fmt.Sprintf("https://fcm.googleapis.com/v1/projects/%s/messages:send", f.projectID)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := f.client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var fcmResp FCMResponse
		json.NewDecoder(resp.Body).Decode(&fcmResp)
		if fcmResp.Error != nil {
			return fmt.Errorf("FCM API error: %d - %s", resp.StatusCode, fcmResp.Error.Message)
		}
		return fmt.Errorf("FCM API error: %d", resp.StatusCode)
	}

	return nil
}

func (f *FCMSvc) SendToUser(userID int, title, body string, data FCMNotificationData) error {
	return f.SendNotification([]int{userID}, title, body, data)
}

func (f *FCMSvc) handleFailedToken(token string, err error) {
	if err != nil {
		log.Printf("Token may be invalid: %s, error: %v", token, err)
		f.removeInvalidToken(token)
	}
}

func (f *FCMSvc) removeInvalidToken(token string) {
	err := f.db.Model(&UserFCMToken{}).
		Where("fcm_token = ?", token).
		Update("is_active", false).Error

	if err != nil {
		log.Printf("Error removing invalid FCM token: %v", err)
	}
}

func (f *FCMSvc) RemoveUserToken(userID int, deviceID string) error {
	return f.db.Model(&UserFCMToken{}).
		Where("user_id = ? AND device_id = ?", userID, deviceID).
		Update("is_active", false).Error
}
