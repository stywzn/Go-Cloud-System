package main

import (
	"log"
	"net/http"
	"time"

	"github.com/stywzn/Go-Cloud-System/gateway/internal/config"
	"github.com/stywzn/Go-Cloud-System/gateway/internal/middleware"
	"github.com/stywzn/Go-Cloud-System/gateway/internal/proxy"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/time/rate"
)

func main() {

	cfg := config.LoadConfig("configs/config.yaml")

	r := gin.Default()

	limiter := middleware.NewIPRateLimiter(rate.Limit(2), 5)
	// 把限流中间件挂载到全局
	r.Use(middleware.RateLimitMiddleware(limiter))

	// 鉴权白名单
	publicPaths := map[string]bool{
		"/healthz":     true,
		"/readyz":      true,
		"/debug/token": true,
	}

	r.Use(func(c *gin.Context) {
		if publicPaths[c.Request.URL.Path] {
			c.Next()
			return
		}
		middleware.JWTAuthMiddleware()
	})

	// 基础探针与工具接口
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.GET("/readyz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.GET("/debug/token", func(c *gin.Context) {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"user_id": 9527,
			"exp":     time.Now().Add(time.Hour).Unix(),
		})
		tokenString, _ := token.SignedString([]byte(cfg.JWT.Secret))
		c.JSON(http.StatusOK, gin.H{"token": tokenString})
	})

	//动态多服务路由
	log.Println("========================================")
	log.Println("正在加载微服务路由表...")
	// 遍历 yaml 路由数组 动态挂载到Gin上
	for _, route := range cfg.Routes {
		prefix := route.PathPrefic
		target := route.TargetURL

		r.Any(prefix, proxy.GinReverseProxy(target))
		r.Any(prefix+"/*path", proxy.GinReverseProxy(target))
		log.Printf("映射成功: %-15s => %s", prefix, target)
	}
	log.Println("========================================")

	log.Printf("Go-Secure-Gateway (启动. 监听端口 %s", cfg.Server.Port)
	if err := r.Run(cfg.Server.Port); err != nil {
		log.Fatalf("网关服务异常退出: %v", err)
	}
}
