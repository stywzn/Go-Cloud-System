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
		log.Printf("[Gin 路由转发] %s %s -> %s", c.Request.Method, c.Request.URL.Path, targetHost)
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}
