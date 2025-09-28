package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"mride-backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockSubscriptionService struct {
	mock.Mock
}

func (m *MockSubscriptionService) CreateSubscription(userID uint, req models.CreateSubscriptionReq) (*models.SubscriptionResp, error) {
	args := m.Called(userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SubscriptionResp), args.Error(1)
}

func (m *MockSubscriptionService) GetSubscription(subscriptionID uint, userID *uint) (*models.SubscriptionResp, error) {
	args := m.Called(subscriptionID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SubscriptionResp), args.Error(1)
}

func (m *MockSubscriptionService) UpdateSubscription(userID, subscriptionID uint, req models.UpdateSubscriptionReq) error {
	args := m.Called(userID, subscriptionID, req)
	return args.Error(0)
}

func (m *MockSubscriptionService) DeleteSubscription(userID, subscriptionID uint) error {
	args := m.Called(userID, subscriptionID)
	return args.Error(0)
}

func (m *MockSubscriptionService) SubscribeToRide(userID, subscriptionID uint) error {
	args := m.Called(userID, subscriptionID)
	return args.Error(0)
}

func (m *MockSubscriptionService) UnsubscribeFromRide(userID, subscriptionID uint) error {
	args := m.Called(userID, subscriptionID)
	return args.Error(0)
}

func (m *MockSubscriptionService) GetUserSubscriptions(userID uint) ([]models.SubscriptionResp, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.SubscriptionResp), args.Error(1)
}

func (m *MockSubscriptionService) GetUserSubscribedRides(userID uint) ([]models.SubscriptionResp, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.SubscriptionResp), args.Error(1)
}

func (m *MockSubscriptionService) SearchSubscriptions(req models.SearchSubscriptionsReq, userID *uint) ([]models.SubscriptionResp, error) {
	args := m.Called(req, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.SubscriptionResp), args.Error(1)
}

func (m *MockSubscriptionService) GetNearbySubscriptions(req models.NearbySubscriptionsReq, userID *uint) ([]models.SubscriptionResp, error) {
	args := m.Called(req, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.SubscriptionResp), args.Error(1)
}

func (m *MockSubscriptionService) GetAllSubscriptions(userID *uint) ([]models.SubscriptionResp, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.SubscriptionResp), args.Error(1)
}

func (m *MockSubscriptionService) GetSubscriptionStats() (*models.SubscriptionStatsResp, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SubscriptionStatsResp), args.Error(1)
}

func (m *MockSubscriptionService) ToggleNotifications(userID, subscriptionID uint, enabled bool) error {
	args := m.Called(userID, subscriptionID, enabled)
	return args.Error(0)
}

func (m *MockSubscriptionService) SendDailyNotifications() error {
	args := m.Called()
	return args.Error(0)
}

func setupSubscriptionHandlerRouter() (*gin.Engine, *MockSubscriptionService) {
	gin.SetMode(gin.TestMode)
	mockSvc := &MockSubscriptionService{}
	handler := NewSubscriptionHandler(mockSvc)

	r := gin.New()
	r.POST("/subscriptions", func(c *gin.Context) {
		c.Set("user_id", 1)
		handler.CreateSubscription(c)
	})
	r.GET("/subscriptions/:id", handler.GetSubscription)
	r.PUT("/subscriptions/:id", func(c *gin.Context) {
		c.Set("user_id", 1)
		handler.UpdateSubscription(c)
	})
	r.DELETE("/subscriptions/:id", func(c *gin.Context) {
		c.Set("user_id", 1)
		handler.DeleteSubscription(c)
	})

	return r, mockSvc
}

func TestSubscriptionHandler_CreateSubscription_Success(t *testing.T) {
	r, mockSvc := setupSubscriptionHandlerRouter()

	subscription := models.RideSubscription{
		ID:            1,
		Title:         "Daily Commute",
		FromLocation:  "Mumbai",
		ToLocation:    "Pune",
		DepartureTime: "09:00",
		RecurringDays: "Monday,Tuesday",
	}

	resp := &models.SubscriptionResp{
		Subscription: subscription,
		Driver:       models.User{ID: 1, FullName: "John Doe"},
	}

	mockSvc.On("CreateSubscription", uint(1), mock.AnythingOfType("models.CreateSubscriptionReq")).Return(resp, nil)

	reqBody := models.CreateSubscriptionReq{
		Title:            "Daily Commute",
		Description:      "Regular office commute",
		CarNumber:        "ABC123",
		CarModel:         "Honda Civic",
		PassengerCount:   3,
		FromLocation:     "Mumbai",
		ToLocation:       "Pune",
		DepartureTime:    "09:00",
		RecurringDays:    []string{"Monday", "Tuesday"},
		StartDate:        time.Now().Add(24 * time.Hour),
		MaxSubscribers:   5,
		NotificationTime: 60,
	}

	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/subscriptions", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Logf("Response body: %s", w.Body.String())
		t.Logf("Response code: %d", w.Code)
	}

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestSubscriptionHandler_CreateSubscription_SameLocation(t *testing.T) {
	r, _ := setupSubscriptionHandlerRouter()

	reqBody := models.CreateSubscriptionReq{
		Title:          "Test",
		CarNumber:      "ABC123",
		CarModel:       "Honda",
		PassengerCount: 2,
		FromLocation:   "Mumbai",
		ToLocation:     "Mumbai",
		DepartureTime:  "09:00",
		RecurringDays:  []string{"Monday"},
		StartDate:      time.Now().Add(24 * time.Hour),
	}

	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/subscriptions", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSubscriptionHandler_GetSubscription_Success(t *testing.T) {
	r, mockSvc := setupSubscriptionHandlerRouter()

	subscription := models.RideSubscription{ID: 1, FromLocation: "Mumbai", ToLocation: "Pune"}
	resp := &models.SubscriptionResp{
		Subscription: subscription,
		Driver:       models.User{ID: 1, FullName: "John Doe"},
	}

	mockSvc.On("GetSubscription", uint(1), (*uint)(nil)).Return(resp, nil)

	req := httptest.NewRequest("GET", "/subscriptions/1", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}
