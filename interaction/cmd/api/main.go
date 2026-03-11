package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stywzn/Go-Interaction-Service/config"
	"github.com/stywzn/Go-Interaction-Service/internal/api"
	"github.com/stywzn/Go-Interaction-Service/internal/service"
)

func main() {
	// 初始化组件
	config.InitConfig()
	// 启动后台异步落盘的流水线消费者
	go service.StartAsyncFlusher()
	// 初始化 Gin 框架引擎
	r := gin.Default()

	// 服务的第一层安全防护  
	// 强制浏览器不要瞎猜内容类型，严格按照我们返回的 Content-Type 解析
	// 禁止别的网站用 iframe 嵌套我们的接口（防点击劫持）
	// 开启浏览器级别的 XSS 防护
	// 放行，进入下一个环节
	r.Use(api.SecurityHeadersMiddleware())

	// 健康检查接口
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"msg": "pong! 服务已启动"})
	})

	// 暴露点赞接口
	v1 := r.Group("/api/v1")

	// 开启 IP 限流
	v1.Use(api.RateLimitMiddleware())
	{
		v1.POST("/like", api.HandleLike)
		v1.GET("/like/count", api.HandleGetLikeCount)
		v1.GET("/leaderboard", api.HandleGetLeaderboard)
	}

	// 配置 HTTP Server
	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("服务监听异常: %s\n", err)
		}
	}()

	// 优雅停机
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("收到关机信号，正在阻止新请求进入...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Gin 强制关闭: ", err)
	}
	log.Println("剩余任务已安全处理完毕，服务完美退出！")

}
