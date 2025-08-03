package services

import "mride-backend/internal/models"

type OTPService interface {
	GenCode() string
	SaveOTP(phone, code string) error
	VerifyOTP(phone, code string) error
	SendOTP(phone, code string) error
}

type NotificationSvcInterface interface {
	CreateRideDeletedNotification(passengerID, rideID, driverID uint) error
	CreateRideJoinNotification(driverID, rideID, passengerID uint, passengerName string) error
	CreateRideLeaveNotification(driverID, rideID, passengerID uint, passengerName string) error
	CreateRideStartedNotification(passengerID, rideID, driverID uint) error
	CreateRideCompletedNotification(passengerID, rideID, driverID uint) error
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
