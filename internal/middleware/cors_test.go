package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestCORSMiddleware_Headers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(CORSMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://saby.to")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	headers := w.Header()
	assert.Equal(t, "https://saby.to", headers.Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", headers.Get("Access-Control-Allow-Credentials"))
	assert.Equal(t, "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, Sec-WebSocket-Protocol, Sec-WebSocket-Key, Sec-WebSocket-Version, Connection, Upgrade", headers.Get("Access-Control-Allow-Headers"))
	assert.Equal(t, "POST, OPTIONS, GET, PUT, DELETE", headers.Get("Access-Control-Allow-Methods"))
}

func TestCORSMiddleware_OptionsRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(CORSMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "should not reach here"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "https://www.saby.to")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, w.Body.String())

	headers := w.Header()
	assert.Equal(t, "https://www.saby.to", headers.Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "POST, OPTIONS, GET, PUT, DELETE", headers.Get("Access-Control-Allow-Methods"))
}

func TestCORSMiddleware_PreflightRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(CORSMiddleware())
	router.POST("/api/data", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "should not reach here"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "/api/data", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type, Authorization")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, w.Body.String())

	headers := w.Header()
	assert.Equal(t, "http://localhost:3000", headers.Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", headers.Get("Access-Control-Allow-Credentials"))
	assert.Equal(t, "POST, OPTIONS, GET, PUT, DELETE", headers.Get("Access-Control-Allow-Methods"))
}

func TestCORSMiddleware_AllMethods(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(CORSMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"method": "GET"})
	})
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"method": "POST"})
	})
	router.PUT("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"method": "PUT"})
	})
	router.DELETE("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"method": "DELETE"})
	})

	methods := []string{"GET", "POST", "PUT", "DELETE"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(method, "/test", nil)
			req.Header.Set("Origin", "https://saby.to")
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Contains(t, w.Body.String(), method)

			headers := w.Header()
			assert.Equal(t, "https://saby.to", headers.Get("Access-Control-Allow-Origin"))
			assert.Equal(t, "true", headers.Get("Access-Control-Allow-Credentials"))
		})
	}
}

func TestCORSMiddleware_InvalidOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(CORSMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://malicious-site.com")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORSMiddleware_CallsNext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	var middlewareCalled bool
	var handlerCalled bool

	router.Use(CORSMiddleware())
	router.Use(func(c *gin.Context) {
		middlewareCalled = true
		c.Next()
	})
	router.GET("/test", func(c *gin.Context) {
		handlerCalled = true
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, middlewareCalled)
	assert.True(t, handlerCalled)
}

func TestCORSMiddleware_WithCustomHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(CORSMiddleware())
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/test", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer token123")
	req.Header.Set("X-Custom-Header", "custom-value")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	headers := w.Header()
	allowedHeaders := headers.Get("Access-Control-Allow-Headers")
	assert.Contains(t, allowedHeaders, "Content-Type")
	assert.Contains(t, allowedHeaders, "Authorization")
}

func TestCORSMiddleware_OptionsDoesNotCallNext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	var handlerCalled bool

	router.Use(CORSMiddleware())
	router.OPTIONS("/test", func(c *gin.Context) {
		handlerCalled = true
		c.JSON(http.StatusOK, gin.H{"message": "should not reach"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.False(t, handlerCalled)
	assert.Empty(t, w.Body.String())
}
