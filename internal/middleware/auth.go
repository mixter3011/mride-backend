package middleware

import (
	"net/http"
	"strings"

	"mride-backend/internal/services"
	"mride-backend/internal/utils"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware(jwtSvc *services.JWTSvc) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			utils.ErrJSON(c, http.StatusUnauthorized, "Authorization header required")
			c.Abort()
			return
		}

		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			utils.ErrJSON(c, http.StatusUnauthorized, "Invalid authorization format")
			c.Abort()
			return
		}

		claims, err := jwtSvc.ValidToken(tokenParts[1])
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
