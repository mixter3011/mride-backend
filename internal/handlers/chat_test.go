package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"mride-backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockChatService struct {
	mock.Mock
}

func (m *MockChatService) MarkChatAsRead(userID uint, rideID uint) error {
	args := m.Called(userID, rideID)
	return args.Error(0)
}

func (m *MockChatService) GetRidesWithChats(userID uint) ([]uint, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]uint), args.Error(1)
}

func (m *MockChatService) GetActiveRidesWithChats(userID uint) ([]models.ChatRoomInfo, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.ChatRoomInfo), args.Error(1)
}

func (m *MockChatService) GetExpiredRidesWithChats(userID uint) ([]models.ChatRoomInfo, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.ChatRoomInfo), args.Error(1)
}

func (m *MockChatService) CanSendMessage(userID, rideID uint) (bool, string) {
	args := m.Called(userID, rideID)
	return args.Bool(0), args.String(1)
}

func (m *MockChatService) SendChatMessage(userID, rideID uint, message string) (*models.ChatMessageResp, error) {
	args := m.Called(userID, rideID, message)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ChatMessageResp), args.Error(1)
}

func (m *MockChatService) GetChatHistory(userID, rideID uint, limit, offset int) (*models.GetChatHistoryResp, error) {
	args := m.Called(userID, rideID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.GetChatHistoryResp), args.Error(1)
}

func (m *MockChatService) GetUnreadChatCount(userID uint) (int, error) {
	args := m.Called(userID)
	return args.Int(0), args.Error(1)
}

func setupChatTestRouter() (*gin.Engine, *MockChatService) {
	gin.SetMode(gin.TestMode)

	mockChatService := new(MockChatService)
	chatHandler := NewChatHandler(mockChatService)

	router := gin.New()

	router.Use(func(c *gin.Context) {
		c.Set("user_id", 1)
		c.Next()
	})

	router.POST("/ride/:id/chat/send", chatHandler.SendMessage)
	router.GET("/ride/:id/chat/history", chatHandler.GetChatHistory)
	router.GET("/ride/:id/chat/can-send", chatHandler.CheckCanSendMessage)
	router.GET("/chat/unread-count", chatHandler.GetUnreadCount)
	router.GET("/chat/active", chatHandler.GetActiveChats)
	router.GET("/chat/expired", chatHandler.GetExpiredChats)
	router.GET("/chat/rides", chatHandler.GetRidesWithChats)
	router.POST("/ride/:id/chat/mark-read", chatHandler.MarkChatAsRead)

	return router, mockChatService
}

func TestChatHandler_SendMessage_Success(t *testing.T) {
	router, mockService := setupChatTestRouter()

	message := "Hello, when are we leaving?"
	rideID := uint(123)
	userID := uint(1)

	expectedResp := &models.ChatMessageResp{
		ID:        1,
		RideID:    rideID,
		SenderID:  userID,
		Message:   message,
		CreatedAt: time.Now(),
		IsMine:    true,
		Sender: struct {
			ID       uint   `json:"id"`
			FullName string `json:"full_name"`
		}{
			ID:       userID,
			FullName: "Test User",
		},
	}

	mockService.On("SendChatMessage", userID, rideID, message).Return(expectedResp, nil)

	reqBody := models.SendChatMessageReq{Message: message}
	bodyBytes, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/ride/123/chat/send", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Message sent successfully", response["message"])
	assert.NotNil(t, response["data"])

	mockService.AssertExpectations(t)
}

func TestChatHandler_SendMessage_InvalidRideID(t *testing.T) {
	router, _ := setupChatTestRouter()

	reqBody := models.SendChatMessageReq{Message: "Test message"}
	bodyBytes, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/ride/invalid/chat/send", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Invalid ride ID", response["error"])
}

func TestChatHandler_SendMessage_EmptyMessage(t *testing.T) {
	router, _ := setupChatTestRouter()

	reqBody := models.SendChatMessageReq{Message: ""}
	bodyBytes, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/ride/123/chat/send", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestChatHandler_SendMessage_ServiceError(t *testing.T) {
	router, mockService := setupChatTestRouter()

	message := "Hello"
	rideID := uint(123)
	userID := uint(1)

	mockService.On("SendChatMessage", userID, rideID, message).Return((*models.ChatMessageResp)(nil), errors.New("service error"))

	reqBody := models.SendChatMessageReq{Message: message}
	bodyBytes, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/ride/123/chat/send", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	mockService.AssertExpectations(t)
}

func TestChatHandler_GetChatHistory_Success(t *testing.T) {
	router, mockService := setupChatTestRouter()

	rideID := uint(123)
	userID := uint(1)
	limit := 50
	offset := 0

	expectedMessages := []models.ChatMessageResp{
		{
			ID:        1,
			RideID:    rideID,
			SenderID:  1,
			Message:   "Hello driver!",
			CreatedAt: time.Now().Add(-2 * time.Minute),
			IsMine:    true,
		},
		{
			ID:        2,
			RideID:    rideID,
			SenderID:  2,
			Message:   "Hi passenger!",
			CreatedAt: time.Now().Add(-1 * time.Minute),
			IsMine:    false,
		},
	}

	expectedResp := &models.GetChatHistoryResp{
		Messages: expectedMessages,
		Count:    2,
	}

	mockService.On("GetChatHistory", userID, rideID, limit, offset).Return(expectedResp, nil)

	req, _ := http.NewRequest("GET", "/ride/123/chat/history", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Chat history retrieved successfully", response["message"])
	assert.NotNil(t, response["data"])

	mockService.AssertExpectations(t)
}

func TestChatHandler_GetActiveChats_Success(t *testing.T) {
	router, mockService := setupChatTestRouter()

	userID := uint(1)
	expectedRooms := []models.ChatRoomInfo{
		{
			RideID:          123,
			OtherUserID:     456,
			OtherUserName:   "John Doe",
			LastMessage:     "See you soon",
			LastMessageTime: time.Now(),
			RideStatus:      "active",
			IsDriver:        true,
		},
	}

	mockService.On("GetActiveRidesWithChats", userID).Return(expectedRooms, nil)

	req, _ := http.NewRequest("GET", "/chat/active", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Active chats retrieved successfully", response["message"])

	mockService.AssertExpectations(t)
}

func TestChatHandler_GetExpiredChats_Success(t *testing.T) {
	router, mockService := setupChatTestRouter()

	userID := uint(1)
	expectedRooms := []models.ChatRoomInfo{
		{
			RideID:          789,
			OtherUserID:     101,
			OtherUserName:   "Jane Smith",
			LastMessage:     "Thanks!",
			LastMessageTime: time.Now().Add(-24 * time.Hour),
			RideStatus:      "completed",
			IsDriver:        false,
		},
	}

	mockService.On("GetExpiredRidesWithChats", userID).Return(expectedRooms, nil)

	req, _ := http.NewRequest("GET", "/chat/expired", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Expired chats retrieved successfully", response["message"])

	mockService.AssertExpectations(t)
}

func TestChatHandler_CheckCanSendMessage_Success(t *testing.T) {
	router, mockService := setupChatTestRouter()

	userID := uint(1)
	rideID := uint(123)

	mockService.On("CanSendMessage", userID, rideID).Return(true, "")

	req, _ := http.NewRequest("GET", "/ride/123/chat/can-send", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Chat status retrieved", response["message"])

	mockService.AssertExpectations(t)
}

func TestChatHandler_GetUnreadCount_Success(t *testing.T) {
	router, mockService := setupChatTestRouter()

	userID := uint(1)
	expectedCount := 5

	mockService.On("GetUnreadChatCount", userID).Return(expectedCount, nil)

	req, _ := http.NewRequest("GET", "/chat/unread-count", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Unread count retrieved successfully", response["message"])

	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(expectedCount), data["unread_count"])

	mockService.AssertExpectations(t)
}

func TestChatHandler_GetRidesWithChats_Success(t *testing.T) {
	router, mockService := setupChatTestRouter()

	userID := uint(1)
	expectedRideIDs := []uint{101, 202, 303}

	mockService.On("GetRidesWithChats", userID).Return(expectedRideIDs, nil)

	req, _ := http.NewRequest("GET", "/chat/rides", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Rides with chats retrieved successfully", response["message"])

	data := response["data"].(map[string]interface{})
	rideIDs := data["ride_ids"].([]interface{})
	assert.Len(t, rideIDs, len(expectedRideIDs))

	mockService.AssertExpectations(t)
}

func TestChatHandler_MarkChatAsRead_Success(t *testing.T) {
	router, mockService := setupChatTestRouter()

	userID := uint(1)
	rideID := uint(123)

	mockService.On("MarkChatAsRead", userID, rideID).Return(nil)

	req, _ := http.NewRequest("POST", "/ride/123/chat/mark-read", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Chat marked as read", response["message"])

	mockService.AssertExpectations(t)
}
