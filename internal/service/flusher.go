package service

import (
	"log"
	"time"

	"github.com/stywzn/Go-Interaction-Service/config"
	"github.com/stywzn/Go-Interaction-Service/internal/model"
	"gorm.io/gorm/clause"
)

// StartAsyncFlusher 是点赞系统在后台跳动的“心脏”
// 它会在 main 函数启动时，作为一个独立的 Goroutine 一直运行
func StartAsyncFlusher() {
	// 定义一个定时器，每 5 秒钟滴答一次 (削峰填谷的核心)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// 准备一个切片，像一个小推车，用来在内存里暂存这 5 秒内收到的任务
	var batchRecords []model.LikeRecord

	log.Println("后台异步落盘引擎已启动，正在监听 LikeQueue...")

	for {
		// select 多路复用，同时监听任务管道和定时器
		select {
		case task := <-LikeQueue:
			// 从管道里拿到了一个点赞任务，转换成 GORM 模型放到推车里
			record := model.LikeRecord{
				UserID:    task.UserID,
				ArticleID: task.ArticleID,
				Status:    task.Action, // 1 或 0
			}
			batchRecords = append(batchRecords, record)

			// 如果突发流量太大，不到 5 秒推车就装满了（比如 500 条）
			// 绝不能死等时间，立刻强行发车落盘，防止占用过多内存！
			if len(batchRecords) >= 500 {
				flushToDB(batchRecords)
				batchRecords = nil // 清空推车，准备接下一批
			}

		case <-ticker.C:
			// 5 秒时间到了！不管推车里装了多少条（哪怕只有 1 条），也必须发车写进数据库
			if len(batchRecords) > 0 {
				flushToDB(batchRecords)
				batchRecords = nil // 清空推车
			}
		}
	}
}

// flushToDB 执行真正的 MySQL 批量写入操作
func flushToDB(records []model.LikeRecord) {
	if len(records) == 0 {
		return
	}

	// UPSERT 冲突处理
	// 使用 GORM 的 CreateInBatches 进行批量插入，极大降低数据库 IO 压力。
	// 使用 clause.OnConflict 处理重复插入冲突：
	// 如果用户在 5 秒内点赞又取消，会触发 MySQL 的 uk_user_article 唯一索引冲突。
	// 这时我们让它自动变成 UPDATE 语句，更新 status 和 updated_at 字段。
	err := config.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "article_id"}},   // 遇到这两个字段冲突时
		DoUpdates: clause.AssignmentColumns([]string{"status", "updated_at"}), // 执行更新操作
	}).CreateInBatches(records, 100).Error // 每次最多插 100 条

	if err != nil {
		log.Printf("批量写入 MySQL 失败: %v\n", err)
	} else {
		log.Printf("[Flusher] 成功将 %d 条点赞数据异步落盘到 MySQL!\n", len(records))
	}
}
