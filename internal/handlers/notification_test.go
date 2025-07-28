package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func ErrJSON(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, gin.H{
		"success": false,
		"message": message,
	})
}

func SuccJSON(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": message,
		"data":    data,
	})
}

type NotificationServiceInterface interface {
	GetUserNotifications(userID uint, limit, offset int) (interface{}, error)
	MarkAsRead(userID, notificationID uint) error
	MarkAllAsRead(userID uint) error
	GetUnreadCount(userID uint) (int, error)
}

type MockNotificationSvc struct {
	mock.Mock
}

func (m *MockNotificationSvc) GetUserNotifications(userID uint, limit, offset int) (interface{}, error) {
	args := m.Called(userID, limit, offset)
	return args.Get(0), args.Error(1)
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

type TestNotificationHandler struct {
	notificationSvc NotificationServiceInterface
}

func NewTestNotificationHandler(notificationSvc NotificationServiceInterface) *TestNotificationHandler {
	return &TestNotificationHandler{
		notificationSvc: notificationSvc,
	}
}
func (h *TestNotificationHandler) GetNotifications(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		ErrJSON(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	resp, err := h.notificationSvc.GetUserNotifications(uint(userID.(int)), limit, offset)
	if err != nil {
		ErrJSON(c, http.StatusInternalServerError, "Failed to fetch notifications")
		return
	}

	SuccJSON(c, "Notifications retrieved successfully", resp)
}
func (h *TestNotificationHandler) MarkAsRead(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		ErrJSON(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	notificationIDStr := c.Param("id")
	notificationID, err := strconv.Atoi(notificationIDStr)
	if err != nil {
		ErrJSON(c, http.StatusBadRequest, "Invalid notification ID")
		return
	}

	err = h.notificationSvc.MarkAsRead(uint(userID.(int)), uint(notificationID))
	if err != nil {
		ErrJSON(c, http.StatusInternalServerError, "Failed to mark notification as read")
		return
	}

	SuccJSON(c, "Notification marked as read", nil)
}
func (h *TestNotificationHandler) MarkAllAsRead(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		ErrJSON(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	err := h.notificationSvc.MarkAllAsRead(uint(userID.(int)))
	if err != nil {
		ErrJSON(c, http.StatusInternalServerError, "Failed to mark all notifications as read")
		return
	}

	SuccJSON(c, "All notifications marked as read", nil)
}
func (h *TestNotificationHandler) GetUnreadCount(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		ErrJSON(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	count, err := h.notificationSvc.GetUnreadCount(uint(userID.(int)))
	if err != nil {
		ErrJSON(c, http.StatusInternalServerError, "Failed to get unread count")
		return
	}

	SuccJSON(c, "Unread count retrieved successfully", map[string]int{"unread_count": count})
}
func setupAuthenticatedHandler(userID int) (*gin.Engine, *MockNotificationSvc) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	mockSvc := new(MockNotificationSvc)
	handler := NewTestNotificationHandler(mockSvc)

	r.Use(func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	})

	api := r.Group("/api")
	{
		api.GET("/notifications", handler.GetNotifications)
		api.PUT("/notifications/:id/read", handler.MarkAsRead)
		api.PUT("/notifications/read-all", handler.MarkAllAsRead)
		api.GET("/notifications/unread-count", handler.GetUnreadCount)
	}

	return r, mockSvc
}
func setupUnauthenticatedHandler() (*gin.Engine, *MockNotificationSvc) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	mockSvc := new(MockNotificationSvc)
	handler := NewTestNotificationHandler(mockSvc)

	api := r.Group("/api")
	{
		api.GET("/notifications", handler.GetNotifications)
		api.PUT("/notifications/:id/read", handler.MarkAsRead)
		api.PUT("/notifications/read-all", handler.MarkAllAsRead)
		api.GET("/notifications/unread-count", handler.GetUnreadCount)
	}

	return r, mockSvc
}
func TestGetNotifications(t *testing.T) {
	t.Run("Success with default pagination", func(t *testing.T) {
		r, mockSvc := setupAuthenticatedHandler(1)
		mockNotifications := []map[string]interface{}{
			{"id": 1, "message": "Test notification 1"},
			{"id": 2, "message": "Test notification 2"},
		}

		mockSvc.On("GetUserNotifications", uint(1), 20, 0).Return(mockNotifications, nil).Once()

		req, _ := http.NewRequest("GET", "/api/notifications", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Errorf("Failed to unmarshal response: %v", err)
		}
		assert.Equal(t, "Notifications retrieved successfully", response["message"])
		assert.NotNil(t, response["data"])

		mockSvc.AssertExpectations(t)
	})

	t.Run("Success with custom pagination", func(t *testing.T) {
		r, mockSvc := setupAuthenticatedHandler(1)
		mockNotifications := []map[string]interface{}{
			{"id": 3, "message": "Test notification 3"},
		}

		mockSvc.On("GetUserNotifications", uint(1), 10, 5).Return(mockNotifications, nil).Once()

		req, _ := http.NewRequest("GET", "/api/notifications?limit=10&offset=5", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("User not authenticated", func(t *testing.T) {
		r, _ := setupUnauthenticatedHandler()
		req, _ := http.NewRequest("GET", "/api/notifications", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Errorf("Failed to unmarshal response: %v", err)
		}
		assert.Equal(t, "User not authenticated", response["message"])
	})

	t.Run("Service error", func(t *testing.T) {
		r, mockSvc := setupAuthenticatedHandler(1)
		mockSvc.On("GetUserNotifications", uint(1), 20, 0).Return(nil, assert.AnError).Once()

		req, _ := http.NewRequest("GET", "/api/notifications", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Errorf("Failed to unmarshal response: %v", err)
		}
		assert.Equal(t, "Failed to fetch notifications", response["message"])

		mockSvc.AssertExpectations(t)
	})

	t.Run("Invalid pagination parameters", func(t *testing.T) {
		r, mockSvc := setupAuthenticatedHandler(1)
		mockSvc.On("GetUserNotifications", uint(1), 0, 0).Return([]interface{}{}, nil).Once()

		req, _ := http.NewRequest("GET", "/api/notifications?limit=invalid&offset=invalid", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockSvc.AssertExpectations(t)
	})
}
func TestMarkAsRead(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		r, mockSvc := setupAuthenticatedHandler(1)
		mockSvc.On("MarkAsRead", uint(1), uint(123)).Return(nil).Once()

		req, _ := http.NewRequest("PUT", "/api/notifications/123/read", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Errorf("Failed to unmarshal response: %v", err)
		}
		assert.Equal(t, "Notification marked as read", response["message"])

		mockSvc.AssertExpectations(t)
	})

	t.Run("User not authenticated", func(t *testing.T) {
		r, _ := setupUnauthenticatedHandler()
		req, _ := http.NewRequest("PUT", "/api/notifications/123/read", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Errorf("Failed to unmarshal response: %v", err)
		}
		assert.Equal(t, "User not authenticated", response["message"])
	})

	t.Run("Invalid notification ID", func(t *testing.T) {
		r, _ := setupAuthenticatedHandler(1)
		req, _ := http.NewRequest("PUT", "/api/notifications/invalid/read", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Errorf("Failed to unmarshal response: %v", err)
		}
		assert.Equal(t, "Invalid notification ID", response["message"])
	})

	t.Run("Service error", func(t *testing.T) {
		r, mockSvc := setupAuthenticatedHandler(1)
		mockSvc.On("MarkAsRead", uint(1), uint(123)).Return(assert.AnError).Once()

		req, _ := http.NewRequest("PUT", "/api/notifications/123/read", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Errorf("Failed to unmarshal response: %v", err)
		}
		assert.Equal(t, "Failed to mark notification as read", response["message"])

		mockSvc.AssertExpectations(t)
	})

	t.Run("Zero notification ID", func(t *testing.T) {
		r, mockSvc := setupAuthenticatedHandler(1)
		mockSvc.On("MarkAsRead", uint(1), uint(0)).Return(nil).Once()

		req, _ := http.NewRequest("PUT", "/api/notifications/0/read", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockSvc.AssertExpectations(t)
	})
}
func TestMarkAllAsRead(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		r, mockSvc := setupAuthenticatedHandler(1)
		mockSvc.On("MarkAllAsRead", uint(1)).Return(nil).Once()

		req, _ := http.NewRequest("PUT", "/api/notifications/read-all", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Errorf("Failed to unmarshal response: %v", err)
		}
		assert.Equal(t, "All notifications marked as read", response["message"])

		mockSvc.AssertExpectations(t)
	})

	t.Run("User not authenticated", func(t *testing.T) {
		r, _ := setupUnauthenticatedHandler()
		req, _ := http.NewRequest("PUT", "/api/notifications/read-all", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Errorf("Failed to unmarshal response: %v", err)
		}
		assert.Equal(t, "User not authenticated", response["message"])
	})

	t.Run("Service error", func(t *testing.T) {
		r, mockSvc := setupAuthenticatedHandler(1)
		mockSvc.On("MarkAllAsRead", uint(1)).Return(assert.AnError).Once()

		req, _ := http.NewRequest("PUT", "/api/notifications/read-all", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Errorf("Failed to unmarshal response: %v", err)
		}
		assert.Equal(t, "Failed to mark all notifications as read", response["message"])

		mockSvc.AssertExpectations(t)
	})
}
func TestGetUnreadCount(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		r, mockSvc := setupAuthenticatedHandler(1)
		mockSvc.On("GetUnreadCount", uint(1)).Return(5, nil).Once()

		req, _ := http.NewRequest("GET", "/api/notifications/unread-count", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Errorf("Failed to unmarshal response: %v", err)
		}
		assert.Equal(t, "Unread count retrieved successfully", response["message"])

		if data, ok := response["data"].(map[string]interface{}); ok {
			assert.Equal(t, float64(5), data["unread_count"])
		} else {
			t.Fatal("Expected data to be a map[string]interface{}")
		}

		mockSvc.AssertExpectations(t)
	})

	t.Run("Success with zero count", func(t *testing.T) {
		r, mockSvc := setupAuthenticatedHandler(1)
		mockSvc.On("GetUnreadCount", uint(1)).Return(0, nil).Once()

		req, _ := http.NewRequest("GET", "/api/notifications/unread-count", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Errorf("Failed to unmarshal response: %v", err)
		}
		if data, ok := response["data"].(map[string]interface{}); ok {
			assert.Equal(t, float64(0), data["unread_count"])
		} else {
			t.Fatal("Expected data to be a map[string]interface{}")
		}

		mockSvc.AssertExpectations(t)
	})

	t.Run("User not authenticated", func(t *testing.T) {
		r, _ := setupUnauthenticatedHandler()
		req, _ := http.NewRequest("GET", "/api/notifications/unread-count", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Errorf("Failed to unmarshal response: %v", err)
		}
		assert.Equal(t, "User not authenticated", response["message"])
	})

	t.Run("Service error", func(t *testing.T) {
		r, mockSvc := setupAuthenticatedHandler(1)
		mockSvc.On("GetUnreadCount", uint(1)).Return(0, assert.AnError).Once()

		req, _ := http.NewRequest("GET", "/api/notifications/unread-count", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Errorf("Failed to unmarshal response: %v", err)
		}
		assert.Equal(t, "Failed to get unread count", response["message"])

		mockSvc.AssertExpectations(t)
	})
}
func TestNewNotificationHandler(t *testing.T) {
	mockSvc := new(MockNotificationSvc)
	handler := NewTestNotificationHandler(mockSvc)

	assert.NotNil(t, handler)
	assert.Equal(t, mockSvc, handler.notificationSvc)
}
func TestNotificationHandlerIntegration(t *testing.T) {
	r, mockSvc := setupAuthenticatedHandler(1)

	t.Run("Complete workflow", func(t *testing.T) {

		mockSvc.On("GetUnreadCount", uint(1)).Return(3, nil).Once()

		req1, _ := http.NewRequest("GET", "/api/notifications/unread-count", nil)
		w1 := httptest.NewRecorder()
		r.ServeHTTP(w1, req1)
		assert.Equal(t, http.StatusOK, w1.Code)

		mockNotifications := []map[string]interface{}{
			{"id": 1, "message": "Test notification 1", "read": false},
			{"id": 2, "message": "Test notification 2", "read": false},
		}
		mockSvc.On("GetUserNotifications", uint(1), 20, 0).Return(mockNotifications, nil).Once()

		req2, _ := http.NewRequest("GET", "/api/notifications", nil)
		w2 := httptest.NewRecorder()
		r.ServeHTTP(w2, req2)
		assert.Equal(t, http.StatusOK, w2.Code)

		mockSvc.On("MarkAsRead", uint(1), uint(1)).Return(nil).Once()

		req3, _ := http.NewRequest("PUT", "/api/notifications/1/read", nil)
		w3 := httptest.NewRecorder()
		r.ServeHTTP(w3, req3)
		assert.Equal(t, http.StatusOK, w3.Code)

		mockSvc.On("MarkAllAsRead", uint(1)).Return(nil).Once()

		req4, _ := http.NewRequest("PUT", "/api/notifications/read-all", nil)
		w4 := httptest.NewRecorder()
		r.ServeHTTP(w4, req4)
		assert.Equal(t, http.StatusOK, w4.Code)

		mockSvc.AssertExpectations(t)
	})
}
