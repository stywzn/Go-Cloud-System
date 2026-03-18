package api

import (
	"net/http"

	"github.com/stywzn/Go-Cloud-System/interaction/internal/service" // 替换为你的真实 module 路径

	"github.com/gin-gonic/gin"
)

// SeckillHandler 处理抽奖请求
func SeckillHandler(c *gin.Context) {
	// 这里的 user_id 是网关解析 JWT 后，通过 Header 透传过来，并由我们的 pkg/middleware 提取放入 Context 的
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未获取到用户身份"})
		return
	}
	// Safe type assertion
	userID, ok := userIDVal.(int)
	if !ok {
		// If it might be a float64 (common with JSON numbers) or string, handle it here
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error: user_id type mismatch"})
		return
	}

	// 调用底层秒杀业务逻辑
	err := service.DoSeckill(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 极速返回 202 Accepted，业务已经完全交给 Storage 去异步处理了
	c.JSON(http.StatusAccepted, gin.H{
		"message": "恭喜您抢到 4G 扩容配额！系统正在为您发货，预计 1 分钟内到账。",
	})
}
