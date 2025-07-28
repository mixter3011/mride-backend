package services

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"gorm.io/gorm"
)

type UserConnection struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      int       `gorm:"uniqueIndex;not null" json:"user_id"`
	IP          string    `gorm:"size:45" json:"ip"`
	ConnectedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"connected_at"`
	LastPong    time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"last_pong"`
	IsActive    bool      `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ConnectionLog struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	UserID         int        `gorm:"not null" json:"user_id"`
	IP             string     `gorm:"size:45" json:"ip"`
	ConnectedAt    time.Time  `gorm:"default:CURRENT_TIMESTAMP" json:"connected_at"`
	DisconnectedAt *time.Time `json:"disconnected_at"`
	Duration       *int64     `json:"duration"`
	CreatedAt      time.Time  `json:"created_at"`
}

type WebSocketSvc struct {
	db             *gorm.DB
	clients        map[int]*Client
	clientsMux     sync.RWMutex
	upgrader       websocket.Upgrader
	rateLimiter    map[string]time.Time
	rateLimiterMux sync.RWMutex
}

type Client struct {
	conn         *websocket.Conn
	userID       int
	connectionID uint
	send         chan WSMessage
	ctx          context.Context
	cancel       context.CancelFunc
	lastPong     time.Time
	ip           string
	connectedAt  time.Time
}

type WSMessage struct {
	Type    string      `json:"type"`
	Title   string      `json:"title,omitempty"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
	rateLimit      = 5 * time.Second
)

func NewWebSocketSvc(db *gorm.DB) *WebSocketSvc {
	if err := db.AutoMigrate(&UserConnection{}, &ConnectionLog{}); err != nil {
		log.Printf("Failed to auto-migrate websocket models: %v", err)
	}

	return &WebSocketSvc{
		db:          db,
		clients:     make(map[int]*Client),
		rateLimiter: make(map[string]time.Time),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				allowedOrigins := []string{
					"",
					"",
				}

				for _, allowed := range allowedOrigins {
					if origin == allowed {
						return true
					}
				}

				return strings.Contains(origin, "localhost") || origin == ""
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}
}

func (ws *WebSocketSvc) HandleConnection(w http.ResponseWriter, r *http.Request, userID int) {
	clientIP := ws.getClientIP(r)
	if !ws.checkRateLimit(clientIP) {
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	conn, err := ws.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	userConn := UserConnection{
		UserID:      userID,
		IP:          clientIP,
		ConnectedAt: time.Now(),
		LastPong:    time.Now(),
		IsActive:    true,
	}

	ws.db.Model(&UserConnection{}).Where("user_id = ? AND is_active = ?", userID, true).
		Update("is_active", false)

	if err := ws.db.Create(&userConn).Error; err != nil {
		log.Printf("Failed to create user connection record: %v", err)
		conn.Close()
		return
	}

	connLog := ConnectionLog{
		UserID:      userID,
		IP:          clientIP,
		ConnectedAt: time.Now(),
	}
	if err := ws.db.Create(&connLog).Error; err != nil {
		log.Printf("Failed to create connection log: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	client := &Client{
		conn:         conn,
		userID:       userID,
		connectionID: userConn.ID,
		send:         make(chan WSMessage, 256),
		ctx:          ctx,
		cancel:       cancel,
		lastPong:     time.Now(),
		ip:           clientIP,
		connectedAt:  time.Now(),
	}

	ws.clientsMux.Lock()
	if existingClient, exists := ws.clients[userID]; exists {
		existingClient.cleanup(ws)
	}
	ws.clients[userID] = client
	ws.clientsMux.Unlock()

	log.Printf("User %d connected via WebSocket from IP %s", userID, clientIP)

	welcomeMsg := WSMessage{
		Type:    "connected",
		Message: "WebSocket connection established",
		Data: map[string]interface{}{
			"user_id":       userID,
			"connection_id": userConn.ID,
			"connected_at":  userConn.ConnectedAt,
		},
	}
	select {
	case client.send <- welcomeMsg:
	default:
		close(client.send)
		ws.removeClient(userID)
		return
	}

	go client.writePump(ws)
	go client.readPump(ws)
}

func (ws *WebSocketSvc) getClientIP(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

func (ws *WebSocketSvc) checkRateLimit(clientIP string) bool {
	ws.rateLimiterMux.Lock()
	defer ws.rateLimiterMux.Unlock()

	lastConnection, exists := ws.rateLimiter[clientIP]
	if exists && time.Since(lastConnection) < rateLimit {
		return false
	}

	ws.rateLimiter[clientIP] = time.Now()

	for ip, lastTime := range ws.rateLimiter {
		if time.Since(lastTime) > rateLimit*10 {
			delete(ws.rateLimiter, ip)
		}
	}

	return true
}

func (c *Client) cleanup(ws *WebSocketSvc) {
	c.cancel()
	if c.send != nil {
		close(c.send)
	}
	if c.conn != nil {
		c.conn.Close()
	}

	disconnectedAt := time.Now()
	duration := int64(disconnectedAt.Sub(c.connectedAt).Seconds())

	ws.db.Model(&UserConnection{}).Where("id = ?", c.connectionID).
		Updates(map[string]interface{}{
			"is_active": false,
			"last_pong": c.lastPong,
		})

	ws.db.Model(&ConnectionLog{}).Where("user_id = ? AND disconnected_at IS NULL", c.userID).
		Updates(map[string]interface{}{
			"disconnected_at": &disconnectedAt,
			"duration":        &duration,
		})
}

func (c *Client) readPump(ws *WebSocketSvc) {
	defer func() {
		ws.removeClient(c.userID)
	}()

	c.conn.SetReadLimit(maxMessageSize)
	if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		log.Printf("Failed to set read deadline: %v", err)
		return
	}
	c.conn.SetPongHandler(func(string) error {
		c.lastPong = time.Now()
		if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
			log.Printf("Failed to set read deadline: %v", err)
			return err
		}

		ws.db.Model(&UserConnection{}).Where("id = ?", c.connectionID).
			Update("last_pong", c.lastPong)

		return nil
	})

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			_, message, err := c.conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket error for user %d: %v", c.userID, err)
				}
				return
			}

			log.Printf("Message from user %d (IP: %s): %s", c.userID, c.ip, string(message))
		}
	}
}

func (c *Client) writePump(ws *WebSocketSvc) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case <-c.ctx.Done():
			return
		case message, ok := <-c.send:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				log.Printf("Failed to set write deadline: %v", err)
				return
			}
			if !ok {
				if err := c.conn.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
					log.Printf("Failed to write close message: %v", err)
				}
				return
			}

			messageJSON, err := json.Marshal(message)
			if err != nil {
				log.Printf("Failed to marshal message for user %d: %v", c.userID, err)
				continue
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, messageJSON); err != nil {
				log.Printf("Failed to write message to user %d: %v", c.userID, err)
				return
			}

		case <-ticker.C:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				log.Printf("Failed to set write deadline: %v", err)
				return
			}
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Failed to send ping to user %d: %v", c.userID, err)
				return
			}
		}
	}
}

func (ws *WebSocketSvc) removeClient(userID int) {
	ws.clientsMux.Lock()
	defer ws.clientsMux.Unlock()

	if client, exists := ws.clients[userID]; exists {
		client.cleanup(ws)
		delete(ws.clients, userID)
		log.Printf("User %d disconnected from WebSocket (IP: %s)", userID, client.ip)
	}
}

func (ws *WebSocketSvc) SendToUser(userID int, message WSMessage) error {
	ws.clientsMux.RLock()
	client, exists := ws.clients[userID]
	ws.clientsMux.RUnlock()

	if !exists {
		return nil
	}

	select {
	case client.send <- message:
		return nil
	default:
		ws.removeClient(userID)
		return nil
	}
}

func (ws *WebSocketSvc) IsUserOnline(userID int) bool {
	ws.clientsMux.RLock()
	defer ws.clientsMux.RUnlock()

	client, exists := ws.clients[userID]
	if !exists {
		return false
	}

	if time.Since(client.lastPong) > pongWait {
		go ws.removeClient(userID)
		return false
	}

	return true
}

func (ws *WebSocketSvc) GetOnlineUsers() []int {
	ws.clientsMux.RLock()
	defer ws.clientsMux.RUnlock()

	users := make([]int, 0, len(ws.clients))
	for userID, client := range ws.clients {
		if time.Since(client.lastPong) <= pongWait {
			users = append(users, userID)
		}
	}
	return users
}

func (ws *WebSocketSvc) GetOnlineUsersFromDB() ([]UserConnection, error) {
	var connections []UserConnection
	cutoff := time.Now().Add(-pongWait)
	err := ws.db.Where("is_active = ? AND last_pong > ?", true, cutoff).Find(&connections).Error
	return connections, err
}

func (ws *WebSocketSvc) GetUserConnectionHistory(userID int, limit int) ([]ConnectionLog, error) {
	var logs []ConnectionLog
	err := ws.db.Where("user_id = ?", userID).
		Order("connected_at DESC").
		Limit(limit).
		Find(&logs).Error
	return logs, err
}

func (ws *WebSocketSvc) GetConnectionStats() (map[string]interface{}, error) {
	var totalConnections int64
	var activeConnections int64
	var avgDuration float64

	ws.db.Model(&ConnectionLog{}).Count(&totalConnections)

	ws.db.Model(&UserConnection{}).Where("is_active = ?", true).Count(&activeConnections)

	ws.db.Model(&ConnectionLog{}).
		Where("duration IS NOT NULL").
		Select("AVG(duration)").
		Scan(&avgDuration)

	return map[string]interface{}{
		"total_connections":  totalConnections,
		"active_connections": activeConnections,
		"average_duration":   avgDuration,
	}, nil
}

func (ws *WebSocketSvc) CleanupStaleConnections() error {
	result := ws.db.Model(&UserConnection{}).
		Where("is_active = ? AND last_pong < ?", true, time.Now().Add(-pongWait*2)).
		Update("is_active", false)

	if result.Error != nil {
		return result.Error
	}

	log.Printf("Cleaned up %d stale connections", result.RowsAffected)
	return nil
}

func (ws *WebSocketSvc) Shutdown() {
	ws.clientsMux.Lock()
	defer ws.clientsMux.Unlock()

	for userID, client := range ws.clients {
		client.cleanup(ws)
		delete(ws.clients, userID)
	}

	ws.db.Model(&UserConnection{}).Where("is_active = ?", true).
		Update("is_active", false)

	log.Println("WebSocket service shutdown complete")
}

var _ WebSocketInterface = (*WebSocketSvc)(nil)
