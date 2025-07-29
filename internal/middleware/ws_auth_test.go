package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"mride-backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type WSClaims struct {
	UserID int    `json:"user_id"`
	Email  string `json:"email"`
}

type WSJWTService interface {
	ValidToken(token string) (*WSClaims, error)
	GenToken(userID int, email string) (string, error)
	GenRefreshToken(userID int, email string) (string, error)
	ValidateRefreshToken(token string) (*WSClaims, error)
}

type MockWSJWTSvc struct {
	mock.Mock
}

func (m *MockWSJWTSvc) ValidToken(token string) (*WSClaims, error) {
	args := m.Called(token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*WSClaims), args.Error(1)
}

func (m *MockWSJWTSvc) GenToken(userID int, email string) (string, error) {
	args := m.Called(userID, email)
	return args.String(0), args.Error(1)
}

func (m *MockWSJWTSvc) GenRefreshToken(userID int, email string) (string, error) {
	args := m.Called(userID, email)
	return args.String(0), args.Error(1)
}

func (m *MockWSJWTSvc) ValidateRefreshToken(token string) (*WSClaims, error) {
	args := m.Called(token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*WSClaims), args.Error(1)
}

func setupWSTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestWebSocketAuthMiddleware_MissingAuthHeader(t *testing.T) {
	mockJWT := new(MockWSJWTSvc)
	router := setupWSTestRouter()

	router.Use(WebSocketAuthMiddlewareWithInterface(mockJWT))
	router.GET("/ws", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/ws", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Missing auth token")
}

func TestWebSocketAuthMiddleware_EmptyAuthHeader(t *testing.T) {
	mockJWT := new(MockWSJWTSvc)
	router := setupWSTestRouter()

	router.Use(WebSocketAuthMiddlewareWithInterface(mockJWT))
	router.GET("/ws", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/ws", nil)
	req.Header.Set("Authorization", "")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Missing auth token")
}

func TestWebSocketAuthMiddleware_InvalidAuthFormat(t *testing.T) {
	mockJWT := new(MockWSJWTSvc)
	router := setupWSTestRouter()

	router.Use(WebSocketAuthMiddlewareWithInterface(mockJWT))
	router.GET("/ws", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	testCases := []struct {
		name   string
		header string
	}{
		{"missing bearer", "token123"},
		{"wrong prefix", "Basic token123"},
		{"no token after bearer", "Bearer"},
		{"bearer with space only", "Bearer "},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/ws", nil)
			req.Header.Set("Authorization", tc.header)
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
			assert.Contains(t, w.Body.String(), "Missing auth token")
		})
	}
}

func TestWebSocketAuthMiddleware_ValidBearerToken(t *testing.T) {
	mockJWT := new(MockWSJWTSvc)
	router := setupWSTestRouter()

	expectedClaims := &WSClaims{
		UserID: 123,
		Email:  "test@example.com",
	}

	mockJWT.On("ValidToken", "valid_token").Return(expectedClaims, nil)

	router.Use(WebSocketAuthMiddlewareWithInterface(mockJWT))
	router.GET("/ws", func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		assert.True(t, exists)
		assert.Equal(t, 123, userID)

		email, exists := c.Get("email")
		assert.True(t, exists)
		assert.Equal(t, "test@example.com", email)

		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/ws", nil)
	req.Header.Set("Authorization", "Bearer valid_token")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "success")
	mockJWT.AssertExpectations(t)
}

func TestWebSocketAuthMiddleware_InvalidToken(t *testing.T) {
	mockJWT := new(MockWSJWTSvc)
	router := setupWSTestRouter()

	mockJWT.On("ValidToken", "invalid_token").Return(nil, assert.AnError)

	router.Use(WebSocketAuthMiddlewareWithInterface(mockJWT))
	router.GET("/ws", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/ws", nil)
	req.Header.Set("Authorization", "Bearer invalid_token")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid token")
	mockJWT.AssertExpectations(t)
}

func TestWebSocketAuthMiddleware_BearerWithSpaces(t *testing.T) {
	mockJWT := new(MockWSJWTSvc)
	router := setupWSTestRouter()

	expectedClaims := &WSClaims{
		UserID: 456,
		Email:  "spaces@example.com",
	}

	mockJWT.On("ValidToken", "token_with_spaces").Return(expectedClaims, nil)

	router.Use(WebSocketAuthMiddlewareWithInterface(mockJWT))
	router.GET("/ws", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/ws", nil)
	req.Header.Set("Authorization", "Bearer   token_with_spaces   ")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockJWT.AssertExpectations(t)
}

func TestWebSocketAuthMiddleware_Integration(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	jwtSvc := services.NewJWTSvc("test-secret", db)
	router := setupWSTestRouter()

	token, err := jwtSvc.GenToken(789, "integration@test.com")
	assert.NoError(t, err)

	router.Use(WebSocketAuthMiddleware(jwtSvc))
	router.GET("/ws", func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		email, _ := c.Get("email")
		c.JSON(http.StatusOK, gin.H{
			"user_id": userID,
			"email":   email,
		})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/ws", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "789")
	assert.Contains(t, w.Body.String(), "integration@test.com")
}

func TestWebSocketAuthMiddleware_ContextValues(t *testing.T) {
	mockJWT := new(MockWSJWTSvc)
	router := setupWSTestRouter()

	claims := &WSClaims{
		UserID: 999,
		Email:  "context@test.com",
	}

	mockJWT.On("ValidToken", "context_token").Return(claims, nil)

	var capturedUserID interface{}
	var capturedEmail interface{}

	router.Use(WebSocketAuthMiddlewareWithInterface(mockJWT))
	router.GET("/ws", func(c *gin.Context) {
		capturedUserID, _ = c.Get("user_id")
		capturedEmail, _ = c.Get("email")
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/ws", nil)
	req.Header.Set("Authorization", "Bearer context_token")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 999, capturedUserID)
	assert.Equal(t, "context@test.com", capturedEmail)
	mockJWT.AssertExpectations(t)
}

func TestWebSocketAuthMiddleware_CallsNext(t *testing.T) {
	mockJWT := new(MockWSJWTSvc)
	router := setupWSTestRouter()

	var middlewareCalled bool
	var handlerCalled bool

	claims := &WSClaims{
		UserID: 111,
		Email:  "next@test.com",
	}

	mockJWT.On("ValidToken", "next_token").Return(claims, nil)

	router.Use(WebSocketAuthMiddlewareWithInterface(mockJWT))
	router.Use(func(c *gin.Context) {
		middlewareCalled = true
		c.Next()
	})
	router.GET("/ws", func(c *gin.Context) {
		handlerCalled = true
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/ws", nil)
	req.Header.Set("Authorization", "Bearer next_token")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, middlewareCalled)
	assert.True(t, handlerCalled)
	mockJWT.AssertExpectations(t)
}

func TestWebSocketAuthMiddleware_LongToken(t *testing.T) {
	mockJWT := new(MockWSJWTSvc)
	router := setupWSTestRouter()

	longToken := strings.Repeat("a", 1000)
	claims := &WSClaims{
		UserID: 222,
		Email:  "long@test.com",
	}

	mockJWT.On("ValidToken", longToken).Return(claims, nil)

	router.Use(WebSocketAuthMiddlewareWithInterface(mockJWT))
	router.GET("/ws", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/ws", nil)
	req.Header.Set("Authorization", "Bearer "+longToken)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockJWT.AssertExpectations(t)
}

func WebSocketAuthMiddlewareWithInterface(jwtSvc WSJWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := ""

		authHeader := c.GetHeader("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		}

		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing auth token"})
			c.Abort()
			return
		}

		claims, err := jwtSvc.ValidToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Next()
	}
}
