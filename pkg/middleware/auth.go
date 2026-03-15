package middleware

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func RequireUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr := c.GetHeader("X-User-Id")
		if userIDStr == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Forbidden: missing internal authentication header"})
			c.Abort()
			return
		}
		userID, err := strconv.Atoi(userIDStr)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID format in header"})
			c.Abort()
			return
		}
		c.Set("user_id", userID)
		c.Next()
	}
}
