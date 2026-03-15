package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TraceMiddleware 全链路追踪中间件
func TraceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 尝试从外部请求获取，如果没有则生成一个新的 UUID
		traceID := c.GetHeader("X-Trace-ID")
		if traceID == "" {
			traceID = uuid.New().String()
		}

		// 1. 将 TraceID 存入当前 Gin 的上下文中，方便后续业务代码打印日志
		c.Set("trace_id", traceID)

		// 2. 将 TraceID 写入响应头（方便前端联调排错）
		c.Header("X-Trace-ID", traceID)

		c.Next()
	}
}
