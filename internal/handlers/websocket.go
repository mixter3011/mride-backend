package handlers

import (
	"net/http"
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
		utils.ErrJSON(c, http.StatusUnauthorized, "Authentication required")
		return
	}

	claims, err := h.jwtSvc.ValidToken(token)
	if err != nil {
		utils.ErrJSON(c, http.StatusUnauthorized, "Invalid token")
		return
	}

	userID := claims.UserID

	if c.GetHeader("Sec-WebSocket-Protocol") != "" {
		c.Header("Sec-WebSocket-Protocol", "access_token")
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
