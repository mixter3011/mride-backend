package services

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"mride-backend/internal/models"

	"gorm.io/gorm"
)

type NotificationSvc struct {
	db                  *gorm.DB
	webSocketSvc        WebSocketInterface
	pushNotificationSvc *PushNotificationSvc
}

func NewNotificationSvc(db *gorm.DB, webSocketSvc WebSocketInterface) *NotificationSvc {
	return &NotificationSvc{
		db:           db,
		webSocketSvc: webSocketSvc,
	}
}

func (n *NotificationSvc) SetPushNotificationService(pushSvc *PushNotificationSvc) {
	n.pushNotificationSvc = pushSvc
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
			log.Printf("Failed to send WebSocket notification: %v\n", err)
		} else {
			log.Printf("WebSocket notification sent to user %d\n", userID)
		}
	} else {

		if n.pushNotificationSvc != nil {
			go n.pushNotificationSvc.SendPushNotification(userID, &notification)
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

func (n *NotificationSvc) CreateSubscriptionJoinNotification(ownerID, subscriptionID, subscriberID uint, subscriberName string) error {
	data := models.NotificationData{
		RideID:     subscriptionID,
		UserID:     subscriberID,
		UserName:   subscriberName,
		ActionType: "subscribe",
	}
	return n.createAndSendNotification(ownerID, "subscription_join", "New Subscriber",
		fmt.Sprintf("%s has subscribed to your ride", subscriberName), data)
}

func (n *NotificationSvc) CreateSubscriptionLeaveNotification(ownerID, subscriptionID, subscriberID uint, subscriberName string) error {
	data := models.NotificationData{
		RideID:     subscriptionID,
		UserID:     subscriberID,
		UserName:   subscriberName,
		ActionType: "unsubscribe",
	}
	return n.createAndSendNotification(ownerID, "subscription_leave", "Subscriber Left",
		fmt.Sprintf("%s has unsubscribed from your ride", subscriberName), data)
}

func (n *NotificationSvc) CreateSubscriptionDeletedNotification(subscriberID, subscriptionID, ownerID uint) error {
	var subscription models.RideSubscription
	message := "A ride subscription has been cancelled by the driver"
	if err := n.db.First(&subscription, subscriptionID).Error; err == nil {
		message = fmt.Sprintf("The ride subscription from %s to %s has been cancelled",
			subscription.FromLocation, subscription.ToLocation)
	}

	data := models.NotificationData{
		RideID:     subscriptionID,
		UserID:     ownerID,
		UserName:   "Driver",
		ActionType: "deleted",
	}
	return n.createAndSendNotification(subscriberID, "subscription_deleted", "Subscription Cancelled", message, data)
}

func (n *NotificationSvc) CreateSubscriptionRideNotification(subscriberID, subscriptionID, driverID uint, departureTime time.Time) error {
	var subscription models.RideSubscription
	var driverName string = "Driver"

	if err := n.db.Preload("User").First(&subscription, subscriptionID).Error; err == nil {
		driverName = subscription.User.FullName
	}

	timeStr := departureTime.Format("3:04 PM")
	message := fmt.Sprintf("Your subscribed ride from %s to %s departs at %s today",
		subscription.FromLocation, subscription.ToLocation, timeStr)

	data := models.NotificationData{
		RideID:     subscriptionID,
		UserID:     driverID,
		UserName:   driverName,
		ActionType: "ride_reminder",
	}
	return n.createAndSendNotification(subscriberID, "subscription_ride_reminder", "Ride Reminder", message, data)
}

func (n *NotificationSvc) CreateSubscriptionUpdatedNotification(subscriptionID, ownerID uint) error {
	var subscription models.RideSubscription
	if err := n.db.First(&subscription, subscriptionID).Error; err != nil {
		return err
	}

	var subscribers []models.SubscriptionSubscriber
	if err := n.db.Where("subscription_id = ? AND status = ?", subscriptionID, "active").Find(&subscribers).Error; err != nil {
		return err
	}

	for _, subscriber := range subscribers {
		data := models.NotificationData{
			RideID:     subscriptionID,
			UserID:     ownerID,
			UserName:   "Driver",
			ActionType: "updated",
		}
		message := fmt.Sprintf("The ride subscription from %s to %s has been updated",
			subscription.FromLocation, subscription.ToLocation)

		if err := n.createAndSendNotification(subscriber.SubscriberID, "subscription_updated",
			"Subscription Updated", message, data); err != nil {
			log.Printf("Failed to create subscription updated notification: %v", err)
		}
	}
	return nil
}

func (n *NotificationSvc) CreateChatNotification(recipientID, rideID, senderID uint, senderName string) error {
	dataMap := map[string]interface{}{
		"ride_id":     rideID,
		"sender_id":   senderID,
		"sender_name": senderName,
	}

	dataJSON, err := json.Marshal(dataMap)
	if err != nil {
		return err
	}

	notification := models.Notification{
		UserID:  recipientID,
		Type:    "chat_message",
		Title:   "New Chat Message",
		Message: fmt.Sprintf("%s sent you a message in ride #%d", senderName, rideID),
		Data:    dataJSON,
	}

	if err := n.db.Create(&notification).Error; err != nil {
		return err
	}

	if n.webSocketSvc != nil && n.webSocketSvc.IsUserOnline(int(recipientID)) {
		wsMessage := WSMessage{
			Type:    "notification",
			Title:   notification.Title,
			Message: notification.Message,
			Data:    dataMap,
		}

		if err := n.webSocketSvc.SendToUser(int(recipientID), wsMessage); err != nil {
			log.Printf("Failed to send WebSocket notification: %v", err)
		}
	} else {

		if n.pushNotificationSvc != nil {
			go n.pushNotificationSvc.SendPushNotification(recipientID, &notification)
		}
	}

	return nil
}

var _ NotificationSvcInterface = (*NotificationSvc)(nil)
