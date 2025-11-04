package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"mride-backend/internal/models"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockSubscriptionChatService struct {
	mock.Mock
}

func (m *MockSubscriptionChatService) SendChatMessage(userID, subscriptionID uint, message string) (*models.ChatMessageResp, error) {
	args := m.Called(userID, subscriptionID, message)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ChatMessageResp), args.Error(1)
}

func (m *MockSubscriptionChatService) GetChatHistory(userID, subscriptionID uint, limit, offset int) (*models.GetChatHistoryResp, error) {
	args := m.Called(userID, subscriptionID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.GetChatHistoryResp), args.Error(1)
}

func (m *MockSubscriptionChatService) MarkChatAsRead(userID, subscriptionID uint) error {
	args := m.Called(userID, subscriptionID)
	return args.Error(0)
}

func (m *MockSubscriptionChatService) GetSubscriptionChats(userID uint) ([]models.SubscriptionChatRoomInfo, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.SubscriptionChatRoomInfo), args.Error(1)
}

func (m *MockSubscriptionChatService) CanSendMessage(userID, subscriptionID uint) (bool, string) {
	args := m.Called(userID, subscriptionID)
	return args.Bool(0), args.String(1)
}

func setupGinTest() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	return router
}

func TestSendMessage_Success(t *testing.T) {
	mockSvc := new(MockSubscriptionChatService)
	handler := NewSubscriptionChatHandler(mockSvc)
	router := setupGinTest()

	router.POST("/subscription/:id/chat/send", func(c *gin.Context) {
		c.Set("user_id", 1)
		handler.SendMessage(c)
	})

	expectedResp := &models.ChatMessageResp{
		ID:        1,
		RideID:    1,
		SenderID:  1,
		Message:   "Test message",
		CreatedAt: time.Now(),
		IsMine:    true,
		IsRead:    false,
	}

	mockSvc.On("SendChatMessage", uint(1), uint(1), "Test message").Return(expectedResp, nil)

	reqBody := map[string]string{"message": "Test message"}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/subscription/1/chat/send", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestSendMessage_InvalidID(t *testing.T) {
	mockSvc := new(MockSubscriptionChatService)
	handler := NewSubscriptionChatHandler(mockSvc)
	router := setupGinTest()

	router.POST("/subscription/:id/chat/send", func(c *gin.Context) {
		c.Set("user_id", 1)
		handler.SendMessage(c)
	})

	reqBody := map[string]string{"message": "Test message"}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/subscription/invalid/chat/send", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSendMessage_Unauthorized(t *testing.T) {
	mockSvc := new(MockSubscriptionChatService)
	handler := NewSubscriptionChatHandler(mockSvc)
	router := setupGinTest()

	router.POST("/subscription/:id/chat/send", handler.SendMessage)

	reqBody := map[string]string{"message": "Test message"}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/subscription/1/chat/send", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetChatHistory_Success(t *testing.T) {
	mockSvc := new(MockSubscriptionChatService)
	handler := NewSubscriptionChatHandler(mockSvc)
	router := setupGinTest()

	router.GET("/subscription/:id/chat/history", func(c *gin.Context) {
		c.Set("user_id", 1)
		handler.GetChatHistory(c)
	})

	expectedResp := &models.GetChatHistoryResp{
		Messages: []models.ChatMessageResp{
			{
				ID:        1,
				RideID:    1,
				SenderID:  1,
				Message:   "Test",
				CreatedAt: time.Now(),
			},
		},
		Count: 1,
	}

	mockSvc.On("GetChatHistory", uint(1), uint(1), 50, 0).Return(expectedResp, nil)

	req := httptest.NewRequest(http.MethodGet, "/subscription/1/chat/history", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestGetChatHistory_ServiceError(t *testing.T) {
	mockSvc := new(MockSubscriptionChatService)
	handler := NewSubscriptionChatHandler(mockSvc)
	router := setupGinTest()

	router.GET("/subscription/:id/chat/history", func(c *gin.Context) {
		c.Set("user_id", 1)
		handler.GetChatHistory(c)
	})

	mockSvc.On("GetChatHistory", uint(1), uint(1), 50, 0).
		Return((*models.GetChatHistoryResp)(nil), errors.New("service error"))

	req := httptest.NewRequest(http.MethodGet, "/subscription/1/chat/history", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestMarkChatAsRead_Success(t *testing.T) {
	mockSvc := new(MockSubscriptionChatService)
	handler := NewSubscriptionChatHandler(mockSvc)
	router := setupGinTest()

	router.POST("/subscription/:id/chat/mark-read", func(c *gin.Context) {
		c.Set("user_id", 1)
		handler.MarkChatAsRead(c)
	})

	mockSvc.On("MarkChatAsRead", uint(1), uint(1)).Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/subscription/1/chat/mark-read", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestGetSubscriptionChats_Success(t *testing.T) {
	mockSvc := new(MockSubscriptionChatService)
	handler := NewSubscriptionChatHandler(mockSvc)
	router := setupGinTest()

	router.GET("/subscription-chats", func(c *gin.Context) {
		c.Set("user_id", 1)
		handler.GetSubscriptionChats(c)
	})

	expectedChats := []models.SubscriptionChatRoomInfo{
		{
			SubscriptionID:  1,
			OtherUserID:     2,
			OtherUserName:   "Test User",
			LastMessage:     "Hello",
			LastMessageTime: time.Now(),
			UnreadCount:     0,
		},
	}

	mockSvc.On("GetSubscriptionChats", uint(1)).Return(expectedChats, nil)

	req := httptest.NewRequest(http.MethodGet, "/subscription-chats", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestCheckCanSendMessage_Success(t *testing.T) {
	mockSvc := new(MockSubscriptionChatService)
	handler := NewSubscriptionChatHandler(mockSvc)
	router := setupGinTest()

	router.GET("/subscription/:id/chat/can-send", func(c *gin.Context) {
		c.Set("user_id", 1)
		handler.CheckCanSendMessage(c)
	})

	mockSvc.On("CanSendMessage", uint(1), uint(1)).Return(true, "")

	req := httptest.NewRequest(http.MethodGet, "/subscription/1/chat/can-send", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}
