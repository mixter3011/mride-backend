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

type ErrorResponse struct {
	Error string `json:"error"`
}

type SuccessResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type MockRideService struct {
	mock.Mock
}

func (m *MockRideService) CreateRide(userID uint, req models.CreateRideReq) (*models.RideResp, error) {
	args := m.Called(userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.RideResp), args.Error(1)
}

func (m *MockRideService) DeleteRide(userID, rideID uint) error {
	args := m.Called(userID, rideID)
	return args.Error(0)
}

func (m *MockRideService) GetAllUserRides(userID uint) ([]models.RideResp, error) {
	args := m.Called(userID)
	return args.Get(0).([]models.RideResp), args.Error(1)
}

func (m *MockRideService) GetRideByID(rideID uint) (*models.RideResp, error) {
	args := m.Called(rideID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.RideResp), args.Error(1)
}

func (m *MockRideService) SearchRides(from, to string, locationReq *models.SearchByLocationReq) ([]models.RideResp, error) {
	args := m.Called(from, to, locationReq)
	return args.Get(0).([]models.RideResp), args.Error(1)
}

func (m *MockRideService) GetNearbyRides(userID uint, req models.NearbyRidesReq) ([]models.RideResp, error) {
	args := m.Called(userID, req)
	return args.Get(0).([]models.RideResp), args.Error(1)
}

func (m *MockRideService) JoinRide(userID, rideID uint) error {
	args := m.Called(userID, rideID)
	return args.Error(0)
}

func (m *MockRideService) LeaveRide(userID, rideID uint) error {
	args := m.Called(userID, rideID)
	return args.Error(0)
}

func (m *MockRideService) GetAllRides() ([]models.RideResp, error) {
	args := m.Called()
	return args.Get(0).([]models.RideResp), args.Error(1)
}

func (m *MockRideService) StartRide(userID, rideID uint, req models.StartRideReq) error {
	args := m.Called(userID, rideID, req)
	return args.Error(0)
}

func (m *MockRideService) CompleteRide(userID, rideID uint) error {
	args := m.Called(userID, rideID)
	return args.Error(0)
}

func (m *MockRideService) GetRideProgress(rideID uint) (*models.RideProgressResp, error) {
	args := m.Called(rideID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.RideProgressResp), args.Error(1)
}

func (m *MockRideService) GetActiveRidesForUser(userID uint) ([]models.RideProgressResp, error) {
	args := m.Called(userID)
	return args.Get(0).([]models.RideProgressResp), args.Error(1)
}

func setupRouter(handler *RideHandler) *gin.Engine {
	router := gin.Default()
	return router
}

func setAuthContext(c *gin.Context) {
	c.Set("user_id", 123)
}

func TestRideHandler_CreateRide(t *testing.T) {
	mockSvc := new(MockRideService)
	handler := NewRideHandler(mockSvc)
	router := setupRouter(handler)
	router.Use(setAuthContext)
	router.POST("/rides", handler.CreateRide)

	t.Run("Success", func(t *testing.T) {
		reqBody := models.CreateRideReq{
			CarNumber:      "ABC123",
			CarModel:       "Model S",
			PassengerCount: 3,
			FromLocation:   "New York",
			ToLocation:     "Boston",
			DepartureTime:  time.Now().Add(time.Hour),
		}
		mockResp := &models.RideResp{}

		mockSvc.On("CreateRide", uint(123), mock.MatchedBy(func(req models.CreateRideReq) bool {
			return req.CarNumber == "ABC123" &&
				req.CarModel == "Model S" &&
				req.PassengerCount == 3 &&
				req.FromLocation == "New York" &&
				req.ToLocation == "Boston"
		})).Return(mockResp, nil)

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/rides", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("ValidationError", func(t *testing.T) {
		testCases := []struct {
			name   string
			req    models.CreateRideReq
			errMsg string
		}{
			{"SameLocations", models.CreateRideReq{
				CarNumber:      "ABC123",
				CarModel:       "Model S",
				PassengerCount: 2,
				FromLocation:   "New York",
				ToLocation:     "New York",
				DepartureTime:  time.Now().Add(time.Hour),
			}, "From and to locations cannot be the same"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				body, _ := json.Marshal(tc.req)
				req, _ := http.NewRequest("POST", "/rides", bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusBadRequest, w.Code)
				var errResp ErrorResponse
				if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
					t.Errorf("Failed to unmarshal error response: %v", err)
				}
				assert.Equal(t, tc.errMsg, errResp.Error)
			})
		}
	})

	t.Run("ServiceError", func(t *testing.T) {
		reqBody := models.CreateRideReq{
			CarNumber:      "ABC123",
			CarModel:       "Model S",
			PassengerCount: 3,
			FromLocation:   "New York",
			ToLocation:     "Boston",
			DepartureTime:  time.Now().Add(time.Hour),
		}

		mockSvc.Mock.ExpectedCalls = nil
		mockSvc.On("CreateRide", uint(123), mock.MatchedBy(func(req models.CreateRideReq) bool {
			return req.CarNumber == "ABC123"
		})).Return((*models.RideResp)(nil), errors.New("service error"))

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/rides", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestRideHandler_DeleteRide(t *testing.T) {
	mockSvc := new(MockRideService)
	handler := NewRideHandler(mockSvc)
	router := setupRouter(handler)
	router.Use(setAuthContext)
	router.DELETE("/rides/:id", handler.DeleteRide)

	t.Run("Success", func(t *testing.T) {
		mockSvc.On("DeleteRide", uint(123), uint(1)).Return(nil)

		req, _ := http.NewRequest("DELETE", "/rides/1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("InvalidID", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", "/rides/abc", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("ServiceError", func(t *testing.T) {
		mockSvc.On("DeleteRide", uint(123), uint(2)).Return(errors.New("service error"))

		req, _ := http.NewRequest("DELETE", "/rides/2", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestRideHandler_GetMyRides(t *testing.T) {
	mockSvc := new(MockRideService)
	handler := NewRideHandler(mockSvc)
	router := setupRouter(handler)
	router.Use(setAuthContext)
	router.GET("/rides/me", handler.GetMyRides)

	t.Run("Success", func(t *testing.T) {
		mockResp := []models.RideResp{}
		mockSvc.On("GetAllUserRides", uint(123)).Return(mockResp, nil)

		req, _ := http.NewRequest("GET", "/rides/me", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("ServiceError", func(t *testing.T) {
		mockSvc.Mock.ExpectedCalls = nil
		mockSvc.On("GetAllUserRides", uint(123)).Return([]models.RideResp(nil), errors.New("service error"))

		req, _ := http.NewRequest("GET", "/rides/me", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestRideHandler_GetRide(t *testing.T) {
	mockSvc := new(MockRideService)
	handler := NewRideHandler(mockSvc)
	router := setupRouter(handler)
	router.GET("/rides/:id", handler.GetRide)

	t.Run("Success", func(t *testing.T) {
		mockResp := &models.RideResp{}
		mockSvc.On("GetRideByID", uint(1)).Return(mockResp, nil)

		req, _ := http.NewRequest("GET", "/rides/1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("NotFound", func(t *testing.T) {
		mockSvc.On("GetRideByID", uint(2)).Return((*models.RideResp)(nil), errors.New("not found"))

		req, _ := http.NewRequest("GET", "/rides/2", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestRideHandler_SearchRides(t *testing.T) {
	mockSvc := new(MockRideService)
	handler := NewRideHandler(mockSvc)
	router := setupRouter(handler)
	router.GET("/rides/search", handler.SearchRides)

	t.Run("TextSearch_Success", func(t *testing.T) {
		mockResp := []models.RideResp{}
		mockSvc.On("SearchRides", "A", "B", (*models.SearchByLocationReq)(nil)).Return(mockResp, nil)

		req, _ := http.NewRequest("GET", "/rides/search?from=A&to=B", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("LocationSearch_Success", func(t *testing.T) {
		mockResp := []models.RideResp{}
		mockSvc.On("SearchRides", "", "", mock.MatchedBy(func(req *models.SearchByLocationReq) bool {
			return req != nil && req.FromLatitude == 19.0760
		})).Return(mockResp, nil)

		url := "/rides/search?from_lat=19.0760&from_lng=72.8777&to_lat=18.5204&to_lng=73.8567&radius=5"
		req, _ := http.NewRequest("GET", url, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("MissingParams", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/rides/search", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("InvalidCoordinates", func(t *testing.T) {
		url := "/rides/search?from_lat=100&from_lng=72.8777&to_lat=18.5204&to_lng=73.8567"
		req, _ := http.NewRequest("GET", url, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestRideHandler_GetNearbyRides(t *testing.T) {
	mockSvc := new(MockRideService)
	handler := NewRideHandler(mockSvc)
	router := setupRouter(handler)
	router.GET("/rides/nearby", handler.GetNearbyRides)

	validReq := models.NearbyRidesReq{
		FromLatitude:  40.0,
		FromLongitude: -74.0,
		ToLatitude:    41.0,
		ToLongitude:   -75.0,
		RadiusKM:      10,
	}

	t.Run("Success", func(t *testing.T) {
		mockResp := []models.RideResp{}
		mockSvc.On("GetNearbyRides", uint(0), validReq).Return(mockResp, nil)

		url := "/rides/nearby?from_lat=40&from_lng=-74&to_lat=41&to_lng=-75&radius=10"
		req, _ := http.NewRequest("GET", url, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("InvalidCoordinates", func(t *testing.T) {
		testCases := []struct {
			name   string
			params string
		}{
			{"InvalidFromLat", "from_lat=100&from_lng=-74&to_lat=41&to_lng=-75"},
			{"InvalidToLng", "from_lat=40&from_lng=-74&to_lat=41&to_lng=200"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				req, _ := http.NewRequest("GET", "/rides/nearby?"+tc.params, nil)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusBadRequest, w.Code)
			})
		}
	})
}

func TestRideHandler_JoinLeaveRide(t *testing.T) {
	mockSvc := new(MockRideService)
	handler := NewRideHandler(mockSvc)
	router := setupRouter(handler)
	router.Use(setAuthContext)

	router.POST("/rides/:id/join", handler.JoinRide)
	router.POST("/rides/:id/leave", handler.LeaveRide)

	t.Run("JoinSuccess", func(t *testing.T) {
		mockSvc.On("JoinRide", uint(123), uint(1)).Return(nil)

		req, _ := http.NewRequest("POST", "/rides/1/join", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("LeaveSuccess", func(t *testing.T) {
		mockSvc.On("LeaveRide", uint(123), uint(1)).Return(nil)

		req, _ := http.NewRequest("POST", "/rides/1/leave", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("InvalidID", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/rides/abc/join", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestRideHandler_StartCompleteRide(t *testing.T) {
	mockSvc := new(MockRideService)
	handler := NewRideHandler(mockSvc)
	router := setupRouter(handler)
	router.Use(setAuthContext)

	router.POST("/rides/:id/start", handler.StartRide)
	router.POST("/rides/:id/complete", handler.CompleteRide)

	t.Run("StartSuccess", func(t *testing.T) {
		mockSvc.On("StartRide", uint(123), uint(1), mock.AnythingOfType("models.StartRideReq")).Return(nil)

		reqBody := models.StartRideReq{
			EstimatedDuration: 60,
			Latitude:          40.0,
			Longitude:         -74.0,
		}

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/rides/1/start", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("CompleteSuccess", func(t *testing.T) {
		mockSvc.On("CompleteRide", uint(123), uint(1)).Return(nil)

		req, _ := http.NewRequest("POST", "/rides/1/complete", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestRideHandler_GetRideProgress(t *testing.T) {
	mockSvc := new(MockRideService)
	handler := NewRideHandler(mockSvc)
	router := setupRouter(handler)
	router.GET("/rides/:id/progress", handler.GetRideProgress)

	t.Run("Success", func(t *testing.T) {
		mockResp := &models.RideProgressResp{}
		mockSvc.On("GetRideProgress", uint(1)).Return(mockResp, nil)

		req, _ := http.NewRequest("GET", "/rides/1/progress", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestRideHandler_GetActiveRides(t *testing.T) {
	mockSvc := new(MockRideService)
	handler := NewRideHandler(mockSvc)
	router := setupRouter(handler)
	router.Use(setAuthContext)
	router.GET("/rides/active", handler.GetActiveRides)

	t.Run("Success", func(t *testing.T) {
		mockResp := []models.RideProgressResp{}
		mockSvc.On("GetActiveRidesForUser", uint(123)).Return(mockResp, nil)

		req, _ := http.NewRequest("GET", "/rides/active", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestRideHandler_GetAllRides(t *testing.T) {
	mockSvc := new(MockRideService)
	handler := NewRideHandler(mockSvc)
	router := setupRouter(handler)
	router.GET("/rides", handler.GetAllRides)

	t.Run("Success", func(t *testing.T) {
		mockResp := []models.RideResp{}
		mockSvc.On("GetAllRides").Return(mockResp, nil)

		req, _ := http.NewRequest("GET", "/rides", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("EmptyResponse", func(t *testing.T) {
		mockSvc.On("GetAllRides").Return([]models.RideResp{}, nil)

		req, _ := http.NewRequest("GET", "/rides", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp SuccessResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Errorf("Failed to unmarshal response: %v", err)
		}
		assert.Empty(t, resp.Data.([]interface{}))
	})
}
