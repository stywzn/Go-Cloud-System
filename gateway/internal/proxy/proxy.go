package proxy

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
)

func GinReverseProxy(targetHost string) gin.HandlerFunc {
	targetURL, err := url.Parse(targetHost)
	if err != nil {
		log.Fatalf("Parse target host failed: %v", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// Override the Director to modify the request before forwarding
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		// Rewrite the Host header to match the backend service
		req.Host = targetURL.Host
	}

	return func(c *gin.Context) {
		// 1. Pass Trace ID to downstream
		if traceID, exists := c.Get("trace_id"); exists {
			c.Request.Header.Set("X-Trace-ID", traceID.(string))
		}

		// 2. Pass User ID to downstream (Requires JWT middleware to set this)
		if userID, exists := c.Get("user_id"); exists {
			c.Request.Header.Set("X-User-ID", fmt.Sprintf("%v", userID))
		}

		log.Printf("[Proxy] %s %s -> %s", c.Request.Method, c.Request.URL.Path, targetHost)

		// Execute the reverse proxy
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}
