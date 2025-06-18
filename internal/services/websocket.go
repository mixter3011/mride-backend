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
)

type WebSocketSvc struct {
	clients        map[int]*Client
	clientsMux     sync.RWMutex
	upgrader       websocket.Upgrader
	rateLimiter    map[string]time.Time
	rateLimiterMux sync.RWMutex
}

type Client struct {
	conn     *websocket.Conn
	userID   int
	send     chan WSMessage
	ctx      context.Context
	cancel   context.CancelFunc
	lastPong time.Time
	ip       string
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

func NewWebSocketSvc() *WebSocketSvc {
	return &WebSocketSvc{
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

	ctx, cancel := context.WithCancel(context.Background())
	client := &Client{
		conn:     conn,
		userID:   userID,
		send:     make(chan WSMessage, 256),
		ctx:      ctx,
		cancel:   cancel,
		lastPong: time.Now(),
		ip:       clientIP,
	}

	ws.clientsMux.Lock()
	if existingClient, exists := ws.clients[userID]; exists {
		existingClient.cleanup()
	}
	ws.clients[userID] = client
	ws.clientsMux.Unlock()

	log.Printf("User %d connected via WebSocket from IP %s", userID, clientIP)

	welcomeMsg := WSMessage{
		Type:    "connected",
		Message: "WebSocket connection established",
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

func (c *Client) cleanup() {
	c.cancel()
	close(c.send)
	c.conn.Close()
}

func (c *Client) readPump(ws *WebSocketSvc) {
	defer func() {
		ws.removeClient(c.userID)
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.lastPong = time.Now()
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
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
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
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
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
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
		client.cleanup()
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

func (ws *WebSocketSvc) Shutdown() {
	ws.clientsMux.Lock()
	defer ws.clientsMux.Unlock()

	for userID, client := range ws.clients {
		client.cleanup()
		delete(ws.clients, userID)
	}
	log.Println("WebSocket service shutdown complete")
}
