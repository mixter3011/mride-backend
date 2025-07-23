package services

import (
	"encoding/json"
	"fmt"

	"mride-backend/internal/models"

	"gorm.io/gorm"
)

type NotificationSvc struct {
	db           *gorm.DB
	webSocketSvc *WebSocketSvc
}

func NewNotificationSvc(db *gorm.DB, webSocketSvc *WebSocketSvc) *NotificationSvc {
	return &NotificationSvc{
		db:           db,
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

	return n.createAndSendNotification(driverID, models.NotificationTypeRideJoin, title, message, data)
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

	return n.createAndSendNotification(driverID, models.NotificationTypeRideLeave, title, message, data)
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

	return n.createAndSendNotification(passengerID, models.NotificationTypeRideDelete, title, message, data)
}

func (n *NotificationSvc) CreateRideStartedNotification(passengerID, rideID, driverID uint) error {
	var user models.User
	var driverName string = "Driver"

	if err := n.db.Select("full_name").First(&user, driverID).Error; err == nil {
		driverName = user.FullName
	}

	title := "Ride Started"
	message := fmt.Sprintf("Your ride with %s has started", driverName)

	data := models.NotificationData{
		RideID:     rideID,
		UserID:     driverID,
		UserName:   driverName,
		ActionType: "started",
	}

	return n.createAndSendNotification(passengerID, models.NotificationTypeRideStarted, title, message, data)
}

func (n *NotificationSvc) CreateRideCompletedNotification(passengerID, rideID, driverID uint) error {
	var user models.User
	var driverName string = "Driver"

	if err := n.db.Select("full_name").First(&user, driverID).Error; err == nil {
		driverName = user.FullName
	}

	title := "Ride Completed"
	message := fmt.Sprintf("Your ride with %s has been completed", driverName)

	data := models.NotificationData{
		RideID:     rideID,
		UserID:     driverID,
		UserName:   driverName,
		ActionType: "completed",
	}

	return n.createAndSendNotification(passengerID, models.NotificationTypeRideCompleted, title, message, data)
}

func (n *NotificationSvc) createAndSendNotification(userID uint, notificationType, title, message string, data models.NotificationData) error {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return err
	}

	notification := models.Notification{
		UserID:  userID,
		Type:    notificationType,
		Title:   title,
		Message: message,
		Data:    dataJSON,
		Read:    false,
	}

	if err := n.db.Create(&notification).Error; err != nil {
		return err
	}

	if n.webSocketSvc != nil && n.webSocketSvc.IsUserOnline(int(userID)) {
		wsMessage := WSMessage{
			Type:    "notification",
			Title:   title,
			Message: message,
			Data:    data,
		}

		if err := n.webSocketSvc.SendToUser(int(userID), wsMessage); err != nil {
			fmt.Printf("Failed to send WebSocket notification: %v\n", err)
		} else {
			fmt.Printf("WebSocket notification sent to user %d\n", userID)
		}
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
