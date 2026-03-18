package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// clientLimiter holds the limiter and the last seen timestamp for an IP
type clientLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type IPRateLimiter struct {
	ips map[string]*clientLimiter
	mu  *sync.RWMutex
	r   rate.Limit
	b   int
}

func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	i := &IPRateLimiter{
		ips: make(map[string]*clientLimiter),
		mu:  &sync.RWMutex{},
		r:   r,
		b:   b,
	}

	// start background goroutine to clean up inactive IPs
	go i.cleanupVisitors()

	return i
}

// getLimiter retrieves or creates a limiter for the given IP
func (i *IPRateLimiter) getLimiter(ip string) *rate.Limiter {
	i.mu.RLock()
	v, exists := i.ips[ip]
	if exists {
		// update last seen time
		v.lastSeen = time.Now()
		i.mu.RUnlock()
		return v.limiter
	}
	i.mu.RUnlock()

	i.mu.Lock()
	defer i.mu.Unlock()

	v, exists = i.ips[ip]
	if !exists {
		limiter := rate.NewLimiter(i.r, i.b)
		i.ips[ip] = &clientLimiter{
			limiter:  limiter,
			lastSeen: time.Now(),
		}
		return limiter
	}

	v.lastSeen = time.Now()
	return v.limiter
}

// cleanupVisitors removes IP limiters that have not been seen for 3 minutes
// runs every 1 minute
func (i *IPRateLimiter) cleanupVisitors() {
	for {
		time.Sleep(time.Minute)
		i.mu.Lock()
		for ip, v := range i.ips {
			if time.Since(v.lastSeen) > 3*time.Minute {
				delete(i.ips, ip)
			}
		}
		i.mu.Unlock()
	}
}

// RateLimitMiddleware gin middleware for ip-based rate limiting
func RateLimitMiddleware(limiter *IPRateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()

		if !limiter.getLimiter(ip).Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "429 Too Many Requests",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
