package proxy

import (
	"log"
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
)

func GinReverseProxy(targetHost string) gin.HandlerFunc {
	targetURL, err := url.Parse(targetHost)
	if err != nil {
		log.Fatalf("解析目标地址失败: %v", err)
	}
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	return func(c *gin.Context) {
		// 获取TraceID并传递给下游服务
		traceID, _ := c.Get("trace_id")
		if traceID != nil {
			c.Request.Header.Set("X-Trace-ID", traceID.(string))
		}

		log.Printf("[Gin 路由转发] %s %s -> %s (TraceID: %v)",
			c.Request.Method, c.Request.URL.Path, targetHost, traceID)
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}
