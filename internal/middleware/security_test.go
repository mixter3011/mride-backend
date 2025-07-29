package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestSecurityHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(SecurityHeaders())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	headers := w.Header()

	assert.Equal(t, "nosniff", headers.Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", headers.Get("X-Frame-Options"))
	assert.Equal(t, "1; mode=block", headers.Get("X-XSS-Protection"))
	assert.Equal(t, "max-age=31536000; includeSubDomains", headers.Get("Strict-Transport-Security"))
	assert.Equal(t, "default-src 'self'", headers.Get("Content-Security-Policy"))
	assert.Equal(t, "strict-origin-when-cross-origin", headers.Get("Referrer-Policy"))
	assert.Equal(t, "geolocation=(), microphone=(), camera=()", headers.Get("Permissions-Policy"))
}

func TestSecurityHeaders_MultipleMethods(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(SecurityHeaders())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"method": "GET"})
	})
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"method": "POST"})
	})
	router.PUT("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"method": "PUT"})
	})

	methods := []string{"GET", "POST", "PUT"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(method, "/test", nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			headers := w.Header()
			assert.Equal(t, "nosniff", headers.Get("X-Content-Type-Options"))
			assert.Equal(t, "DENY", headers.Get("X-Frame-Options"))
			assert.Equal(t, "1; mode=block", headers.Get("X-XSS-Protection"))
		})
	}
}

func TestSecurityHeaders_HeadersPersist(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(SecurityHeaders())
	router.GET("/modify", func(c *gin.Context) {

		c.Writer.Header().Set("X-Frame-Options", "SAMEORIGIN")
		c.JSON(http.StatusOK, gin.H{"message": "modified"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/modify", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	assert.Equal(t, "SAMEORIGIN", w.Header().Get("X-Frame-Options"))
}

func TestSecurityHeaders_DoesNotAffectResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(SecurityHeaders())
	router.GET("/json", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"data": "test", "number": 123})
	})
	router.GET("/text", func(c *gin.Context) {
		c.String(http.StatusOK, "plain text response")
	})

	t.Run("JSON response", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/json", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "test")
		assert.Contains(t, w.Body.String(), "123")
		assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	})

	t.Run("Text response", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/text", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "plain text response")
		assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	})
}

func TestSecurityHeaders_WithMultipleMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(SecurityHeaders())

	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Custom-Header", "custom-value")
		c.Next()
	})

	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	headers := w.Header()

	assert.Equal(t, "nosniff", headers.Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", headers.Get("X-Frame-Options"))

	assert.Equal(t, "custom-value", headers.Get("Custom-Header"))
}

func TestSecurityHeaders_AllHeadersPresent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(SecurityHeaders())
	router.GET("/comprehensive", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"test": "comprehensive"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/comprehensive", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	expectedHeaders := map[string]string{
		"X-Content-Type-Options":    "nosniff",
		"X-Frame-Options":           "DENY",
		"X-XSS-Protection":          "1; mode=block",
		"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
		"Content-Security-Policy":   "default-src 'self'",
		"Referrer-Policy":           "strict-origin-when-cross-origin",
		"Permissions-Policy":        "geolocation=(), microphone=(), camera=()",
	}

	for headerName, expectedValue := range expectedHeaders {
		actualValue := w.Header().Get(headerName)
		assert.Equal(t, expectedValue, actualValue, "Header %s should have value %s but got %s", headerName, expectedValue, actualValue)
	}
}

func TestSecurityHeaders_CallsNext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	var middlewareCalled bool
	var handlerCalled bool

	router.Use(SecurityHeaders())
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
	assert.True(t, middlewareCalled, "Next middleware should be called")
	assert.True(t, handlerCalled, "Handler should be called")
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
}
