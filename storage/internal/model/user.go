package model

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Username string `gorm:"unique;not null"`
	Password string `gorm:"not null"`
	Quota    int64  `gorm:"default:5368709120"` // 5GB默认配额
}

func (User) TableName() string {
	return "users"
}
