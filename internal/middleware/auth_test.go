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

type Claims struct {
	UserID int    `json:"user_id"`
	Email  string `json:"email"`
}

type JWTService interface {
	ValidToken(token string) (*Claims, error)
	GenToken(userID int, email string) (string, error)
	GenRefreshToken(userID int, email string) (string, error)
	ValidateRefreshToken(token string) (*Claims, error)
}

type MockJWTSvc struct {
	mock.Mock
}

func (m *MockJWTSvc) ValidToken(token string) (*Claims, error) {
	args := m.Called(token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Claims), args.Error(1)
}

func (m *MockJWTSvc) GenToken(userID int, email string) (string, error) {
	args := m.Called(userID, email)
	return args.String(0), args.Error(1)
}

func (m *MockJWTSvc) GenRefreshToken(userID int, email string) (string, error) {
	args := m.Called(userID, email)
	return args.String(0), args.Error(1)
}

func (m *MockJWTSvc) ValidateRefreshToken(token string) (*Claims, error) {
	args := m.Called(token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Claims), args.Error(1)
}

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestAuthMiddleware_MissingAuthHeader(t *testing.T) {
	mockJWT := new(MockJWTSvc)
	router := setupTestRouter()

	router.Use(AuthMiddlewareWithInterface(mockJWT))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Authorization header required")
}

func TestAuthMiddleware_InvalidAuthFormat(t *testing.T) {
	mockJWT := new(MockJWTSvc)
	router := setupTestRouter()

	router.Use(AuthMiddlewareWithInterface(mockJWT))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	testCases := []struct {
		name   string
		header string
	}{
		{"missing bearer", "token123"},
		{"wrong prefix", "Basic token123"},
		{"extra parts", "Bearer token123 extra"},
		{"empty bearer", "Bearer "},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", tc.header)
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
			assert.Contains(t, w.Body.String(), "Invalid authorization format")
		})
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	mockJWT := new(MockJWTSvc)
	router := setupTestRouter()

	mockJWT.On("ValidToken", "invalid_token").Return(nil, assert.AnError)

	router.Use(AuthMiddlewareWithInterface(mockJWT))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid_token")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid token")
	mockJWT.AssertExpectations(t)
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	mockJWT := new(MockJWTSvc)
	router := setupTestRouter()

	expectedClaims := &Claims{
		UserID: 123,
		Email:  "test@example.com",
	}

	mockJWT.On("ValidToken", "valid_token").Return(expectedClaims, nil)

	router.Use(AuthMiddlewareWithInterface(mockJWT))
	router.GET("/test", func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		assert.True(t, exists)
		assert.Equal(t, 123, userID)

		email, exists := c.Get("email")
		assert.True(t, exists)
		assert.Equal(t, "test@example.com", email)

		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer valid_token")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "success")
	mockJWT.AssertExpectations(t)
}

func TestAuthMiddleware_Integration(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	jwtSvc := services.NewJWTSvc("test-secret", db)
	router := setupTestRouter()

	token, err := jwtSvc.GenToken(456, "integration@test.com")
	assert.NoError(t, err)

	router.Use(AuthMiddleware(jwtSvc))
	router.GET("/protected", func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		email, _ := c.Get("email")
		c.JSON(http.StatusOK, gin.H{
			"user_id": userID,
			"email":   email,
		})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "456")
	assert.Contains(t, w.Body.String(), "integration@test.com")
}

func TestAuthMiddleware_ContextValues(t *testing.T) {
	mockJWT := new(MockJWTSvc)
	router := setupTestRouter()

	claims := &Claims{
		UserID: 789,
		Email:  "context@test.com",
	}

	mockJWT.On("ValidToken", "context_token").Return(claims, nil)

	var capturedUserID interface{}
	var capturedEmail interface{}

	router.Use(AuthMiddlewareWithInterface(mockJWT))
	router.GET("/context", func(c *gin.Context) {
		capturedUserID, _ = c.Get("user_id")
		capturedEmail, _ = c.Get("email")
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/context", nil)
	req.Header.Set("Authorization", "Bearer context_token")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 789, capturedUserID)
	assert.Equal(t, "context@test.com", capturedEmail)
	mockJWT.AssertExpectations(t)
}

func AuthMiddlewareWithInterface(jwtSvc JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" || strings.TrimSpace(tokenParts[1]) == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization format"})
			c.Abort()
			return
		}

		claims, err := jwtSvc.ValidToken(tokenParts[1])
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
