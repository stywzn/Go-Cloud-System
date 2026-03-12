package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/stywzn/Go-Cloud-System/pkg/jwt"
)

// JWTAuthMiddleware 网关鉴权拦截器
func JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未提供 Authorization 请求头"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token 格式错误，需为 Bearer {token}"})
			c.Abort()
			return
		}

		// 直接使用我们封装好的 ParseToken
		claims, err := jwt.ParseToken(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token 无效或已过期"})
			c.Abort()
			return
		}

		// 将 UserID 放入 Request Header 透传给下游
		c.Request.Header.Set("X-User-Id", strconv.Itoa(claims.UserID))

		c.Next()
	}
}
