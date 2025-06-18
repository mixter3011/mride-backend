package middleware

import (
	"net/http"
	"strings"

	"mride-backend/internal/services"
	"mride-backend/internal/utils"

	"github.com/gin-gonic/gin"
)

func WebSocketAuthMiddleware(jwtSvc *services.JWTSvc) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := ""

		authHeader := c.GetHeader("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		}

		if token == "" {
			utils.ErrJSON(c, http.StatusUnauthorized, "Missing auth token")
			c.Abort()
			return
		}

		claims, err := jwtSvc.ValidToken(token)
		if err != nil {
			utils.ErrJSON(c, http.StatusUnauthorized, "Invalid token")
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Next()
	}
}
