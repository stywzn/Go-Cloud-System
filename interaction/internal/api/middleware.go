package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stywzn/Go-Interaction-Service/config"
)

// SecurityHeadersMiddleware 第一道防线：全局安全响应头
// 防御目标：XSS 攻击、MIME 嗅探、点击劫持
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 强制浏览器不要瞎猜内容类型，严格按照我们返回的 Content-Type 解析
		c.Header("X-Content-Type-Options", "nosniff")
		// 禁止别的网站用 iframe 嵌套我们的接口（防点击劫持）
		c.Header("X-Frame-Options", "DENY")
		// 开启浏览器级别的 XSS 防护
		c.Header("X-XSS-Protection", "1; mode=block")

		c.Next() // 放行，进入下一个环节
	}
}

// RateLimitMiddleware 第二道防线：基于 Redis 的 IP 限流
// 防御目标：恶意脚本爆破、BurpSuite 并发重放攻击
func RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP() // 拿到黑客的真实 IP
		key := fmt.Sprintf("rate_limit:ip:%s", ip)

		// 利用 Redis 的 INCR 命令原子递增
		count, err := config.Redis.Incr(c.Request.Context(), key).Result()
		if err != nil {
			// 如果 Redis 挂了，为了保证核心业务可用，这里可以选择放行或报错
			// 我们这里选择严苛模式：报错拦截
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"msg": "安全拦截器异常"})
			return
		}

		// 如果是这个 IP 的第一次访问，给这个 Key 设置一个 10 秒的过期时间
		if count == 1 {
			config.Redis.Expire(c.Request.Context(), key, 10*time.Second)
		}

		// 核心审判逻辑：同一个 IP，10 秒内只允许发 50 次请求！
		if count > 50 {
			// 直接在网关层截断，连 Controller 都进不去！
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"code": 429,
				"msg":  "警告：您的操作太频繁，触发系统安全限流！",
			})
			return
		}

		c.Next() // 校验通过，放行！
	}
}
