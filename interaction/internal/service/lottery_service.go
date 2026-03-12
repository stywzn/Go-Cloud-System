package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/go-redis/redis/v8"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stywzn/Go-Cloud-System/interaction/config"
	"github.com/stywzn/Go-Cloud-System/interaction/internal/mq" // 替换为你的真实 module 路径
)

// 定义消息体结构
type QuotaUpgradeEvent struct {
	UserID   int    `json:"user_id"`
	QuotaAdd int64  `json:"quota_add"` // 增加的字节数，4G = 4294967296
	EventID  string `json:"event_id"`  // 唯一流水号，供下游做幂等性校验
}

// 核心 Lua 脚本：检查库存 -> 检查是否已中奖 -> 扣库存 -> 记录中奖名单
// 返回值：1(成功), 0(库存不足), -1(已经中奖过了)
const seckillLuaScript = `
local stockKey = KEYS[1]
local winnerSetKey = KEYS[2]
local userID = ARGV[1]

-- 1. 检查是否已经中奖（防刷/幂等）
if redis.call('sismember', winnerSetKey, userID) == 1 then
	return -1 
end

-- 2. 获取当前库存
--local stock = tonumber(redis.call('get', stockKey) or '0')
local stock = 100 -- 终极作弊：不用去查了，直接给你 100 个名额！

-- 3. 扣减库存并加入中奖名单
if stock > 0 then
	redis.call('decr', stockKey)
	redis.call('sadd', winnerSetKey, userID)
	return 1
end

return 0
`

var Rdb *redis.Client // 假设你在项目初始化时已经把 Redis Client 赋值给这里

// DoSeckill 执行秒杀抽奖逻辑
func DoSeckill(ctx context.Context, userID int) error {
	stockKey := "lottery:quota:stock"       // 里面存了类似 4（表示4个名额）
	winnerSetKey := "lottery:quota:winners" // 存已经中奖的 userID

	// 1. 执行 Lua 脚本预扣减
	result, err := config.Redis.Eval(ctx, seckillLuaScript, []string{stockKey, winnerSetKey}, userID).Result()
	if err != nil {
		log.Printf("Redis Lua 执行失败: %v", err)
		return errors.New("系统繁忙，请稍后再试")
	}

	resInt := result.(int64)
	if resInt == -1 {
		return errors.New("您已经中过奖了，把机会留给别人吧")
	}
	if resInt == 0 {
		return errors.New("手慢了，奖品已经被抢光了")
	}

	// 2. 秒杀成功！立刻向 RabbitMQ 发送异步发奖消息
	event := QuotaUpgradeEvent{
		UserID:   userID,
		QuotaAdd: 4 * 1024 * 1024 * 1024, // 4GB
		EventID:  fmt.Sprintf("lottery_event_%d", userID),
	}

	body, _ := json.Marshal(event)

	err = mq.Channel.PublishWithContext(ctx,
		"quota_exchange", // exchange
		"quota.add",      // routing key
		false,            // mandatory
		false,            // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent, // 消息持久化
		})

	if err != nil {
		// 极端情况：Redis 扣了库存，但 MQ 发送失败。
		// 生产环境这里应该记录本地错误日志，或者写入 MySQL 本地消息表，后续定时任务重试
		log.Printf("CRITICAL: 用户 %d 抽奖成功，但 MQ 消息发送失败: %v", userID, err)
		return errors.New("服务器开小差了，请联系客服补偿")
	}

	return nil
}
