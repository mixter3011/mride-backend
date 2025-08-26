package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"mride-backend/internal/services"
	"mride-backend/internal/utils"

	"github.com/gin-gonic/gin"
)

type WebSocketHandler struct {
	webSocketSvc *services.WebSocketSvc
	jwtSvc       *services.JWTSvc
}

func NewWebSocketHandler(webSocketSvc *services.WebSocketSvc, jwtSvc *services.JWTSvc) *WebSocketHandler {
	return &WebSocketHandler{
		webSocketSvc: webSocketSvc,
		jwtSvc:       jwtSvc,
	}
}

func (h *WebSocketHandler) HandleWebSocket(c *gin.Context) {
	token := ""
	selectedProtocol := ""

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
					selectedProtocol = "access_token"
					break
				}
			}
		}
	}

	if token == "" {
		c.Header("Sec-WebSocket-Protocol", "access_token")
		utils.ErrJSON(c, http.StatusUnauthorized, "Authentication required")
		return
	}

	claims, err := h.jwtSvc.ValidToken(token)
	if err != nil {
		utils.ErrJSON(c, http.StatusUnauthorized, "Invalid token")
		return
	}

	userID := claims.UserID

	if selectedProtocol != "" {
		c.Header("Sec-WebSocket-Protocol", selectedProtocol)
	}

	h.webSocketSvc.HandleConnection(c.Writer, c.Request, userID)
}

func (h *WebSocketHandler) GetOnlineUsers(c *gin.Context) {
	users := h.webSocketSvc.GetOnlineUsers()
	utils.SuccJSON(c, "Online users retrieved successfully", map[string]interface{}{
		"online_users": users,
		"count":        len(users),
	})
}

func (h *WebSocketHandler) GetOnlineUsersFromDB(c *gin.Context) {
	connections, err := h.webSocketSvc.GetOnlineUsersFromDB()
	if err != nil {
		utils.ErrJSON(c, http.StatusInternalServerError, "Failed to retrieve online users from database")
		return
	}

	utils.SuccJSON(c, "Online users retrieved from database", map[string]interface{}{
		"connections": connections,
		"count":       len(connections),
	})
}

func (h *WebSocketHandler) GetUserConnectionHistory(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		utils.ErrJSON(c, http.StatusBadRequest, "Invalid user ID")
		return
	}

	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 10
	}

	history, err := h.webSocketSvc.GetUserConnectionHistory(userID, limit)
	if err != nil {
		utils.ErrJSON(c, http.StatusInternalServerError, "Failed to retrieve connection history")
		return
	}

	utils.SuccJSON(c, "Connection history retrieved successfully", map[string]interface{}{
		"history": history,
		"count":   len(history),
	})
}

func (h *WebSocketHandler) GetConnectionStats(c *gin.Context) {
	stats, err := h.webSocketSvc.GetConnectionStats()
	if err != nil {
		utils.ErrJSON(c, http.StatusInternalServerError, "Failed to retrieve connection statistics")
		return
	}

	utils.SuccJSON(c, "Connection statistics retrieved successfully", stats)
}

func (h *WebSocketHandler) CleanupStaleConnections(c *gin.Context) {
	err := h.webSocketSvc.CleanupStaleConnections()
	if err != nil {
		utils.ErrJSON(c, http.StatusInternalServerError, "Failed to cleanup stale connections")
		return
	}

	utils.SuccJSON(c, "Stale connections cleaned up successfully", nil)
}
