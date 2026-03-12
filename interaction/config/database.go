package config

import (
	"context"
	"fmt"
	"log"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var (
	DB    *gorm.DB
	Redis *redis.Client
)

func InitConfig() {
	// 连接 MySQL：注意这里我已经帮你改成了 docker-compose 里的密码和新数据库名
	dsn := "root:root@tcp(127.0.0.1:3306)/cloud_system?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf(" MySQL 连接失败: %v", err)
	}

	// 删除了旧的点赞表 AutoMigrate，交互服务现在专注于抽奖，不需要在这建表

	DB = db
	fmt.Println(" MySQL 连接成功，且数据库已就绪!")

	// 连接 Redis
	Redis = redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379",
		Password: "", // 本地 Redis 没设密码
		DB:       0,
	})

	if _, err := Redis.Ping(context.Background()).Result(); err != nil {
		log.Fatalf(" Redis 连接失败: %v", err)
	}
	fmt.Println(" Redis 连接成功!")
}
