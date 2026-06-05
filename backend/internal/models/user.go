package models

import (
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Username string `gorm:"uniqueIndex;not null"`
	Provider string `gorm:"not null"` // 例如: "guest" 或 "discord"
	ProviderID string // Discord 的 User ID，訪客可留空
}