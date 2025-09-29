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
	router.GET("/chat/unread-count", chatHandler.GetUnreadCount)

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

func TestChatHandler_SendMessage_Unauthenticated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockService := new(MockChatService)
	chatHandler := NewChatHandler(mockService)

	router := gin.New()
	router.POST("/ride/:id/chat/send", chatHandler.SendMessage)

	reqBody := models.SendChatMessageReq{Message: "Test"}
	bodyBytes, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/ride/123/chat/send", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
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
		},
		{
			ID:        2,
			RideID:    rideID,
			SenderID:  2,
			Message:   "Hi passenger!",
			CreatedAt: time.Now().Add(-1 * time.Minute),
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

func TestChatHandler_GetChatHistory_WithPagination(t *testing.T) {
	router, mockService := setupChatTestRouter()

	rideID := uint(123)
	userID := uint(1)
	limit := 20
	offset := 10

	expectedResp := &models.GetChatHistoryResp{
		Messages: []models.ChatMessageResp{},
		Count:    0,
	}

	mockService.On("GetChatHistory", userID, rideID, limit, offset).Return(expectedResp, nil)

	req, _ := http.NewRequest("GET", "/ride/123/chat/history?limit=20&offset=10", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	mockService.AssertExpectations(t)
}

func TestChatHandler_GetChatHistory_InvalidRideID(t *testing.T) {
	router, _ := setupChatTestRouter()

	req, _ := http.NewRequest("GET", "/ride/invalid/chat/history", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Invalid ride ID", response["error"])
}

func TestChatHandler_GetChatHistory_ServiceError(t *testing.T) {
	router, mockService := setupChatTestRouter()

	rideID := uint(123)
	userID := uint(1)
	limit := 50
	offset := 0

	mockService.On("GetChatHistory", userID, rideID, limit, offset).Return((*models.GetChatHistoryResp)(nil), errors.New("service error"))

	req, _ := http.NewRequest("GET", "/ride/123/chat/history", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	mockService.AssertExpectations(t)
}

func TestChatHandler_GetChatHistory_Unauthenticated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockService := new(MockChatService)
	chatHandler := NewChatHandler(mockService)

	router := gin.New()
	router.GET("/ride/:id/chat/history", chatHandler.GetChatHistory)

	req, _ := http.NewRequest("GET", "/ride/123/chat/history", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
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

func TestChatHandler_GetUnreadCount_ServiceError(t *testing.T) {
	router, mockService := setupChatTestRouter()

	userID := uint(1)

	mockService.On("GetUnreadChatCount", userID).Return(0, errors.New("service error"))

	req, _ := http.NewRequest("GET", "/chat/unread-count", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Failed to get unread count", response["error"])

	mockService.AssertExpectations(t)
}

func TestChatHandler_GetUnreadCount_Unauthenticated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockService := new(MockChatService)
	chatHandler := NewChatHandler(mockService)

	router := gin.New()
	router.GET("/chat/unread-count", chatHandler.GetUnreadCount)

	req, _ := http.NewRequest("GET", "/chat/unread-count", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "User not authenticated", response["error"])
}

func TestChatHandler_SendMessage_InvalidJSON(t *testing.T) {
	router, _ := setupChatTestRouter()

	invalidJSON := `{"message": "test",}`

	req, _ := http.NewRequest("POST", "/ride/123/chat/send", bytes.NewBufferString(invalidJSON))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestChatHandler_GetChatHistory_DefaultPagination(t *testing.T) {
	router, mockService := setupChatTestRouter()

	rideID := uint(123)
	userID := uint(1)
	defaultLimit := 50
	defaultOffset := 0

	expectedResp := &models.GetChatHistoryResp{
		Messages: []models.ChatMessageResp{},
		Count:    0,
	}

	mockService.On("GetChatHistory", userID, rideID, defaultLimit, defaultOffset).Return(expectedResp, nil)

	req, _ := http.NewRequest("GET", "/ride/123/chat/history", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	mockService.AssertExpectations(t)
}
