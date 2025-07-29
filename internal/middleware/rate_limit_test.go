package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestNewTokenBucket(t *testing.T) {
	tb := NewTokenBucket(10, 5)

	assert.Equal(t, 10, tb.tokens)
	assert.Equal(t, 10, tb.maxTokens)
	assert.Equal(t, 5, tb.refillRate)
	assert.WithinDuration(t, time.Now(), tb.lastRefill, time.Second)
}

func TestTokenBucket_Allow(t *testing.T) {
	tb := NewTokenBucket(2, 1)

	assert.True(t, tb.Allow())
	assert.Equal(t, 1, tb.tokens)

	assert.True(t, tb.Allow())
	assert.Equal(t, 0, tb.tokens)

	assert.False(t, tb.Allow())
	assert.Equal(t, 0, tb.tokens)
}

func TestTokenBucket_Refill(t *testing.T) {
	tb := NewTokenBucket(5, 2)

	for i := 0; i < 5; i++ {
		assert.True(t, tb.Allow())
	}
	assert.False(t, tb.Allow())

	tb.lastRefill = time.Now().Add(-2 * time.Second)

	assert.True(t, tb.Allow())
	assert.Equal(t, 3, tb.tokens)
}

func TestTokenBucket_MaxTokensLimit(t *testing.T) {
	tb := NewTokenBucket(3, 10)

	tb.lastRefill = time.Now().Add(-10 * time.Second)

	assert.True(t, tb.Allow())
	assert.Equal(t, 2, tb.tokens)
}

func TestNewRateLimiter(t *testing.T) {
	rl := NewRateLimiter()

	assert.NotNil(t, rl)
	assert.NotNil(t, rl.visitors)
	assert.Equal(t, 0, len(rl.visitors))
}

func TestRateLimiter_Allow_NewVisitor(t *testing.T) {
	rl := NewRateLimiter()

	assert.True(t, rl.Allow("192.168.1.1"))
	assert.Equal(t, 1, len(rl.visitors))

	visitor, exists := rl.visitors["192.168.1.1"]
	assert.True(t, exists)
	assert.NotNil(t, visitor.limiter)
	assert.WithinDuration(t, time.Now(), visitor.lastSeen, time.Second)
}

func TestRateLimiter_Allow_ExistingVisitor(t *testing.T) {
	rl := NewRateLimiter()
	ip := "192.168.1.1"

	assert.True(t, rl.Allow(ip))

	initialTime := rl.visitors[ip].lastSeen

	time.Sleep(10 * time.Millisecond)
	assert.True(t, rl.Allow(ip))

	assert.True(t, rl.visitors[ip].lastSeen.After(initialTime))
	assert.Equal(t, 1, len(rl.visitors))
}

func TestRateLimiter_Allow_RateLimit(t *testing.T) {
	rl := NewRateLimiter()
	ip := "192.168.1.1"

	for i := 0; i < 100; i++ {
		assert.True(t, rl.Allow(ip))
	}

	assert.False(t, rl.Allow(ip))
}

func TestRateLimitMiddleware_Allowed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(RateLimitMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "success")
}

func TestRateLimitMiddleware_RateLimited(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	testRateLimiter := NewRateLimiter()

	router.Use(func(c *gin.Context) {
		ip := c.ClientIP()
		if !testRateLimiter.Allow(ip) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Rate limit exceeded"})
			c.Abort()
			return
		}
		c.Next()
	})

	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	for i := 0; i < 100; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Contains(t, w.Body.String(), "Rate limit exceeded")
}

func TestRateLimitMiddleware_DifferentIPs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	testRateLimiter := NewRateLimiter()

	router.Use(func(c *gin.Context) {
		ip := c.ClientIP()
		if !testRateLimiter.Allow(ip) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Rate limit exceeded"})
			c.Abort()
			return
		}
		c.Next()
	})

	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	for i := 0; i < 100; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/test", nil)
	req1.RemoteAddr = "192.168.1.1:12345"
	router.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusTooManyRequests, w1.Code)

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/test", nil)
	req2.RemoteAddr = "192.168.1.2:12345"
	router.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
}

func TestMin(t *testing.T) {
	assert.Equal(t, 5, min(5, 10))
	assert.Equal(t, 5, min(10, 5))
	assert.Equal(t, 5, min(5, 5))
	assert.Equal(t, -1, min(-1, 0))
}

func TestRateLimiter_Cleanup(t *testing.T) {
	rl := NewRateLimiter()

	rl.Allow("192.168.1.1")
	assert.Equal(t, 1, len(rl.visitors))

	rl.mu.Lock()
	rl.visitors["192.168.1.1"].lastSeen = time.Now().Add(-2 * time.Hour)
	rl.mu.Unlock()

	rl.mu.Lock()
	for ip, v := range rl.visitors {
		if time.Since(v.lastSeen) > time.Hour {
			delete(rl.visitors, ip)
		}
	}
	rl.mu.Unlock()

	assert.Equal(t, 0, len(rl.visitors))
}

func TestTokenBucket_ConcurrentAccess(t *testing.T) {
	tb := NewTokenBucket(100, 10)

	results := make(chan bool, 200)

	for i := 0; i < 200; i++ {
		go func() {
			results <- tb.Allow()
		}()
	}

	allowed := 0
	denied := 0
	for i := 0; i < 200; i++ {
		if <-results {
			allowed++
		} else {
			denied++
		}
	}

	assert.Equal(t, 100, allowed)
	assert.Equal(t, 100, denied)
	assert.Equal(t, 0, tb.tokens)
}
