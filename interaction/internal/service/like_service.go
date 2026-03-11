package service

import (
	"context"
	"fmt"
	"log"

	"github.com/stywzn/Go-Interaction-Service/config"
)

// LikeTask 定义要在 Channel 里流转的“点赞任务包”
type LikeTask struct {
	UserID    int
	ArticleID int
	Action    int // 1:点赞, 0:取消
}

// LikeQueue ：本地高并发缓冲队列
// 容量 10000，足以抗住绝大多数突发流量，保护底层 MySQL
var LikeQueue = make(chan LikeTask, 10000)

// ToggleLike ：快进快出，不碰 MySQL
func ToggleLike(ctx context.Context, userID int, articleID int, action int) error {
	//  定义 Redis 的 Key
	// setKey 用于记录“谁给这篇文章点了赞” (去重防刷)
	setKey := fmt.Sprintf("like:users:%d", articleID)
	// countKey 用于记录“每篇文章的总赞数” (Hash 结构)
	countKey := "like:counts"
	articleField := fmt.Sprintf("%d", articleID)

	// 新增：定义全站排行榜的 ZSet Key
	rankingKey := "article:ranking"

	if action == 1 {
		// 执行点赞】
		// 利用 Redis 单线程架构下 SADD 的原生原子性，瞬间拦截并发刷赞
		added, err := config.Redis.SAdd(ctx, setKey, userID).Result()
		if err != nil {
			return fmt.Errorf("操作太快啦，请稍后再试 (Redis写入异常)")
		}
		if added == 0 {
			// 如果 SADD 返回 0，说明集合里已经有这个人了
			// 直接报错退回，不管黑客开多少线程，全部死在这里
			return fmt.Errorf("您已经点过赞了，请勿重复操作")
		}

		// 累加总赞数 (原子加1)
		config.Redis.HIncrBy(ctx, countKey, articleField, 1)

		// 新增：给排行榜里这篇文章加 1 分 (ZINCRBY 天然是原子操作)
		config.Redis.ZIncrBy(ctx, rankingKey, 1, articleField)

	} else {
		// 执行取消赞
		removed, err := config.Redis.SRem(ctx, setKey, userID).Result()
		if err != nil {
			return fmt.Errorf("操作太快啦，请稍后再试 (Redis删除异常)")
		}
		if removed == 0 {
			return fmt.Errorf("您还没有点赞，无法取消")
		}

		// 扣减总赞数 (原子减1)
		config.Redis.HIncrBy(ctx, countKey, articleField, -1)

		// 新增：给排行榜里这篇文章减 1 分
		config.Redis.ZIncrBy(ctx, rankingKey, -1, articleField)
	}

	// 丢入异步队列削峰
	// 此时 Redis 数据已经更新，直接把落盘任务扔进 Channel
	select {
	case LikeQueue <- LikeTask{UserID: userID, ArticleID: articleID, Action: action}:
		// 极速塞入队列，主业务逻辑瞬间跑完
		log.Printf("[入队成功] 用户 %d 对文章 %d 执行动作 %d\n", userID, articleID, action)
	default:
		// 如果瞬间涌入几万请求把管子塞满了，绝不卡死 Goroutine！
		// 宁愿丢弃这条落盘任务（因为 Redis 里数据是对的），也要保证服务存活
		log.Println("[严重告警] 异步落盘队列已满！触发降级，当前请求丢弃落盘！")
	}

	return nil
}

// GetArticleLikeCount 极速获取单篇文章总赞数
func GetArticleLikeCount(ctx context.Context, articleID int) (string, error) {
	countKey := "like:counts"
	articleField := fmt.Sprintf("%d", articleID)

	// 直接从 Redis Hash 中以 O(1) 的时间复杂度取值
	countStr, err := config.Redis.HGet(ctx, countKey, articleField).Result()
	if err != nil {
		// 如果 Redis 里没有，真实业务中这里应该去查 MySQL 兜底，这里为了极简先返回 0
		return "0", nil
	}
	return countStr, nil
}

// GetLeaderboard 获取全站点赞排行榜 Top N
//
//	ZREVRANGE 的运用
func GetLeaderboard(ctx context.Context, topN int64) ([]string, error) {
	rankingKey := "article:ranking"

	// ZRevRange 按照分数从大到小（Rev代表Reverse）取前 N 名
	// 0 是第一名，topN-1 是第 N 名。时间复杂度是极低的 O(log(N)+M)
	topArticles, err := config.Redis.ZRevRange(ctx, rankingKey, 0, topN-1).Result()
	if err != nil {
		return nil, fmt.Errorf("获取排行榜失败: %v", err)
	}
	return topArticles, nil
}
