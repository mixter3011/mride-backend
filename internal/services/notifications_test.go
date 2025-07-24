package services

import (
	"encoding/json"
	"testing"

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
