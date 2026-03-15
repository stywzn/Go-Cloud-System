package trace

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TraceContextKey 用于在context中传递TraceID
type TraceContextKey struct{}

// GetTraceID 从Gin Context获取TraceID
func GetTraceID(c *gin.Context) string {
	if traceID, exists := c.Get("trace_id"); exists {
		return traceID.(string)
	}
	return ""
}

// GetTraceIDFromContext 从context.Context获取TraceID
func GetTraceIDFromContext(ctx context.Context) string {
	if traceID := ctx.Value(TraceContextKey{}); traceID != nil {
		return traceID.(string)
	}
	return ""
}

// SetTraceIDToContext 将TraceID设置到context.Context中
func SetTraceIDToContext(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceContextKey{}, traceID)
}

// GenerateTraceID 生成新的TraceID
func GenerateTraceID() string {
	return uuid.New().String()
}

// ExtractTraceMiddleware 从请求头提取TraceID的中间件
func ExtractTraceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.GetHeader("X-Trace-ID")
		if traceID == "" {
			traceID = GenerateTraceID()
		}
		
		// 存入Gin Context
		c.Set("trace_id", traceID)
		
		// 存入context.Context
		ctx := SetTraceIDToContext(c.Request.Context(), traceID)
		c.Request = c.Request.WithContext(ctx)
		
		// 写入响应头
		c.Header("X-Trace-ID", traceID)
		
		c.Next()
	}
}
