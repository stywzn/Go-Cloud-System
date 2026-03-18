package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stywzn/Go-Cloud-System/interaction/config"
)

// Lua script ensures INCR and EXPIRE are executed atomically in Redis
var rateLimitScript = redis.NewScript(`
    local current = redis.call("INCR", KEYS[1])
    if current == 1 then
        redis.call("EXPIRE", KEYS[1], ARGV[1])
    end
    return current
`)

func RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// In a microservice, you should ideally rate limit by user_id passed from gateway
		// userID, _ := c.Get("user_id")
		// key := fmt.Sprintf("rate_limit:user:%v", userID)

		// If you must use IP, ensure Gin trusts the proxy to get the real client IP
		ip := c.ClientIP()
		key := fmt.Sprintf("rate_limit:ip:%s", ip)

		// Execute Lua script: expire in 10 seconds
		result, err := rateLimitScript.Run(c.Request.Context(), config.Redis, []string{key}, 10).Result()
		if err != nil {
			// Log the error here in production
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"msg": "Security interceptor error"})
			return
		}

		count := result.(int64)

		if count > 50 {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"code": 429,
				"msg":  "Warning: Too many requests.",
			})
			return
		}

		c.Next()
	}
}
