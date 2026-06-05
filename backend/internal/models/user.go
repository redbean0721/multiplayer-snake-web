package models

import (
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Username   string `gorm:"uniqueIndex;not null"`
	Provider   string `gorm:"not null"` // 例如: "guest" 或 "discord"
	ProviderID string // Discord 的 User ID，訪客可留空

	// ✨ 經濟系統資源
	Coins    int `gorm:"default:0"`
	Stars    int `gorm:"default:0"`
	Diamonds int `gorm:"default:0"`
}