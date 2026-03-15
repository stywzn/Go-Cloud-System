package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stywzn/Go-Cloud-System/interaction/config"
	"github.com/stywzn/Go-Cloud-System/interaction/internal/api"
	"github.com/stywzn/Go-Cloud-System/interaction/internal/mq"
	"github.com/stywzn/Go-Cloud-System/pkg/graceful"
	"github.com/stywzn/Go-Cloud-System/pkg/middleware"
	"github.com/stywzn/Go-Cloud-System/pkg/trace"
)

func main() {
	config.InitConfig()
	// 初始化 RabbitMQ
	mq.InitRabbitMQ("amqp://guest:guest@localhost:5672/")
	defer mq.Close()

	// 启动死信队列监控
	ctx := context.Background()
	mq.StartDLQMonitor(ctx)

	r := gin.Default()

	// 挂载链路追踪中间件
	r.Use(trace.ExtractTraceMiddleware())

	// 健康检查接口
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// 用户相关接口
	r.POST("/api/v1/auth/register", api.RegisterHandler)
	r.POST("/api/v1/auth/login", api.LoginHandler)
	r.GET("/api/v1/user/info", middleware.RequireUser(), api.GetUserInfoHandler)

	// 挂载鉴权中间件并注册抽奖路由
	r.POST("/api/v1/lottery/seckill", middleware.RequireUser(), api.SeckillHandler)

	// 创建HTTP服务器
	server := &http.Server{
		Addr:    ":8082",
		Handler: r,
	}

	// 创建优雅停机管理器
	shutdownManager := graceful.NewShutdownManager(server, 30*time.Second)

	// 添加停机钩子
	shutdownManager.AddShutdownHook(func() error {
		log.Println("正在关闭RabbitMQ连接...")
		mq.Close()
		return nil
	})

	// 启动优雅停机监控
	go shutdownManager.WaitForShutdown()

	log.Println("Interaction 抽奖服务启动，监听端口 8082...")

	// 启动服务器
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Interaction服务异常退出: %v", err)
	}

	// 等待停机完成
	<-shutdownManager.Done()
}
