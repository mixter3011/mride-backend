package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"

	"mride-backend/internal/models"
)

type NotificationSvc struct {
	db     *sql.DB
	fcmSvc *FCMSvc
}

func NewNotificationSvc(db *sql.DB, fcmSvc *FCMSvc) *NotificationSvc {
	return &NotificationSvc{
		db:     db,
		fcmSvc: fcmSvc,
	}
}

func (n *NotificationSvc) CreateRideJoinNotification(driverID, rideID, passengerID int, passengerName string) error {
	title := "New Passenger Joined"
	message := fmt.Sprintf("%s joined your ride", passengerName)

	data := models.NotificationData{
		RideID:     rideID,
		UserID:     passengerID,
		UserName:   passengerName,
		ActionType: "join",
	}

	dataJSON, err := json.Marshal(data)
	if err != nil {
		return err
	}

	query := `INSERT INTO notifications (user_id, type, title, message, data) 
			  VALUES ($1, $2, $3, $4, $5)`

	_, err = n.db.Exec(query, driverID, models.NotificationTypeRideJoin, title, message, dataJSON)
	if err != nil {
		return err
	}

	if n.fcmSvc != nil {
		fcmData := FCMNotificationData{
			RideID:     strconv.Itoa(rideID),
			UserID:     strconv.Itoa(passengerID),
			UserName:   passengerName,
			ActionType: "join",
			Type:       models.NotificationTypeRideJoin,
		}

		go func() {
			if err := n.fcmSvc.SendToUser(driverID, title, message, fcmData); err != nil {
				fmt.Printf("Failed to send FCM notification: %v\n", err)
			}
		}()
	}

	return nil
}

func (n *NotificationSvc) GetUserNotifications(userID int, limit, offset int) (*models.NotificationsResp, error) {
	if limit <= 0 {
		limit = 20
	}

	query := `SELECT id, user_id, type, title, message, data, read, created_at, updated_at 
			  FROM notifications 
			  WHERE user_id = $1 
			  ORDER BY created_at DESC 
			  LIMIT $2 OFFSET $3`

	rows, err := n.db.Query(query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []models.Notification
	for rows.Next() {
		var notification models.Notification
		err := rows.Scan(
			&notification.ID, &notification.UserID, &notification.Type,
			&notification.Title, &notification.Message, &notification.Data,
			&notification.Read, &notification.CreatedAt, &notification.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		notifications = append(notifications, notification)
	}

	var unreadCount int
	countQuery := `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read = FALSE`
	err = n.db.QueryRow(countQuery, userID).Scan(&unreadCount)
	if err != nil {
		unreadCount = 0
	}

	return &models.NotificationsResp{
		Notifications: notifications,
		UnreadCount:   unreadCount,
	}, nil
}

func (n *NotificationSvc) MarkAsRead(userID, notificationID int) error {
	query := `UPDATE notifications SET read = TRUE, updated_at = NOW() 
			  WHERE id = $1 AND user_id = $2`

	_, err := n.db.Exec(query, notificationID, userID)
	return err
}

func (n *NotificationSvc) MarkAllAsRead(userID int) error {
	query := `UPDATE notifications SET read = TRUE, updated_at = NOW() 
			  WHERE user_id = $1 AND read = FALSE`

	_, err := n.db.Exec(query, userID)
	return err
}

func (n *NotificationSvc) GetUnreadCount(userID int) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read = FALSE`
	err := n.db.QueryRow(query, userID).Scan(&count)
	return count, err
}

func (n *NotificationSvc) SaveFCMToken(userID int, req FCMTokenReq) error {
	if n.fcmSvc == nil {
		return fmt.Errorf("FCM service not available")
	}
	return n.fcmSvc.SaveFCMToken(userID, req)
}

func (n *NotificationSvc) RemoveFCMToken(userID int, deviceID string) error {
	if n.fcmSvc == nil {
		return fmt.Errorf("FCM service not available")
	}
	return n.fcmSvc.RemoveUserToken(userID, deviceID)
}
