package model

import "gorm.io/gorm"

type UploadTask struct {
	gorm.Model
	UserID          uint   `gorm:"index;not null"`
	UploadID        string `gorm:"uniqueIndex;type:char(32);not null"`
	FileName        string `gorm:"not null"`
	FileSize        int64  `gorm:"not null"`
	ChunkSize       int64  `gorm:"default:5242880"` // 5MB
	TotalChunks     int    `gorm:"not null"`
	CompletedChunks string `gorm:"type:json"` // JSON数组，记录已完成的chunk编号：[1,2,3]
	Status          int    `gorm:"default:0"` // 0:上传中, 1:已完成, 2:已取消
	CreatedAt       int64  `gorm:"autoCreateTime:milli"`
	UpdatedAt       int64  `gorm:"autoUpdateTime:milli"`
}

func (UploadTask) TableName() string {
	return "upload_tasks"
}
