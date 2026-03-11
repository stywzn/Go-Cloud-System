package model

import "time"

// LikeRecord 对应 MySQL 中的 likes 表
type LikeRecord struct {
	ID int64 `gorm:"primaryKey;autoIncrement"`
	// 联合唯一索引 uk_user_article，死死防住并发重复插入
	UserID    int       `gorm:"column:user_id;uniqueIndex:uk_user_article;not null"`
	ArticleID int       `gorm:"column:article_id;uniqueIndex:uk_user_article;not null"`
	Status    int       `gorm:"column:status;type:tinyint;default:1;not null"` // 1:点赞 0:取消赞
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

// TableName 强制指定表名，防止 GORM 乱加 "s"
func (LikeRecord) TableName() string {
	return "likes"
}
