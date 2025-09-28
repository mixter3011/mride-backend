package services

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupWebSocketTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	err = db.AutoMigrate(&UserConnection{}, &ConnectionLog{})
	assert.NoError(t, err)

	return db
}

func TestNewWebSocketSvc(t *testing.T) {
	db := setupWebSocketTestDB(t)
	ws := NewWebSocketSvc(db)

	assert.NotNil(t, ws)
	assert.NotNil(t, ws.db)
	assert.NotNil(t, ws.clients)
	assert.NotNil(t, ws.rateLimiter)
}

func TestWebSocketSvc_IsUserOnline(t *testing.T) {
	db := setupWebSocketTestDB(t)
	ws := NewWebSocketSvc(db)

	assert.False(t, ws.IsUserOnline(1))

	ctx, cancel := context.WithCancel(context.Background())
	client := &Client{
		userID:   1,
		lastPong: time.Now(),
		ctx:      ctx,
		cancel:   cancel,
	}
	ws.clients[1] = client

	assert.True(t, ws.IsUserOnline(1))

	client.lastPong = time.Now().Add(-pongWait - time.Minute)
	assert.False(t, ws.IsUserOnline(1))

	cancel()
}

func TestWebSocketSvc_SendToUser(t *testing.T) {
	db := setupWebSocketTestDB(t)
	ws := NewWebSocketSvc(db)

	message := WSMessage{Type: "test", Message: "hello"}
	err := ws.SendToUser(1, message)
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())

	client := &Client{
		userID:   1,
		send:     make(chan WSMessage, 1),
		lastPong: time.Now(),
		ctx:      ctx,
		cancel:   cancel,
	}
	ws.clients[1] = client

	err = ws.SendToUser(1, message)
	assert.NoError(t, err)

	select {
	case receivedMsg := <-client.send:
		assert.Equal(t, "test", receivedMsg.Type)
		assert.Equal(t, "hello", receivedMsg.Message)
	case <-time.After(time.Second):
		t.Fatal("Message not received")
	}

	cancel()
	close(client.send)
}

func TestWebSocketSvc_GetOnlineUsers(t *testing.T) {
	db := setupWebSocketTestDB(t)
	ws := NewWebSocketSvc(db)

	users := ws.GetOnlineUsers()
	assert.Empty(t, users)

	for i := 1; i <= 3; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		client := &Client{
			userID:   i,
			lastPong: time.Now(),
			ctx:      ctx,
			cancel:   cancel,
		}
		ws.clients[i] = client
		defer cancel()
	}

	ctx, cancel := context.WithCancel(context.Background())
	staleClient := &Client{
		userID:   4,
		lastPong: time.Now().Add(-pongWait - time.Minute),
		ctx:      ctx,
		cancel:   cancel,
	}
	ws.clients[4] = staleClient
	defer cancel()

	users = ws.GetOnlineUsers()
	assert.Len(t, users, 3)
	assert.Contains(t, users, 1)
	assert.Contains(t, users, 2)
	assert.Contains(t, users, 3)
	assert.NotContains(t, users, 4)
}

func TestWebSocketSvc_CleanupStaleConnections(t *testing.T) {
	db := setupWebSocketTestDB(t)
	ws := NewWebSocketSvc(db)

	connections := []UserConnection{
		{
			UserID:      1,
			IP:          "127.0.0.1",
			ConnectedAt: time.Now(),
			LastPong:    time.Now(),
			IsActive:    true,
		},
		{
			UserID:      2,
			IP:          "127.0.0.1",
			ConnectedAt: time.Now(),
			LastPong:    time.Now().Add(-pongWait * 3),
			IsActive:    true,
		},
	}

	for _, conn := range connections {
		db.Create(&conn)
	}

	err := ws.CleanupStaleConnections()
	assert.NoError(t, err)

	var activeCount int64
	db.Model(&UserConnection{}).Where("is_active = ?", true).Count(&activeCount)
	assert.Equal(t, int64(1), activeCount)
}

func TestWebSocketSvc_GetConnectionStats(t *testing.T) {
	db := setupWebSocketTestDB(t)
	ws := NewWebSocketSvc(db)

	db.Create(&ConnectionLog{
		UserID:      1,
		IP:          "127.0.0.1",
		ConnectedAt: time.Now(),
		Duration:    intPtr(300),
	})
	db.Create(&ConnectionLog{
		UserID:      2,
		IP:          "127.0.0.1",
		ConnectedAt: time.Now(),
		Duration:    intPtr(500),
	})
	db.Create(&UserConnection{
		UserID:      1,
		IP:          "127.0.0.1",
		ConnectedAt: time.Now(),
		IsActive:    true,
	})

	stats, err := ws.GetConnectionStats()
	assert.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, int64(2), stats["total_connections"])
	assert.Equal(t, int64(1), stats["active_connections"])
	assert.Equal(t, float64(400), stats["average_duration"])
}

func TestWebSocketSvc_GetUserConnectionHistory(t *testing.T) {
	db := setupWebSocketTestDB(t)
	ws := NewWebSocketSvc(db)

	userID := 1

	logs := []ConnectionLog{
		{
			UserID:      userID,
			IP:          "127.0.0.1",
			ConnectedAt: time.Now(),
			Duration:    intPtr(300),
		},
		{
			UserID:      userID,
			IP:          "127.0.0.2",
			ConnectedAt: time.Now().Add(-time.Hour),
			Duration:    intPtr(600),
		},
		{
			UserID:      2,
			IP:          "127.0.0.3",
			ConnectedAt: time.Now(),
			Duration:    intPtr(400),
		},
	}

	for _, log := range logs {
		db.Create(&log)
	}

	history, err := ws.GetUserConnectionHistory(userID, 10)
	assert.NoError(t, err)
	assert.Len(t, history, 2)
}

func TestWebSocketSvc_GetOnlineUsersFromDB(t *testing.T) {
	db := setupWebSocketTestDB(t)
	ws := NewWebSocketSvc(db)

	conn1 := UserConnection{
		UserID:      1,
		IP:          "127.0.0.1",
		ConnectedAt: time.Now(),
		LastPong:    time.Now(),
		IsActive:    true,
	}
	db.Create(&conn1)

	conn2 := UserConnection{
		UserID:      2,
		IP:          "127.0.0.2",
		ConnectedAt: time.Now(),
		LastPong:    time.Now().Add(-pongWait * 3),
		IsActive:    true,
	}
	db.Create(&conn2)

	conn3 := UserConnection{
		UserID:      3,
		IP:          "127.0.0.3",
		ConnectedAt: time.Now(),
		LastPong:    time.Now().Add(-pongWait * 3),
		IsActive:    false,
	}
	db.Create(&conn3)

	onlineUsers, err := ws.GetOnlineUsersFromDB()
	assert.NoError(t, err)
	assert.Len(t, onlineUsers, 1)
	assert.Equal(t, 1, onlineUsers[0].UserID)
}

func TestWebSocketSvc_Shutdown(t *testing.T) {
	db := setupWebSocketTestDB(t)
	ws := NewWebSocketSvc(db)

	clients := make([]*Client, 3)

	for i := 1; i <= 3; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		client := &Client{
			userID:       i,
			connectionID: uint(i),
			send:         make(chan WSMessage, 1),
			ctx:          ctx,
			cancel:       cancel,
			connectedAt:  time.Now(),
		}
		ws.clients[i] = client
		clients[i-1] = client
	}

	for i := 1; i <= 3; i++ {
		db.Create(&UserConnection{
			UserID:   i,
			IsActive: true,
		})
	}

	ws.Shutdown()

	assert.Empty(t, ws.clients)

	var activeCount int64
	db.Model(&UserConnection{}).Where("is_active = ?", true).Count(&activeCount)
	assert.Equal(t, int64(0), activeCount)

	for _, client := range clients {
		select {
		case <-client.ctx.Done():
		case <-time.After(100 * time.Millisecond):
			t.Error("Client context was not properly cancelled")
		}
	}
}

func TestWebSocketSvc_CheckRateLimit(t *testing.T) {
	db := setupWebSocketTestDB(t)
	ws := NewWebSocketSvc(db)

	clientIP := "127.0.0.1"

	assert.True(t, ws.checkRateLimit(clientIP))

	assert.False(t, ws.checkRateLimit(clientIP))

	time.Sleep(rateLimit + 100*time.Millisecond)

	assert.True(t, ws.checkRateLimit(clientIP))
}

func TestWebSocketSvc_GetClientIP(t *testing.T) {
	db := setupWebSocketTestDB(t)
	ws := NewWebSocketSvc(db)

	req := httptest.NewRequest("GET", "/ws", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.1, 10.0.0.1")
	ip := ws.getClientIP(req)
	assert.Equal(t, "192.168.1.1", ip)

	req = httptest.NewRequest("GET", "/ws", nil)
	req.Header.Set("X-Real-IP", "192.168.1.2")
	ip = ws.getClientIP(req)
	assert.Equal(t, "192.168.1.2", ip)

	req = httptest.NewRequest("GET", "/ws", nil)
	req.RemoteAddr = "192.168.1.3:12345"
	ip = ws.getClientIP(req)
	assert.Equal(t, "192.168.1.3", ip)
}

func TestClient_Cleanup(t *testing.T) {
	db := setupWebSocketTestDB(t)
	ws := NewWebSocketSvc(db)

	userConn := UserConnection{
		UserID:      1,
		IP:          "127.0.0.1",
		ConnectedAt: time.Now(),
		IsActive:    true,
	}
	db.Create(&userConn)

	connLog := ConnectionLog{
		UserID:      1,
		IP:          "127.0.0.1",
		ConnectedAt: time.Now(),
	}
	db.Create(&connLog)

	ctx, cancel := context.WithCancel(context.Background())
	client := &Client{
		userID:       1,
		connectionID: userConn.ID,
		ctx:          ctx,
		cancel:       cancel,
		connectedAt:  time.Now(),
		send:         make(chan WSMessage, 1),
		conn:         nil,
	}

	client.cleanup(ws)

	var updatedConn UserConnection
	db.First(&updatedConn, userConn.ID)
	assert.False(t, updatedConn.IsActive)

	var updatedLog ConnectionLog
	db.First(&updatedLog, connLog.ID)
	assert.NotNil(t, updatedLog.DisconnectedAt)
	assert.NotNil(t, updatedLog.Duration)
}

func intPtr(i int64) *int64 {
	return &i
}
