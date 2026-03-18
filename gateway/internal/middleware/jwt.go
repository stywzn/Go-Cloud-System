package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/stywzn/Go-Cloud-System/pkg/jwt"
)

// JWTAuthMiddleware Gateway authentication interceptor
func JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Strip potentially forged internal headers from the client
		c.Request.Header.Del("X-User-Id")

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing Authorization header"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token format, must be Bearer {token}"})
			c.Abort()
			return
		}

		// 2. Parse token
		claims, err := jwt.ParseToken(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// 3. Set standard Gin context for other gateway middlewares (e.g., Rate Limiter)
		c.Set("user_id", claims.UserID)

		// 4. Inject into HTTP Request Header for downstream services
		c.Request.Header.Set("X-User-Id", strconv.Itoa(claims.UserID))

		c.Next()
	}
}
