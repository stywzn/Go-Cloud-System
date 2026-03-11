package config

import (
	"context"
	"fmt"
	"log"

	"github.com/redis/go-redis/v9"
	"github.com/stywzn/Go-Interaction-Service/internal/model" 
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var (
	DB    *gorm.DB
	Redis *redis.Client
)

func InitConfig() {
	// 连接mysql  注意mysql 的账号密码和 interaction_db 的空数据库
	dsn := "root:root@tcp(127.0.0.1:3306)/interaction_db?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf(" MySQL 连接失败: %v", err)
	}

	// 利用 GORM 的 AutoMigrate 自动建表
	err = db.AutoMigrate(&model.LikeRecord{})
	if err != nil {
		log.Fatalf(" MySQL 自动建表失败: %v", err)
	}
	DB = db  
	fmt.Println(" MySQL 连接成功，且数据表已就绪!")

	// 连接 redis
	Redis = redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379",
		Password: "",
		DB:       0,
	})

	if _, err := Redis.Ping(context.Background()).Result(); err != nil {
		log.Fatalf(" Redis 连接失败: %v", err)
	}
	fmt.Println(" Redis 连接成功!")

}
