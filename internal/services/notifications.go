package services

import (
	"encoding/json"
	"fmt"
	"strconv"

	"mride-backend/internal/models"

	"gorm.io/gorm"
)

type NotificationSvc struct {
	db           *gorm.DB
	fcmSvc       *FCMSvc
	webSocketSvc *WebSocketSvc
}

func NewNotificationSvc(db *gorm.DB, fcmSvc *FCMSvc, webSocketSvc *WebSocketSvc) *NotificationSvc {
	return &NotificationSvc{
		db:           db,
		fcmSvc:       fcmSvc,
		webSocketSvc: webSocketSvc,
	}
}

func (n *NotificationSvc) CreateRideJoinNotification(driverID, rideID, passengerID uint, passengerName string) error {
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

	notification := models.Notification{
		UserID:  driverID,
		Type:    models.NotificationTypeRideJoin,
		Title:   title,
		Message: message,
		Data:    dataJSON,
		Read:    false,
	}

	if err := n.db.Create(&notification).Error; err != nil {
		return err
	}

	if n.webSocketSvc != nil && n.webSocketSvc.IsUserOnline(int(driverID)) {
		wsMessage := WSMessage{
			Type:    "notification",
			Title:   title,
			Message: message,
			Data:    data,
		}

		if err := n.webSocketSvc.SendToUser(int(driverID), wsMessage); err != nil {
			fmt.Printf("Failed to send WebSocket notification: %v\n", err)
		} else {
			fmt.Printf("WebSocket notification sent to user %d\n", driverID)
		}
	} else if n.fcmSvc != nil {
		fcmData := FCMNotificationData{
			RideID:     strconv.FormatUint(uint64(rideID), 10),
			UserID:     strconv.FormatUint(uint64(passengerID), 10),
			UserName:   passengerName,
			ActionType: "join",
			Type:       models.NotificationTypeRideJoin,
		}

		go func() {
			if err := n.fcmSvc.SendToUser(int(driverID), title, message, fcmData); err != nil {
				fmt.Printf("Failed to send FCM notification: %v\n", err)
			}
		}()
	}

	return nil
}

func (n *NotificationSvc) CreateRideLeaveNotification(driverID, rideID, passengerID uint, passengerName string) error {
	title := "Passenger Left"
	message := fmt.Sprintf("%s left your ride", passengerName)

	data := models.NotificationData{
		RideID:     rideID,
		UserID:     passengerID,
		UserName:   passengerName,
		ActionType: "leave",
	}

	dataJSON, err := json.Marshal(data)
	if err != nil {
		return err
	}

	notification := models.Notification{
		UserID:  driverID,
		Type:    models.NotificationTypeRideLeave,
		Title:   title,
		Message: message,
		Data:    dataJSON,
		Read:    false,
	}

	if err := n.db.Create(&notification).Error; err != nil {
		return err
	}

	if n.webSocketSvc != nil && n.webSocketSvc.IsUserOnline(int(driverID)) {
		wsMessage := WSMessage{
			Type:    "notification",
			Title:   title,
			Message: message,
			Data:    data,
		}

		if err := n.webSocketSvc.SendToUser(int(driverID), wsMessage); err != nil {
			fmt.Printf("Failed to send WebSocket notification: %v\n", err)
		}
	} else if n.fcmSvc != nil {
		fcmData := FCMNotificationData{
			RideID:     strconv.FormatUint(uint64(rideID), 10),
			UserID:     strconv.FormatUint(uint64(passengerID), 10),
			UserName:   passengerName,
			ActionType: "leave",
			Type:       models.NotificationTypeRideLeave,
		}

		go func() {
			if err := n.fcmSvc.SendToUser(int(driverID), title, message, fcmData); err != nil {
				fmt.Printf("Failed to send FCM notification: %v\n", err)
			}
		}()
	}

	return nil
}

func (n *NotificationSvc) CreateRideDeletedNotification(passengerID, rideID, driverID uint) error {
	var user models.User
	var driverName string = "Driver"

	if err := n.db.Select("full_name").First(&user, driverID).Error; err == nil {
		driverName = user.FullName
	}

	title := "Ride Cancelled"
	message := fmt.Sprintf("The ride you joined has been cancelled by %s", driverName)

	data := models.NotificationData{
		RideID:     rideID,
		UserID:     driverID,
		UserName:   driverName,
		ActionType: "delete",
	}

	dataJSON, err := json.Marshal(data)
	if err != nil {
		return err
	}

	notification := models.Notification{
		UserID:  passengerID,
		Type:    models.NotificationTypeRideDelete,
		Title:   title,
		Message: message,
		Data:    dataJSON,
		Read:    false,
	}

	if err := n.db.Create(&notification).Error; err != nil {
		return err
	}

	if n.webSocketSvc != nil && n.webSocketSvc.IsUserOnline(int(passengerID)) {
		wsMessage := WSMessage{
			Type:    "notification",
			Title:   title,
			Message: message,
			Data:    data,
		}

		if err := n.webSocketSvc.SendToUser(int(passengerID), wsMessage); err != nil {
			fmt.Printf("Failed to send WebSocket notification: %v\n", err)
		}
	} else if n.fcmSvc != nil {
		fcmData := FCMNotificationData{
			RideID:     strconv.FormatUint(uint64(rideID), 10),
			UserID:     strconv.FormatUint(uint64(driverID), 10),
			UserName:   driverName,
			ActionType: "delete",
			Type:       models.NotificationTypeRideDelete,
		}

		go func() {
			if err := n.fcmSvc.SendToUser(int(passengerID), title, message, fcmData); err != nil {
				fmt.Printf("Failed to send FCM notification: %v\n", err)
			}
		}()
	}

	return nil
}

func (n *NotificationSvc) GetUserNotifications(userID uint, limit, offset int) (*models.NotificationsResp, error) {
	if limit <= 0 {
		limit = 20
	}

	var notifications []models.Notification
	if err := n.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&notifications).Error; err != nil {
		return nil, err
	}

	var unreadCount int64
	if err := n.db.Model(&models.Notification{}).
		Where("user_id = ? AND read = ?", userID, false).
		Count(&unreadCount).Error; err != nil {
		unreadCount = 0
	}

	return &models.NotificationsResp{
		Notifications: notifications,
		UnreadCount:   int(unreadCount),
	}, nil
}

func (n *NotificationSvc) MarkAsRead(userID, notificationID uint) error {
	return n.db.Model(&models.Notification{}).
		Where("id = ? AND user_id = ?", notificationID, userID).
		Update("read", true).Error
}

func (n *NotificationSvc) MarkAllAsRead(userID uint) error {
	return n.db.Model(&models.Notification{}).
		Where("user_id = ? AND read = ?", userID, false).
		Update("read", true).Error
}

func (n *NotificationSvc) GetUnreadCount(userID uint) (int, error) {
	var count int64
	err := n.db.Model(&models.Notification{}).
		Where("user_id = ? AND read = ?", userID, false).
		Count(&count).Error
	return int(count), err
}

func (n *NotificationSvc) SaveFCMToken(userID uint, req FCMTokenReq) error {
	if n.fcmSvc == nil {
		return fmt.Errorf("FCM service not available")
	}
	return n.fcmSvc.SaveFCMToken(int(userID), req)
}

func (n *NotificationSvc) RemoveFCMToken(userID uint, deviceID string) error {
	if n.fcmSvc == nil {
		return fmt.Errorf("FCM service not available")
	}
	return n.fcmSvc.RemoveUserToken(int(userID), deviceID)
}
