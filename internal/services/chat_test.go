package services

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"mride-backend/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type MockWebSocketInterface struct {
	mock.Mock
}

func (m *MockWebSocketInterface) Shutdown() {
	m.Called()
}

func (m *MockWebSocketInterface) SendToUser(userID int, message WSMessage) error {
	args := m.Called(userID, message)
	return args.Error(0)
}

func (m *MockWebSocketInterface) IsUserOnline(userID int) bool {
	args := m.Called(userID)
	return args.Bool(0)
}

func (m *MockWebSocketInterface) GetOnlineUsers() []int {
	args := m.Called()
	return args.Get(0).([]int)
}

func (m *MockWebSocketInterface) CleanupStaleConnections() error {
	args := m.Called()
	return args.Error(0)
}

type MockNotificationInterface struct {
	mock.Mock
}

func (m *MockNotificationInterface) GetUnreadCount(userID uint) (int, error) {
	args := m.Called(userID)
	return args.Int(0), args.Error(1)
}

func (m *MockNotificationInterface) GetUserNotifications(userID uint, limit, offset int) (*models.NotificationsResp, error) {
	args := m.Called(userID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.NotificationsResp), args.Error(1)
}

func (m *MockNotificationInterface) MarkAllAsRead(userID uint) error {
	args := m.Called(userID)
	return args.Error(0)
}

func (m *MockNotificationInterface) MarkAsRead(userID, notificationID uint) error {
	args := m.Called(userID, notificationID)
	return args.Error(0)
}

func (m *MockNotificationInterface) CreateRideDeletedNotification(passengerID, rideID, driverID uint) error {
	args := m.Called(passengerID, rideID, driverID)
	return args.Error(0)
}

func (m *MockNotificationInterface) CreateRideJoinNotification(driverID, rideID, passengerID uint, passengerName string) error {
	args := m.Called(driverID, rideID, passengerID, passengerName)
	return args.Error(0)
}

func (m *MockNotificationInterface) CreateRideLeaveNotification(driverID, rideID, passengerID uint, passengerName string) error {
	args := m.Called(driverID, rideID, passengerID, passengerName)
	return args.Error(0)
}

func (m *MockNotificationInterface) CreateRideStartedNotification(passengerID, rideID, driverID uint) error {
	args := m.Called(passengerID, rideID, driverID)
	return args.Error(0)
}

func (m *MockNotificationInterface) CreateRideCompletedNotification(passengerID, rideID, driverID uint) error {
	args := m.Called(passengerID, rideID, driverID)
	return args.Error(0)
}

func (m *MockNotificationInterface) CreateChatNotification(recipientID, rideID, senderID uint, senderName string) error {
	args := m.Called(recipientID, rideID, senderID, senderName)
	return args.Error(0)
}

func (m *MockNotificationInterface) CreateSubscriptionJoinNotification(ownerID, subscriptionID, subscriberID uint, subscriberName string) error {
	args := m.Called(ownerID, subscriptionID, subscriberID, subscriberName)
	return args.Error(0)
}

func (m *MockNotificationInterface) CreateSubscriptionLeaveNotification(ownerID, subscriptionID, subscriberID uint, subscriberName string) error {
	args := m.Called(ownerID, subscriptionID, subscriberID, subscriberName)
	return args.Error(0)
}

func (m *MockNotificationInterface) CreateSubscriptionDeletedNotification(subscriberID, subscriptionID, ownerID uint) error {
	args := m.Called(subscriberID, subscriptionID, ownerID)
	return args.Error(0)
}

func (m *MockNotificationInterface) CreateSubscriptionRideNotification(subscriberID, subscriptionID, driverID uint, departureTime time.Time) error {
	args := m.Called(subscriberID, subscriptionID, driverID, departureTime)
	return args.Error(0)
}

func (m *MockNotificationInterface) CreateSubscriptionUpdatedNotification(subscriptionID, ownerID uint) error {
	args := m.Called(subscriptionID, ownerID)
	return args.Error(0)
}

func setupChatTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	err = db.AutoMigrate(&models.User{}, &models.Ride{}, &models.RidePassenger{}, &models.RideChat{})
	assert.NoError(t, err)

	return db
}

func createTestData(db *gorm.DB) (uint, uint, uint) {
	driver := models.User{ID: 1, FullName: "Driver Smith", Email: "driver@test.com"}
	passenger := models.User{ID: 2, FullName: "Passenger John", Email: "passenger@test.com"}
	db.Create(&driver)
	db.Create(&passenger)

	ride := models.Ride{
		ID:             1,
		UserID:         1,
		CarNumber:      "ABC123",
		CarModel:       "Honda",
		PassengerCount: 3,
		FromLocation:   "Mumbai",
		ToLocation:     "Pune",
		DepartureTime:  time.Now().Add(2 * time.Hour),
		Status:         "active",
	}
	db.Create(&ride)

	ridePassenger := models.RidePassenger{
		RideID:      1,
		PassengerID: 2,
		Status:      "active",
	}
	db.Create(&ridePassenger)

	return 1, 2, 1
}

func TestChatSvc_SendChatMessage_Success(t *testing.T) {
	db := setupChatTestDB(t)
	mockWS := new(MockWebSocketInterface)
	mockNotif := new(MockNotificationInterface)

	chatSvc := NewChatSvc(db, mockWS, mockNotif)

	driverID, passengerID, rideID := createTestData(db)

	message := "Hello, when are we starting?"

	mockWS.On("SendToUser", int(driverID), mock.AnythingOfType("WSMessage")).Return(nil)
	mockNotif.On("CreateChatNotification", driverID, rideID, passengerID, "Passenger John").Return(nil)

	resp, err := chatSvc.SendChatMessage(passengerID, rideID, message)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, message, resp.Message)
	assert.Equal(t, passengerID, resp.SenderID)
	assert.Equal(t, rideID, resp.RideID)
	assert.Equal(t, "Passenger John", resp.Sender.FullName)

	var chatMessage models.RideChat
	err = db.First(&chatMessage, resp.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, message, chatMessage.Message)

	mockWS.AssertExpectations(t)
	mockNotif.AssertExpectations(t)
}

func TestChatSvc_SendChatMessage_UserNotPartOfRide(t *testing.T) {
	db := setupChatTestDB(t)
	mockWS := new(MockWebSocketInterface)
	mockNotif := new(MockNotificationInterface)

	chatSvc := NewChatSvc(db, mockWS, mockNotif)

	_, _, rideID := createTestData(db)
	unauthorizedUserID := uint(999)

	resp, err := chatSvc.SendChatMessage(unauthorizedUserID, rideID, "Hello")

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "you are not part of this ride")
}

func TestChatSvc_SendChatMessage_RideNotActive(t *testing.T) {
	db := setupChatTestDB(t)
	mockWS := new(MockWebSocketInterface)
	mockNotif := new(MockNotificationInterface)

	chatSvc := NewChatSvc(db, mockWS, mockNotif)

	driverID, _, rideID := createTestData(db)

	db.Model(&models.Ride{}).Where("id = ?", rideID).Update("status", "completed")

	resp, err := chatSvc.SendChatMessage(driverID, rideID, "Hello")

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "chat is only available for active rides")
}

func TestChatSvc_SendChatMessage_RideNotFound(t *testing.T) {
	db := setupChatTestDB(t)
	mockWS := new(MockWebSocketInterface)
	mockNotif := new(MockNotificationInterface)

	chatSvc := NewChatSvc(db, mockWS, mockNotif)

	driverID, _, _ := createTestData(db)
	nonExistentRideID := uint(999)

	resp, err := chatSvc.SendChatMessage(driverID, nonExistentRideID, "Hello")

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t,
		strings.Contains(err.Error(), "ride not found") ||
			strings.Contains(err.Error(), "you are not part of this ride"),
		"Expected error about ride not found or not being part of ride, got: %v", err)
}

func TestChatSvc_GetChatHistory_Success(t *testing.T) {
	db := setupChatTestDB(t)
	mockWS := new(MockWebSocketInterface)
	mockNotif := new(MockNotificationInterface)

	chatSvc := NewChatSvc(db, mockWS, mockNotif)

	driverID, passengerID, rideID := createTestData(db)

	messages := []models.RideChat{
		{RideID: rideID, SenderID: driverID, Message: "Hello passenger!"},
		{RideID: rideID, SenderID: passengerID, Message: "Hi driver!"},
		{RideID: rideID, SenderID: driverID, Message: "Ready to go?"},
	}

	for _, msg := range messages {
		db.Create(&msg)
	}

	resp, err := chatSvc.GetChatHistory(driverID, rideID, 10, 0)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Messages, 3)
	assert.Equal(t, 3, resp.Count)

	assert.Equal(t, "Hello passenger!", resp.Messages[0].Message)
	assert.Equal(t, "Hi driver!", resp.Messages[1].Message)
	assert.Equal(t, "Ready to go?", resp.Messages[2].Message)
}

func TestChatSvc_GetChatHistory_UserNotPartOfRide(t *testing.T) {
	db := setupChatTestDB(t)
	mockWS := new(MockWebSocketInterface)
	mockNotif := new(MockNotificationInterface)

	chatSvc := NewChatSvc(db, mockWS, mockNotif)

	_, _, rideID := createTestData(db)
	unauthorizedUserID := uint(999)

	resp, err := chatSvc.GetChatHistory(unauthorizedUserID, rideID, 10, 0)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "you are not part of this ride")
}

func TestChatSvc_GetChatHistory_WithPagination(t *testing.T) {
	db := setupChatTestDB(t)
	mockWS := new(MockWebSocketInterface)
	mockNotif := new(MockNotificationInterface)

	chatSvc := NewChatSvc(db, mockWS, mockNotif)

	driverID, _, rideID := createTestData(db)

	for i := 1; i <= 15; i++ {
		msg := models.RideChat{
			RideID:   rideID,
			SenderID: driverID,
			Message:  fmt.Sprintf("Message %d", i),
		}
		db.Create(&msg)
		time.Sleep(1 * time.Millisecond)
	}

	resp, err := chatSvc.GetChatHistory(driverID, rideID, 5, 0)
	assert.NoError(t, err)
	assert.Len(t, resp.Messages, 5)

	resp, err = chatSvc.GetChatHistory(driverID, rideID, 5, 5)
	assert.NoError(t, err)
	assert.Len(t, resp.Messages, 5)

	resp, err = chatSvc.GetChatHistory(driverID, rideID, 200, 0)
	assert.NoError(t, err)
	assert.Len(t, resp.Messages, 15)
}

func TestChatSvc_isUserPartOfRide_Driver(t *testing.T) {
	db := setupChatTestDB(t)
	mockWS := new(MockWebSocketInterface)
	mockNotif := new(MockNotificationInterface)

	chatSvc := NewChatSvc(db, mockWS, mockNotif)

	driverID, _, rideID := createTestData(db)

	isPartOfRide := chatSvc.isUserPartOfRide(driverID, rideID)
	assert.True(t, isPartOfRide)
}

func TestChatSvc_isUserPartOfRide_Passenger(t *testing.T) {
	db := setupChatTestDB(t)
	mockWS := new(MockWebSocketInterface)
	mockNotif := new(MockNotificationInterface)

	chatSvc := NewChatSvc(db, mockWS, mockNotif)

	_, passengerID, rideID := createTestData(db)

	isPartOfRide := chatSvc.isUserPartOfRide(passengerID, rideID)
	assert.True(t, isPartOfRide)
}

func TestChatSvc_isUserPartOfRide_NotPartOfRide(t *testing.T) {
	db := setupChatTestDB(t)
	mockWS := new(MockWebSocketInterface)
	mockNotif := new(MockNotificationInterface)

	chatSvc := NewChatSvc(db, mockWS, mockNotif)

	_, _, rideID := createTestData(db)
	unauthorizedUserID := uint(999)

	isPartOfRide := chatSvc.isUserPartOfRide(unauthorizedUserID, rideID)
	assert.False(t, isPartOfRide)
}

func TestChatSvc_DeleteChatHistory(t *testing.T) {
	db := setupChatTestDB(t)
	mockWS := new(MockWebSocketInterface)
	mockNotif := new(MockNotificationInterface)

	chatSvc := NewChatSvc(db, mockWS, mockNotif)

	driverID, _, rideID := createTestData(db)

	messages := []models.RideChat{
		{RideID: rideID, SenderID: driverID, Message: "Message 1"},
		{RideID: rideID, SenderID: driverID, Message: "Message 2"},
	}

	for _, msg := range messages {
		db.Create(&msg)
	}

	err := chatSvc.DeleteChatHistory(rideID)
	assert.NoError(t, err)

	var count int64
	db.Model(&models.RideChat{}).Where("ride_id = ?", rideID).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestChatSvc_GetUnreadChatCount_Placeholder(t *testing.T) {
	db := setupChatTestDB(t)
	mockWS := new(MockWebSocketInterface)
	mockNotif := new(MockNotificationInterface)

	chatSvc := NewChatSvc(db, mockWS, mockNotif)

	count, err := chatSvc.GetUnreadChatCount(1)
	assert.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestChatSvc_SendChatMessage_StartedRide(t *testing.T) {
	db := setupChatTestDB(t)
	mockWS := new(MockWebSocketInterface)
	mockNotif := new(MockNotificationInterface)

	chatSvc := NewChatSvc(db, mockWS, mockNotif)

	driverID, passengerID, rideID := createTestData(db)

	db.Model(&models.Ride{}).Where("id = ?", rideID).Update("status", "started")

	message := "We have started the ride!"

	mockWS.On("SendToUser", int(passengerID), mock.AnythingOfType("WSMessage")).Return(nil)
	mockNotif.On("CreateChatNotification", passengerID, rideID, driverID, "Driver Smith").Return(nil)

	resp, err := chatSvc.SendChatMessage(driverID, rideID, message)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, message, resp.Message)

	mockWS.AssertExpectations(t)
	mockNotif.AssertExpectations(t)
}
