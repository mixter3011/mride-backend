package services

import (
	"mride-backend/internal/models"
	"time"
)

type OTPService interface {
	GenCode() string
	GenCryptoCode() (string, error)
	SaveOTP(phone, code string) error
	SaveEmailOTP(email, code string) error
	VerifyOTP(phone, code string) error
	VerifyEmailOTP(email, code string) error
	SendOTP(phone, code string) error
	SendEmailOTP(email, code string) error
}

type NotificationSvcInterface interface {
	CreateRideDeletedNotification(passengerID, rideID, driverID uint) error
	CreateRideJoinNotification(driverID, rideID, passengerID uint, passengerName string) error
	CreateRideLeaveNotification(driverID, rideID, passengerID uint, passengerName string) error
	CreateRideStartedNotification(passengerID, rideID, driverID uint) error
	CreateRideCompletedNotification(passengerID, rideID, driverID uint) error
	CreateSubscriptionJoinNotification(ownerID, subscriptionID, subscriberID uint, subscriberName string) error
	CreateSubscriptionLeaveNotification(ownerID, subscriptionID, subscriberID uint, subscriberName string) error
	CreateSubscriptionDeletedNotification(subscriberID, subscriptionID, ownerID uint) error
	CreateSubscriptionRideNotification(subscriberID, subscriptionID, driverID uint, departureTime time.Time) error
	CreateSubscriptionUpdatedNotification(subscriptionID, ownerID uint) error
	GetUserNotifications(userID uint, limit, offset int) (*models.NotificationsResp, error)
	MarkAsRead(userID, notificationID uint) error
	MarkAllAsRead(userID uint) error
	GetUnreadCount(userID uint) (int, error)
}

type WebSocketInterface interface {
	IsUserOnline(userID int) bool
	SendToUser(userID int, message WSMessage) error
	GetOnlineUsers() []int
	CleanupStaleConnections() error
	Shutdown()
}
