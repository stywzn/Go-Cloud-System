package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/stywzn/Go-Cloud-System/interaction/config"
	"github.com/stywzn/Go-Cloud-System/interaction/internal/api"
	"github.com/stywzn/Go-Cloud-System/interaction/internal/mq"
	"github.com/stywzn/Go-Cloud-System/pkg/middleware"
)

func main() {
	config.InitConfig()
	// 初始化 RabbitMQ
	mq.InitRabbitMQ("amqp://guest:guest@localhost:5672/")
	defer mq.Close()

	r := gin.Default()

	// 挂载鉴权中间件并注册抽奖路由
	r.POST("/api/v1/lottery/seckill", middleware.RequireUser(), api.SeckillHandler)

	log.Println("Interaction 抽奖服务启动，监听端口 8082...")
	r.Run(":8082")
}
