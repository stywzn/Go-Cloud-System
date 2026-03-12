package middleware

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// JWTAuth 替换原有的 JWT 逻辑，现在只作为内部信任网关的 Header 提取器
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 直接读取网关透传的 Header
		userIDStr := c.GetHeader("X-User-Id")
		if userIDStr == "" {
			c.JSON(http.StatusForbidden, gin.H{"error": "禁止访问：缺少内部网关鉴权头"})
			c.Abort()
			return
		}

		userID, err := strconv.Atoi(userIDStr)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的用户 ID"})
			c.Abort()
			return
		}

		// 存入 Gin Context，你的 handler（比如 upload.go）直接用 c.Get("user_id") 获取即可
		c.Set("user_id", userID)
		c.Next()
	}
}

// GetUserID 从 Context 中获取透传的用户 ID
func GetUserID(c *gin.Context) uint {
	id, exists := c.Get("user_id")
	if !exists {
		return 0
	}
	return uint(id.(int)) // 强转为 uint
}
