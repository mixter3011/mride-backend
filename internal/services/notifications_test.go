package services

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"mride-backend/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type MockWebSocketSvc struct {
	mock.Mock
}

func (m *MockWebSocketSvc) IsUserOnline(userID int) bool {
	args := m.Called(userID)
	return args.Bool(0)
}

func (m *MockWebSocketSvc) SendToUser(userID int, message WSMessage) error {
	args := m.Called(userID, message)
	return args.Error(0)
}

func (m *MockWebSocketSvc) GetOnlineUsers() []int {
	args := m.Called()
	return args.Get(0).([]int)
}

func (m *MockWebSocketSvc) CleanupStaleConnections() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockWebSocketSvc) Shutdown() {
	m.Called()
}

func (m *MockNotificationSvc) CreateRideJoinNotification(driverID, rideID, passengerID uint, passengerName string) error {
	args := m.Called(driverID, rideID, passengerID, passengerName)
	return args.Error(0)
}

func (m *MockNotificationSvc) CreateRideLeaveNotification(driverID, rideID, passengerID uint, passengerName string) error {
	args := m.Called(driverID, rideID, passengerID, passengerName)
	return args.Error(0)
}

func (m *MockNotificationSvc) CreateRideDeletedNotification(passengerID, rideID, driverID uint) error {
	args := m.Called(passengerID, rideID, driverID)
	return args.Error(0)
}

func (m *MockNotificationSvc) CreateRideStartedNotification(passengerID, rideID, driverID uint) error {
	args := m.Called(passengerID, rideID, driverID)
	return args.Error(0)
}

func (m *MockNotificationSvc) CreateRideCompletedNotification(passengerID, rideID, driverID uint) error {
	args := m.Called(passengerID, rideID, driverID)
	return args.Error(0)
}

func (m *MockNotificationSvc) CreateSubscriptionJoinNotification(ownerID, subscriptionID, subscriberID uint, subscriberName string) error {
	args := m.Called(ownerID, subscriptionID, subscriberID, subscriberName)
	return args.Error(0)
}

func (m *MockNotificationSvc) CreateSubscriptionLeaveNotification(ownerID, subscriptionID, subscriberID uint, subscriberName string) error {
	args := m.Called(ownerID, subscriptionID, subscriberID, subscriberName)
	return args.Error(0)
}

func (m *MockNotificationSvc) CreateSubscriptionDeletedNotification(subscriberID, subscriptionID, ownerID uint) error {
	args := m.Called(subscriberID, subscriptionID, ownerID)
	return args.Error(0)
}

func (m *MockNotificationSvc) CreateSubscriptionRideNotification(subscriberID, subscriptionID, driverID uint, departureTime time.Time) error {
	args := m.Called(subscriberID, subscriptionID, driverID, departureTime)
	return args.Error(0)
}

func (m *MockNotificationSvc) CreateSubscriptionUpdatedNotification(subscriptionID, ownerID uint) error {
	args := m.Called(subscriptionID, ownerID)
	return args.Error(0)
}

func (m *MockNotificationSvc) GetUserNotifications(userID uint, limit, offset int) (*models.NotificationsResp, error) {
	args := m.Called(userID, limit, offset)
	return args.Get(0).(*models.NotificationsResp), args.Error(1)
}

func (m *MockNotificationSvc) CreateChatNotification(recipientID, rideID, senderID uint, senderName string) error {
	args := m.Called(recipientID, rideID, senderID, senderName)
	return args.Error(0)
}

func (m *MockNotificationSvc) MarkAsRead(userID, notificationID uint) error {
	args := m.Called(userID, notificationID)
	return args.Error(0)
}

func (m *MockNotificationSvc) MarkAllAsRead(userID uint) error {
	args := m.Called(userID)
	return args.Error(0)
}

func (m *MockNotificationSvc) GetUnreadCount(userID uint) (int, error) {
	args := m.Called(userID)
	return args.Int(0), args.Error(1)
}

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	err = db.AutoMigrate(&models.User{}, &models.Notification{})
	assert.NoError(t, err)

	return db
}

func TestNotificationSvc_CreateRideJoinNotification(t *testing.T) {
	db := setupTestDB(t)

	mockWS := new(MockWebSocketSvc)

	notificationSvc := &NotificationSvc{
		db:           db,
		webSocketSvc: mockWS,
	}

	driverID := uint(1)
	rideID := uint(100)
	passengerID := uint(2)
	passengerName := "John Doe"

	mockWS.On("IsUserOnline", 1).Return(true)
	mockWS.On("SendToUser", 1, mock.AnythingOfType("WSMessage")).Return(nil)

	err := notificationSvc.CreateRideJoinNotification(driverID, rideID, passengerID, passengerName)
	assert.NoError(t, err)

	var notification models.Notification
	err = db.Where("user_id = ? AND type = ?", driverID, models.NotificationTypeRideJoin).First(&notification).Error
	assert.NoError(t, err)
	assert.Equal(t, "New Passenger Joined", notification.Title)
	assert.Equal(t, "John Doe joined your ride", notification.Message)
	assert.False(t, notification.Read)

	var data models.NotificationData
	err = json.Unmarshal(notification.Data, &data)
	assert.NoError(t, err)
	assert.Equal(t, rideID, data.RideID)
	assert.Equal(t, passengerID, data.UserID)
	assert.Equal(t, passengerName, data.UserName)
	assert.Equal(t, "join", data.ActionType)

	mockWS.AssertExpectations(t)
}

func TestNotificationSvc_CreateRideLeaveNotification(t *testing.T) {
	db := setupTestDB(t)
	mockWS := new(MockWebSocketSvc)
	notificationSvc := &NotificationSvc{
		db:           db,
		webSocketSvc: mockWS,
	}

	driverID := uint(1)
	rideID := uint(100)
	passengerID := uint(2)
	passengerName := "John Doe"

	mockWS.On("IsUserOnline", 1).Return(false)

	err := notificationSvc.CreateRideLeaveNotification(driverID, rideID, passengerID, passengerName)
	assert.NoError(t, err)

	var notification models.Notification
	err = db.Where("user_id = ? AND type = ?", driverID, models.NotificationTypeRideLeave).First(&notification).Error
	assert.NoError(t, err)
	assert.Equal(t, "Passenger Left", notification.Title)
	assert.Equal(t, "John Doe left your ride", notification.Message)

	mockWS.AssertExpectations(t)
}

func TestNotificationSvc_CreateRideDeletedNotification(t *testing.T) {
	db := setupTestDB(t)
	mockWS := new(MockWebSocketSvc)
	notificationSvc := &NotificationSvc{
		db:           db,
		webSocketSvc: mockWS,
	}

	driver := models.User{
		ID:       1,
		FullName: "Driver Smith",
	}
	db.Create(&driver)

	passengerID := uint(2)
	rideID := uint(100)
	driverID := uint(1)

	mockWS.On("IsUserOnline", 2).Return(true)
	mockWS.On("SendToUser", 2, mock.AnythingOfType("WSMessage")).Return(nil)

	err := notificationSvc.CreateRideDeletedNotification(passengerID, rideID, driverID)
	assert.NoError(t, err)

	var notification models.Notification
	err = db.Where("user_id = ? AND type = ?", passengerID, models.NotificationTypeRideDelete).First(&notification).Error
	assert.NoError(t, err)
	assert.Equal(t, "Ride Cancelled", notification.Title)
	assert.Contains(t, notification.Message, "Driver Smith")

	mockWS.AssertExpectations(t)
}

func TestNotificationSvc_CreateRideStartedNotification(t *testing.T) {
	db := setupTestDB(t)
	mockWS := new(MockWebSocketSvc)
	notificationSvc := &NotificationSvc{
		db:           db,
		webSocketSvc: mockWS,
	}

	driver := models.User{
		ID:       1,
		FullName: "Driver Smith",
	}
	db.Create(&driver)

	passengerID := uint(2)
	rideID := uint(100)
	driverID := uint(1)

	mockWS.On("IsUserOnline", 2).Return(true)
	mockWS.On("SendToUser", 2, mock.AnythingOfType("WSMessage")).Return(nil)

	err := notificationSvc.CreateRideStartedNotification(passengerID, rideID, driverID)
	assert.NoError(t, err)

	var notification models.Notification
	err = db.Where("user_id = ? AND type = ?", passengerID, models.NotificationTypeRideStarted).First(&notification).Error
	assert.NoError(t, err)
	assert.Equal(t, "Ride Started", notification.Title)
	assert.Contains(t, notification.Message, "Driver Smith")

	mockWS.AssertExpectations(t)
}

func TestNotificationSvc_CreateRideCompletedNotification(t *testing.T) {
	db := setupTestDB(t)
	mockWS := new(MockWebSocketSvc)
	notificationSvc := &NotificationSvc{
		db:           db,
		webSocketSvc: mockWS,
	}

	driver := models.User{
		ID:       1,
		FullName: "Driver Smith",
	}
	db.Create(&driver)

	passengerID := uint(2)
	rideID := uint(100)
	driverID := uint(1)

	mockWS.On("IsUserOnline", 2).Return(false)

	err := notificationSvc.CreateRideCompletedNotification(passengerID, rideID, driverID)
	assert.NoError(t, err)

	var notification models.Notification
	err = db.Where("user_id = ? AND type = ?", passengerID, models.NotificationTypeRideCompleted).First(&notification).Error
	assert.NoError(t, err)
	assert.Equal(t, "Ride Completed", notification.Title)
	assert.Contains(t, notification.Message, "Driver Smith")

	mockWS.AssertExpectations(t)
}

func TestNotificationSvc_GetUserNotifications(t *testing.T) {
	db := setupTestDB(t)
	mockWS := new(MockWebSocketSvc)
	notificationSvc := &NotificationSvc{
		db:           db,
		webSocketSvc: mockWS,
	}

	userID := uint(1)

	notifications := []models.Notification{
		{
			UserID:  userID,
			Type:    models.NotificationTypeRideJoin,
			Title:   "Test 1",
			Message: "Message 1",
			Data:    []byte(`{"test": "data1"}`),
			Read:    false,
		},
		{
			UserID:  userID,
			Type:    models.NotificationTypeRideLeave,
			Title:   "Test 2",
			Message: "Message 2",
			Data:    []byte(`{"test": "data2"}`),
			Read:    true,
		},
	}

	for _, notif := range notifications {
		db.Create(&notif)
	}

	resp, err := notificationSvc.GetUserNotifications(userID, 10, 0)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Notifications, 2)
	assert.Equal(t, 1, resp.UnreadCount)
}

func TestNotificationSvc_MarkAsRead(t *testing.T) {
	db := setupTestDB(t)
	mockWS := new(MockWebSocketSvc)
	notificationSvc := &NotificationSvc{
		db:           db,
		webSocketSvc: mockWS,
	}

	userID := uint(1)
	notification := models.Notification{
		UserID:  userID,
		Type:    models.NotificationTypeRideJoin,
		Title:   "Test",
		Message: "Test Message",
		Data:    []byte(`{"test": "data"}`),
		Read:    false,
	}
	db.Create(&notification)

	err := notificationSvc.MarkAsRead(userID, notification.ID)
	assert.NoError(t, err)

	var updatedNotif models.Notification
	db.First(&updatedNotif, notification.ID)
	assert.True(t, updatedNotif.Read)
}

func TestNotificationSvc_MarkAllAsRead(t *testing.T) {
	db := setupTestDB(t)
	mockWS := new(MockWebSocketSvc)
	notificationSvc := &NotificationSvc{
		db:           db,
		webSocketSvc: mockWS,
	}

	userID := uint(1)
	notifications := []models.Notification{
		{UserID: userID, Type: "test1", Title: "Test 1", Message: "Msg 1", Data: []byte(`{}`), Read: false},
		{UserID: userID, Type: "test2", Title: "Test 2", Message: "Msg 2", Data: []byte(`{}`), Read: false},
	}

	for _, notif := range notifications {
		db.Create(&notif)
	}

	err := notificationSvc.MarkAllAsRead(userID)
	assert.NoError(t, err)

	var count int64
	db.Model(&models.Notification{}).Where("user_id = ? AND read = ?", userID, false).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestNotificationSvc_GetUnreadCount(t *testing.T) {
	db := setupTestDB(t)
	mockWS := new(MockWebSocketSvc)
	notificationSvc := &NotificationSvc{
		db:           db,
		webSocketSvc: mockWS,
	}

	userID := uint(1)
	notifications := []models.Notification{
		{UserID: userID, Type: "test1", Title: "Test 1", Message: "Msg 1", Data: []byte(`{}`), Read: false},
		{UserID: userID, Type: "test2", Title: "Test 2", Message: "Msg 2", Data: []byte(`{}`), Read: false},
		{UserID: userID, Type: "test3", Title: "Test 3", Message: "Msg 3", Data: []byte(`{}`), Read: true},
	}

	for _, notif := range notifications {
		db.Create(&notif)
	}

	count, err := notificationSvc.GetUnreadCount(userID)
	assert.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestNotificationSvc_CreateSubscriptionJoinNotification(t *testing.T) {
	db := setupTestDB(t)
	mockWS := new(MockWebSocketSvc)
	notificationSvc := &NotificationSvc{
		db:           db,
		webSocketSvc: mockWS,
	}

	ownerID := uint(1)
	subscriptionID := uint(100)
	subscriberID := uint(2)
	subscriberName := "John Doe"

	mockWS.On("IsUserOnline", 1).Return(true)
	mockWS.On("SendToUser", 1, mock.AnythingOfType("WSMessage")).Return(nil)

	err := notificationSvc.CreateSubscriptionJoinNotification(ownerID, subscriptionID, subscriberID, subscriberName)
	assert.NoError(t, err)

	var notification models.Notification
	err = db.Where("user_id = ? AND type = ?", ownerID, "subscription_join").First(&notification).Error
	assert.NoError(t, err)
	assert.Equal(t, "New Subscriber", notification.Title)
	assert.Contains(t, notification.Message, "John Doe")

	mockWS.AssertExpectations(t)
}

func TestNotificationSvc_CreateSubscriptionDeletedNotification(t *testing.T) {
	db := setupTestDB(t)

	err := db.AutoMigrate(&models.RideSubscription{})
	if err != nil {
		t.Skip("Skipping test due to migration failure")
	}

	mockWS := new(MockWebSocketSvc)
	notificationSvc := &NotificationSvc{
		db:           db,
		webSocketSvc: mockWS,
	}

	subscription := models.RideSubscription{
		ID:               100,
		UserID:           1,
		Title:            "Test Subscription",
		CarNumber:        "ABC123",
		CarModel:         "Honda",
		PassengerCount:   3,
		FromLocation:     "Mumbai",
		ToLocation:       "Pune",
		FromLatitude:     19.0760,
		FromLongitude:    72.8777,
		ToLatitude:       18.5204,
		ToLongitude:      73.8567,
		DepartureTime:    "09:00",
		RecurringDays:    `["Monday"]`,
		StartDate:        time.Now().Add(24 * time.Hour),
		Status:           "active",
		MaxSubscribers:   4,
		NotificationTime: 60,
	}
	err = db.Create(&subscription).Error
	if err != nil {
		t.Skip("Skipping test due to subscription creation failure")
	}

	subscriberID := uint(2)
	subscriptionID := uint(100)
	ownerID := uint(1)

	mockWS.On("IsUserOnline", 2).Return(false)

	err = notificationSvc.CreateSubscriptionDeletedNotification(subscriberID, subscriptionID, ownerID)
	assert.NoError(t, err)

	var notification models.Notification
	err = db.Where("user_id = ? AND type = ?", subscriberID, "subscription_deleted").First(&notification).Error
	assert.NoError(t, err)
	assert.Equal(t, "Subscription Cancelled", notification.Title)
	assert.Contains(t, notification.Message, "Mumbai")
	assert.Contains(t, notification.Message, "Pune")

	mockWS.AssertExpectations(t)
}

func TestNotificationSvc_CreateSubscriptionRideNotification(t *testing.T) {
	db := setupTestDB(t)

	err := db.AutoMigrate(&models.RideSubscription{})
	assert.NoError(t, err)

	mockWS := new(MockWebSocketSvc)
	notificationSvc := &NotificationSvc{
		db:           db,
		webSocketSvc: mockWS,
	}

	user := models.User{ID: 1, FullName: "Driver Smith"}
	db.Create(&user)

	subscription := models.RideSubscription{
		ID:               100,
		UserID:           1,
		Title:            "Test Subscription",
		CarNumber:        "ABC123",
		CarModel:         "Honda",
		PassengerCount:   3,
		FromLocation:     "Mumbai",
		ToLocation:       "Pune",
		FromLatitude:     19.0760,
		FromLongitude:    72.8777,
		ToLatitude:       18.5204,
		ToLongitude:      73.8567,
		DepartureTime:    "09:00",
		RecurringDays:    `["Monday"]`,
		StartDate:        time.Now().Add(24 * time.Hour),
		Status:           "active",
		MaxSubscribers:   4,
		NotificationTime: 60,
	}
	db.Create(&subscription)

	subscriberID := uint(2)
	subscriptionID := uint(100)
	driverID := uint(1)
	departureTime := time.Now().Add(2 * time.Hour)

	mockWS.On("IsUserOnline", 2).Return(true)
	mockWS.On("SendToUser", 2, mock.AnythingOfType("WSMessage")).Return(nil)

	err = notificationSvc.CreateSubscriptionRideNotification(subscriberID, subscriptionID, driverID, departureTime)
	assert.NoError(t, err)

	var notification models.Notification
	err = db.Where("user_id = ? AND type = ?", subscriberID, "subscription_ride_reminder").First(&notification).Error
	assert.NoError(t, err)
	assert.Equal(t, "Ride Reminder", notification.Title)
	assert.Contains(t, notification.Message, "Mumbai")
	assert.Contains(t, notification.Message, "Pune")

	mockWS.AssertExpectations(t)
}

func TestNotificationSvc_CreateSubscriptionUpdatedNotification(t *testing.T) {
	db := setupTestDB(t)
	err := db.AutoMigrate(&models.RideSubscription{}, &models.SubscriptionSubscriber{})
	assert.NoError(t, err)

	mockWS := new(MockWebSocketSvc)
	notificationSvc := &NotificationSvc{
		db:           db,
		webSocketSvc: mockWS,
	}

	subscription := models.RideSubscription{
		ID:               100,
		UserID:           1,
		Title:            "Test Subscription",
		CarNumber:        "ABC123",
		CarModel:         "Honda",
		PassengerCount:   3,
		FromLocation:     "Mumbai",
		ToLocation:       "Pune",
		FromLatitude:     19.0760,
		FromLongitude:    72.8777,
		ToLatitude:       18.5204,
		ToLongitude:      73.8567,
		DepartureTime:    "09:00",
		RecurringDays:    `["Monday"]`,
		StartDate:        time.Now().Add(24 * time.Hour),
		Status:           "active",
		MaxSubscribers:   4,
		NotificationTime: 60,
	}
	db.Create(&subscription)

	subscriber := models.SubscriptionSubscriber{
		SubscriptionID: 100,
		SubscriberID:   2,
		Status:         "active",
		NotifyEnabled:  true,
	}
	db.Create(&subscriber)

	mockWS.On("IsUserOnline", 2).Return(false)

	err = notificationSvc.CreateSubscriptionUpdatedNotification(100, 1)
	assert.NoError(t, err)

	var notification models.Notification
	err = db.Where("user_id = ? AND type = ?", uint(2), "subscription_updated").First(&notification).Error
	assert.NoError(t, err)
	assert.Equal(t, "Subscription Updated", notification.Title)

	mockWS.AssertExpectations(t)
}

func TestNotificationSvc_NilWebSocketService(t *testing.T) {
	db := setupTestDB(t)
	notificationSvc := &NotificationSvc{
		db:           db,
		webSocketSvc: nil,
	}

	err := notificationSvc.CreateRideJoinNotification(1, 100, 2, "John Doe")
	assert.NoError(t, err)

	var notification models.Notification
	err = db.Where("user_id = ? AND type = ?", uint(1), models.NotificationTypeRideJoin).First(&notification).Error
	assert.NoError(t, err)
}

func TestNotificationSvc_GetUserNotifications_DefaultLimit(t *testing.T) {
	db := setupTestDB(t)
	notificationSvc := &NotificationSvc{db: db}

	userID := uint(1)

	for i := 0; i < 25; i++ {
		notification := models.Notification{
			UserID:  userID,
			Type:    "test",
			Title:   fmt.Sprintf("Test %d", i),
			Message: fmt.Sprintf("Message %d", i),
			Data:    []byte(`{}`),
			Read:    i%2 == 0,
		}
		db.Create(&notification)
	}

	resp, err := notificationSvc.GetUserNotifications(userID, 0, 0)
	assert.NoError(t, err)
	assert.Len(t, resp.Notifications, 20)
	assert.Equal(t, 12, resp.UnreadCount)
}

func TestNotificationSvc_WebSocketError(t *testing.T) {
	db := setupTestDB(t)
	mockWS := new(MockWebSocketSvc)
	notificationSvc := &NotificationSvc{
		db:           db,
		webSocketSvc: mockWS,
	}

	mockWS.On("IsUserOnline", 1).Return(true)
	mockWS.On("SendToUser", 1, mock.AnythingOfType("WSMessage")).Return(fmt.Errorf("websocket error"))

	err := notificationSvc.CreateRideJoinNotification(1, 100, 2, "John Doe")
	assert.NoError(t, err)

	var notification models.Notification
	err = db.Where("user_id = ? AND type = ?", uint(1), models.NotificationTypeRideJoin).First(&notification).Error
	assert.NoError(t, err)

	mockWS.AssertExpectations(t)
}
