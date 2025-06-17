package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

type FCMSvc struct {
	db          *sql.DB
	client      *messaging.Client
	firebaseApp *firebase.App
}

type FCMTokenReq struct {
	FCMToken string `json:"fcm_token" binding:"required"`
	DeviceID string `json:"device_id"`
	Platform string `json:"platform" binding:"required,oneof=android ios web"`
}

type FCMNotificationData struct {
	RideID     string `json:"ride_id,omitempty"`
	UserID     string `json:"user_id,omitempty"`
	UserName   string `json:"user_name,omitempty"`
	ActionType string `json:"action_type,omitempty"`
	Type       string `json:"type,omitempty"`
}

func NewFCMSvc(db *sql.DB, firebaseCredentialsPath string) (*FCMSvc, error) {
	if firebaseCredentialsPath == "" {
		log.Println("Firebase credentials path not provided, FCM service will be disabled")
		return &FCMSvc{db: db}, nil
	}

	ctx := context.Background()
	opt := option.WithCredentialsFile(firebaseCredentialsPath)
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		return nil, fmt.Errorf("error initializing firebase app: %v", err)
	}

	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting messaging client: %v", err)
	}

	return &FCMSvc{
		db:          db,
		client:      client,
		firebaseApp: app,
	}, nil
}

func (f *FCMSvc) SaveFCMToken(userID int, req FCMTokenReq) error {
	query := `
		INSERT INTO user_fcm_tokens (user_id, fcm_token, device_id, platform, is_active)
		VALUES ($1, $2, $3, $4, TRUE)
		ON CONFLICT (user_id, device_id)
		DO UPDATE SET 
			fcm_token = EXCLUDED.fcm_token,
			platform = EXCLUDED.platform,
			is_active = TRUE,
			updated_at = NOW()
	`

	_, err := f.db.Exec(query, userID, req.FCMToken, req.DeviceID, req.Platform)
	return err
}

func (f *FCMSvc) GetUserFCMTokens(userID int) ([]string, error) {
	query := `
		SELECT fcm_token 
		FROM user_fcm_tokens 
		WHERE user_id = $1 AND is_active = TRUE
	`

	rows, err := f.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []string
	for rows.Next() {
		var token string
		if err := rows.Scan(&token); err != nil {
			continue
		}
		tokens = append(tokens, token)
	}

	return tokens, nil
}

func (f *FCMSvc) SendNotification(userIDs []int, title, body string, data FCMNotificationData) error {
	if f.client == nil {
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

	message := &messaging.MulticastMessage{
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data:   dataMap,
		Tokens: allTokens,
		Android: &messaging.AndroidConfig{
			Notification: &messaging.AndroidNotification{
				Priority: messaging.PriorityHigh,
				Sound:    "default",
			},
		},
		APNS: &messaging.APNSConfig{
			Payload: &messaging.APNSPayload{
				Aps: &messaging.Aps{
					Alert: &messaging.ApsAlert{
						Title: title,
						Body:  body,
					},
					Sound: "default",
				},
			},
		},
		Webpush: &messaging.WebpushConfig{
			Notification: &messaging.WebpushNotification{
				Title: title,
				Body:  body,
				Icon:  "/icon-192x192.png",
			},
		},
	}

	ctx := context.Background()
	response, err := f.client.SendMulticast(ctx, message)
	if err != nil {
		return fmt.Errorf("error sending FCM message: %v", err)
	}

	log.Printf("FCM notification sent. Success: %d, Failure: %d", response.SuccessCount, response.FailureCount)

	if response.FailureCount > 0 {
		f.handleFailedTokens(allTokens, response.Responses)
	}

	return nil
}

func (f *FCMSvc) SendToUser(userID int, title, body string, data FCMNotificationData) error {
	return f.SendNotification([]int{userID}, title, body, data)
}

func (f *FCMSvc) handleFailedTokens(tokens []string, responses []*messaging.SendResponse) {
	for i, response := range responses {
		if response.Error != nil {
			if messaging.IsInvalidArgument(response.Error) ||
				messaging.IsUnregistered(response.Error) ||
				messaging.IsRegistrationTokenNotRegistered(response.Error) {
				if i < len(tokens) {
					f.removeInvalidToken(tokens[i])
				}
			}
		}
	}
}

func (f *FCMSvc) removeInvalidToken(token string) {
	query := `UPDATE user_fcm_tokens SET is_active = FALSE WHERE fcm_token = $1`
	_, err := f.db.Exec(query, token)
	if err != nil {
		log.Printf("Error removing invalid FCM token: %v", err)
	}
}

func (f *FCMSvc) RemoveUserToken(userID int, deviceID string) error {
	query := `UPDATE user_fcm_tokens SET is_active = FALSE WHERE user_id = $1 AND device_id = $2`
	_, err := f.db.Exec(query, userID, deviceID)
	return err
}
