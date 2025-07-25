package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type WebSocketSvcInterface interface {
	HandleConnection(w http.ResponseWriter, r *http.Request, userID int)
	GetOnlineUsers() []interface{}
	GetOnlineUsersFromDB() ([]interface{}, error)
	GetUserConnectionHistory(userID, limit int) ([]interface{}, error)
	GetConnectionStats() (map[string]interface{}, error)
	CleanupStaleConnections() error
}

type JWTSvcInterface interface {
	ValidToken(token string) (*Claims, error)
}

type MockWebSocketSvc struct {
	mock.Mock
}

func (m *MockWebSocketSvc) HandleConnection(w http.ResponseWriter, r *http.Request, userID int) {
	m.Called(w, r, userID)
}

func (m *MockWebSocketSvc) GetOnlineUsers() []interface{} {
	args := m.Called()
	return args.Get(0).([]interface{})
}

func (m *MockWebSocketSvc) GetOnlineUsersFromDB() ([]interface{}, error) {
	args := m.Called()
	return args.Get(0).([]interface{}), args.Error(1)
}

func (m *MockWebSocketSvc) GetUserConnectionHistory(userID, limit int) ([]interface{}, error) {
	args := m.Called(userID, limit)
	return args.Get(0).([]interface{}), args.Error(1)
}

func (m *MockWebSocketSvc) GetConnectionStats() (map[string]interface{}, error) {
	args := m.Called()
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockWebSocketSvc) CleanupStaleConnections() error {
	args := m.Called()
	return args.Error(0)
}

type MockJWTSvc struct {
	mock.Mock
}

type Claims struct {
	UserID int `json:"user_id"`
}

func (m *MockJWTSvc) ValidToken(token string) (*Claims, error) {
	args := m.Called(token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Claims), args.Error(1)
}

type TestWebSocketHandler struct {
	webSocketSvc WebSocketSvcInterface
	jwtSvc       JWTSvcInterface
}

func NewTestWebSocketHandler(webSocketSvc WebSocketSvcInterface, jwtSvc JWTSvcInterface) *TestWebSocketHandler {
	return &TestWebSocketHandler{
		webSocketSvc: webSocketSvc,
		jwtSvc:       jwtSvc,
	}
}

func (h *TestWebSocketHandler) HandleWebSocket(c *gin.Context) {
	token := ""

	authHeader := c.GetHeader("Authorization")
	if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
		token = strings.TrimPrefix(authHeader, "Bearer ")
	}

	if token == "" {
		protocols := c.GetHeader("Sec-WebSocket-Protocol")
		if protocols != "" {
			parts := strings.Split(protocols, ", ")
			for _, part := range parts {
				if strings.HasPrefix(part, "access_token.") {
					token = strings.TrimPrefix(part, "access_token.")
					break
				}
			}
		}
	}

	if token == "" {
		c.Header("Sec-WebSocket-Protocol", "access_token")
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Authentication required"})
		return
	}

	claims, err := h.jwtSvc.ValidToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid token"})
		return
	}

	userID := claims.UserID

	if c.GetHeader("Sec-WebSocket-Protocol") != "" {
		c.Header("Sec-WebSocket-Protocol", "access_token")
	}

	h.webSocketSvc.HandleConnection(c.Writer, c.Request, userID)
}

func (h *TestWebSocketHandler) GetOnlineUsers(c *gin.Context) {
	users := h.webSocketSvc.GetOnlineUsers()
	c.JSON(http.StatusOK, gin.H{
		"message": "Online users retrieved successfully",
		"data": map[string]interface{}{
			"online_users": users,
			"count":        len(users),
		},
	})
}

func (h *TestWebSocketHandler) GetOnlineUsersFromDB(c *gin.Context) {
	connections, err := h.webSocketSvc.GetOnlineUsersFromDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve online users from database"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Online users retrieved from database",
		"data": map[string]interface{}{
			"connections": connections,
			"count":       len(connections),
		},
	})
}

func (h *TestWebSocketHandler) GetUserConnectionHistory(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid user ID"})
		return
	}

	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 10
	}

	history, err := h.webSocketSvc.GetUserConnectionHistory(userID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve connection history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Connection history retrieved successfully",
		"data": map[string]interface{}{
			"history": history,
			"count":   len(history),
		},
	})
}

func (h *TestWebSocketHandler) GetConnectionStats(c *gin.Context) {
	stats, err := h.webSocketSvc.GetConnectionStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve connection statistics"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Connection statistics retrieved successfully",
		"data":    stats,
	})
}

func (h *TestWebSocketHandler) CleanupStaleConnections(c *gin.Context) {
	err := h.webSocketSvc.CleanupStaleConnections()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to cleanup stale connections"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Stale connections cleaned up successfully"})
}

func setupTestRouter() (*gin.Engine, *MockWebSocketSvc, *MockJWTSvc) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	mockWebSocketSvc := &MockWebSocketSvc{}
	mockJWTSvc := &MockJWTSvc{}

	handler := NewTestWebSocketHandler(mockWebSocketSvc, mockJWTSvc)

	router.GET("/ws", handler.HandleWebSocket)
	router.GET("/online-users", handler.GetOnlineUsers)
	router.GET("/online-users-db", handler.GetOnlineUsersFromDB)
	router.GET("/user/:user_id/history", handler.GetUserConnectionHistory)
	router.GET("/stats", handler.GetConnectionStats)
	router.POST("/cleanup", handler.CleanupStaleConnections)

	return router, mockWebSocketSvc, mockJWTSvc
}

func TestNewWebSocketHandler(t *testing.T) {
	mockWebSocketSvc := &MockWebSocketSvc{}
	mockJWTSvc := &MockJWTSvc{}

	handler := NewTestWebSocketHandler(mockWebSocketSvc, mockJWTSvc)

	assert.NotNil(t, handler)
	assert.Equal(t, mockWebSocketSvc, handler.webSocketSvc)
	assert.Equal(t, mockJWTSvc, handler.jwtSvc)
}

func TestHandleWebSocket_AuthHeader_Success(t *testing.T) {
	router, mockWebSocketSvc, mockJWTSvc := setupTestRouter()

	claims := &Claims{UserID: 123}
	mockJWTSvc.On("ValidToken", "valid-token").Return(claims, nil)
	mockWebSocketSvc.On("HandleConnection", mock.Anything, mock.Anything, 123).Return()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/ws", nil)
	req.Header.Set("Authorization", "Bearer valid-token")

	router.ServeHTTP(w, req)

	mockJWTSvc.AssertExpectations(t)
	mockWebSocketSvc.AssertExpectations(t)
}

func TestHandleWebSocket_WebSocketProtocol_Success(t *testing.T) {
	router, mockWebSocketSvc, mockJWTSvc := setupTestRouter()

	claims := &Claims{UserID: 456}
	mockJWTSvc.On("ValidToken", "protocol-token").Return(claims, nil)
	mockWebSocketSvc.On("HandleConnection", mock.Anything, mock.Anything, 456).Return()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/ws", nil)
	req.Header.Set("Sec-WebSocket-Protocol", "access_token.protocol-token, other-protocol")

	router.ServeHTTP(w, req)

	assert.Equal(t, "access_token", w.Header().Get("Sec-WebSocket-Protocol"))
	mockJWTSvc.AssertExpectations(t)
	mockWebSocketSvc.AssertExpectations(t)
}

func TestHandleWebSocket_NoToken(t *testing.T) {
	router, _, _ := setupTestRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/ws", nil)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, "access_token", w.Header().Get("Sec-WebSocket-Protocol"))

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "Authentication required", response["message"])
}

func TestHandleWebSocket_InvalidToken(t *testing.T) {
	router, _, mockJWTSvc := setupTestRouter()

	mockJWTSvc.On("ValidToken", "invalid-token").Return(nil, errors.New("invalid token"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/ws", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "Invalid token", response["message"])

	mockJWTSvc.AssertExpectations(t)
}

func TestGetOnlineUsers_Success(t *testing.T) {
	router, mockWebSocketSvc, _ := setupTestRouter()

	users := []interface{}{
		map[string]interface{}{"id": 1, "name": "User1"},
		map[string]interface{}{"id": 2, "name": "User2"},
	}
	mockWebSocketSvc.On("GetOnlineUsers").Return(users)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/online-users", nil)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(2), data["count"])
	assert.Len(t, data["online_users"], 2)

	mockWebSocketSvc.AssertExpectations(t)
}

func TestGetOnlineUsersFromDB_Error(t *testing.T) {
	router, mockWebSocketSvc, _ := setupTestRouter()

	mockWebSocketSvc.On("GetOnlineUsersFromDB").Return([]interface{}{}, errors.New("database error"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/online-users-db", nil)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "Failed to retrieve online users from database", response["message"])

	mockWebSocketSvc.AssertExpectations(t)
}

func TestGetUserConnectionHistory_InvalidUserID(t *testing.T) {
	router, _, _ := setupTestRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/user/invalid/history", nil)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "Invalid user ID", response["message"])
}

func TestGetConnectionStats_Success(t *testing.T) {
	router, mockWebSocketSvc, _ := setupTestRouter()

	stats := map[string]interface{}{
		"total_connections":  100,
		"active_connections": 50,
	}
	mockWebSocketSvc.On("GetConnectionStats").Return(stats, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/stats", nil)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(100), data["total_connections"])
	assert.Equal(t, float64(50), data["active_connections"])

	mockWebSocketSvc.AssertExpectations(t)
}

func TestCleanupStaleConnections_Success(t *testing.T) {
	router, mockWebSocketSvc, _ := setupTestRouter()

	mockWebSocketSvc.On("CleanupStaleConnections").Return(nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/cleanup", nil)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "Stale connections cleaned up successfully", response["message"])

	mockWebSocketSvc.AssertExpectations(t)
}
