package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"mride-backend/internal/models"
	"mride-backend/internal/services"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockPushNotificationSvc struct {
	mock.Mock
}

func (m *MockPushNotificationSvc) RegisterDeviceToken(userID uint, token, platform string) error {
	args := m.Called(userID, token, platform)
	return args.Error(0)
}

func (m *MockPushNotificationSvc) UnregisterDeviceToken(userID uint, token string) error {
	args := m.Called(userID, token)
	return args.Error(0)
}

func (m *MockPushNotificationSvc) GetUserDeviceTokens(userID uint) ([]services.DeviceToken, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]services.DeviceToken), args.Error(1)
}

func (m *MockPushNotificationSvc) SendPushNotification(userID uint, notification *models.Notification) error {
	args := m.Called(userID, notification)
	return args.Error(0)
}

func (m *MockPushNotificationSvc) ShouldSendPushNotification(userID uint, notificationType string) bool {
	args := m.Called(userID, notificationType)
	return args.Bool(0)
}

func (m *MockPushNotificationSvc) UpdateNotificationPreferences(userID uint, prefs *services.NotificationPreference) error {
	args := m.Called(userID, prefs)
	return args.Error(0)
}

func (m *MockPushNotificationSvc) GetNotificationPreferences(userID uint) (*services.NotificationPreference, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.NotificationPreference), args.Error(1)
}

func (m *MockPushNotificationSvc) SendBulkNotification(userIDs []uint, title, message string, data map[string]interface{}) error {
	args := m.Called(userIDs, title, message, data)
	return args.Error(0)
}

func (m *MockPushNotificationSvc) CleanupInactiveTokens() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockPushNotificationSvc) GetPushNotificationStats() (map[string]interface{}, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func setupPushTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestPushRegisterDeviceToken(t *testing.T) {
	t.Run("Success - register device token", func(t *testing.T) {
		mockSvc := new(MockPushNotificationSvc)
		handler := NewPushNotificationHandler(mockSvc)

		mockSvc.On("RegisterDeviceToken", uint(1), "test_token_123", "android").Return(nil)

		router := setupPushTestRouter()
		router.POST("/push/register", func(c *gin.Context) {
			c.Set("user_id", 1)
			handler.RegisterDeviceToken(c)
		})

		body := map[string]string{
			"token":    "test_token_123",
			"platform": "android",
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/push/register", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("Error - invalid platform", func(t *testing.T) {
		mockSvc := new(MockPushNotificationSvc)
		handler := NewPushNotificationHandler(mockSvc)

		router := setupPushTestRouter()
		router.POST("/push/register", func(c *gin.Context) {
			c.Set("user_id", 1)
			handler.RegisterDeviceToken(c)
		})

		body := map[string]string{
			"token":    "test_token_123",
			"platform": "invalid_platform",
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/push/register", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Error - missing token", func(t *testing.T) {
		mockSvc := new(MockPushNotificationSvc)
		handler := NewPushNotificationHandler(mockSvc)

		router := setupPushTestRouter()
		router.POST("/push/register", func(c *gin.Context) {
			c.Set("user_id", 1)
			handler.RegisterDeviceToken(c)
		})

		body := map[string]string{
			"platform": "android",
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/push/register", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Error - no user_id in context", func(t *testing.T) {
		mockSvc := new(MockPushNotificationSvc)
		handler := NewPushNotificationHandler(mockSvc)

		router := setupPushTestRouter()
		router.POST("/push/register", handler.RegisterDeviceToken)

		body := map[string]string{
			"token":    "test_token_123",
			"platform": "android",
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/push/register", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Error - service error", func(t *testing.T) {
		mockSvc := new(MockPushNotificationSvc)
		handler := NewPushNotificationHandler(mockSvc)

		mockSvc.On("RegisterDeviceToken", uint(1), "test_token_123", "ios").
			Return(errors.New("database error"))

		router := setupPushTestRouter()
		router.POST("/push/register", func(c *gin.Context) {
			c.Set("user_id", 1)
			handler.RegisterDeviceToken(c)
		})

		body := map[string]string{
			"token":    "test_token_123",
			"platform": "ios",
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/push/register", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		mockSvc.AssertExpectations(t)
	})
}

func TestPushUnregisterDeviceToken(t *testing.T) {
	t.Run("Success - unregister device token", func(t *testing.T) {
		mockSvc := new(MockPushNotificationSvc)
		handler := NewPushNotificationHandler(mockSvc)

		mockSvc.On("UnregisterDeviceToken", uint(1), "test_token_123").Return(nil)

		router := setupPushTestRouter()
		router.POST("/push/unregister", func(c *gin.Context) {
			c.Set("user_id", 1)
			handler.UnregisterDeviceToken(c)
		})

		body := map[string]string{
			"token": "test_token_123",
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/push/unregister", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("Error - missing token", func(t *testing.T) {
		mockSvc := new(MockPushNotificationSvc)
		handler := NewPushNotificationHandler(mockSvc)

		router := setupPushTestRouter()
		router.POST("/push/unregister", func(c *gin.Context) {
			c.Set("user_id", 1)
			handler.UnregisterDeviceToken(c)
		})

		body := map[string]string{}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/push/unregister", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestPushGetDeviceTokens(t *testing.T) {
	t.Run("Success - get device tokens", func(t *testing.T) {
		mockSvc := new(MockPushNotificationSvc)
		handler := NewPushNotificationHandler(mockSvc)

		mockTokens := []services.DeviceToken{
			{ID: 1, UserID: 1, Token: "token1", Platform: "android"},
			{ID: 2, UserID: 1, Token: "token2", Platform: "ios"},
		}
		mockSvc.On("GetUserDeviceTokens", uint(1)).Return(mockTokens, nil)

		router := setupPushTestRouter()
		router.GET("/push/devices", func(c *gin.Context) {
			c.Set("user_id", 1)
			handler.GetDeviceTokens(c)
		})

		req, _ := http.NewRequest("GET", "/push/devices", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		data := response["data"].(map[string]interface{})
		assert.Equal(t, float64(2), data["count"])
		mockSvc.AssertExpectations(t)
	})

	t.Run("Error - service error", func(t *testing.T) {
		mockSvc := new(MockPushNotificationSvc)
		handler := NewPushNotificationHandler(mockSvc)

		mockSvc.On("GetUserDeviceTokens", uint(1)).Return(nil, errors.New("database error"))

		router := setupPushTestRouter()
		router.GET("/push/devices", func(c *gin.Context) {
			c.Set("user_id", 1)
			handler.GetDeviceTokens(c)
		})

		req, _ := http.NewRequest("GET", "/push/devices", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		mockSvc.AssertExpectations(t)
	})
}

func TestPushUpdateNotificationPreferences(t *testing.T) {
	t.Run("Success - update preferences", func(t *testing.T) {
		mockSvc := new(MockPushNotificationSvc)
		handler := NewPushNotificationHandler(mockSvc)

		mockSvc.On("UpdateNotificationPreferences", uint(1), mock.AnythingOfType("*services.NotificationPreference")).
			Return(nil)

		router := setupPushTestRouter()
		router.PUT("/push/preferences", func(c *gin.Context) {
			c.Set("user_id", 1)
			handler.UpdateNotificationPreferences(c)
		})

		body := map[string]interface{}{
			"push_enabled":        true,
			"ride_join_enabled":   false,
			"quiet_hours_enabled": true,
			"quiet_hours_start":   "22:00",
			"quiet_hours_end":     "08:00",
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("PUT", "/push/preferences", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("Error - invalid JSON", func(t *testing.T) {
		mockSvc := new(MockPushNotificationSvc)
		handler := NewPushNotificationHandler(mockSvc)

		router := setupPushTestRouter()
		router.PUT("/push/preferences", func(c *gin.Context) {
			c.Set("user_id", 1)
			handler.UpdateNotificationPreferences(c)
		})

		req, _ := http.NewRequest("PUT", "/push/preferences", bytes.NewBuffer([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestPushGetNotificationPreferences(t *testing.T) {
	t.Run("Success - get preferences", func(t *testing.T) {
		mockSvc := new(MockPushNotificationSvc)
		handler := NewPushNotificationHandler(mockSvc)

		mockPrefs := &services.NotificationPreference{
			ID:                1,
			UserID:            1,
			PushEnabled:       true,
			RideJoinEnabled:   true,
			QuietHoursEnabled: false,
		}
		mockSvc.On("GetNotificationPreferences", uint(1)).Return(mockPrefs, nil)

		router := setupPushTestRouter()
		router.GET("/push/preferences", func(c *gin.Context) {
			c.Set("user_id", 1)
			handler.GetNotificationPreferences(c)
		})

		req, _ := http.NewRequest("GET", "/push/preferences", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		data := response["data"].(map[string]interface{})
		assert.Equal(t, true, data["push_enabled"])
		mockSvc.AssertExpectations(t)
	})
}

func TestPushGetPushNotificationStats(t *testing.T) {
	t.Run("Success - get stats", func(t *testing.T) {
		mockSvc := new(MockPushNotificationSvc)
		handler := NewPushNotificationHandler(mockSvc)

		mockStats := map[string]interface{}{
			"total_tokens":  100,
			"active_tokens": 75,
			"total_sent":    1000,
			"total_failed":  10,
			"success_rate":  99.0,
		}
		mockSvc.On("GetPushNotificationStats").Return(mockStats, nil)

		router := setupPushTestRouter()
		router.GET("/push/stats", handler.GetPushNotificationStats)

		req, _ := http.NewRequest("GET", "/push/stats", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		data := response["data"].(map[string]interface{})
		assert.Equal(t, float64(100), data["total_tokens"])
		assert.Equal(t, float64(99.0), data["success_rate"])
		mockSvc.AssertExpectations(t)
	})

	t.Run("Error - service error", func(t *testing.T) {
		mockSvc := new(MockPushNotificationSvc)
		handler := NewPushNotificationHandler(mockSvc)

		mockSvc.On("GetPushNotificationStats").Return(nil, errors.New("database error"))

		router := setupPushTestRouter()
		router.GET("/push/stats", handler.GetPushNotificationStats)

		req, _ := http.NewRequest("GET", "/push/stats", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		mockSvc.AssertExpectations(t)
	})
}

func TestPushTestPushNotification(t *testing.T) {
	t.Run("Success - test notification", func(t *testing.T) {
		mockSvc := new(MockPushNotificationSvc)
		handler := NewPushNotificationHandler(mockSvc)

		router := setupPushTestRouter()
		router.POST("/push/test", func(c *gin.Context) {
			c.Set("user_id", 1)
			handler.TestPushNotification(c)
		})

		body := map[string]string{
			"title":   "Test Notification",
			"message": "This is a test",
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/push/test", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Error - missing title", func(t *testing.T) {
		mockSvc := new(MockPushNotificationSvc)
		handler := NewPushNotificationHandler(mockSvc)

		router := setupPushTestRouter()
		router.POST("/push/test", func(c *gin.Context) {
			c.Set("user_id", 1)
			handler.TestPushNotification(c)
		})

		body := map[string]string{
			"message": "This is a test",
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/push/test", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func BenchmarkPushRegisterDeviceTokenHandler(b *testing.B) {
	mockSvc := new(MockPushNotificationSvc)
	handler := NewPushNotificationHandler(mockSvc)

	mockSvc.On("RegisterDeviceToken", mock.AnythingOfType("uint"),
		mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Return(nil)

	router := setupPushTestRouter()
	router.POST("/push/register", func(c *gin.Context) {
		c.Set("user_id", 1)
		handler.RegisterDeviceToken(c)
	})

	body := map[string]string{
		"token":    "benchmark_token",
		"platform": "android",
	}
	jsonBody, _ := json.Marshal(body)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("POST", "/push/register", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}
