package models

import (
	"gorm.io/gorm"
)

type Message struct {
	gorm.Model
	Username string `gorm:"index;not null"` // 加上 index 加快查詢
	Content  string `gorm:"not null"`
}