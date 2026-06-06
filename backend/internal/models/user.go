package models

import (
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Username   string `gorm:"uniqueIndex;not null"`
	Provider   string `gorm:"not null"` // 例如: "guest" 或 "discord"
	ProviderID string 

	// 經濟系統資源
	Coins        int `gorm:"default:0"`
	Stars        int `gorm:"default:0"`
	Diamonds     int `gorm:"default:0"`
	HighestScore int `gorm:"default:0"` 

	// ✨ 每日任務進度追蹤
	DailyApples       int  `gorm:"default:0"`     // 今日累積吃蘋果數
	DailyKills        int  `gorm:"default:0"`     // 今日累積擊殺數
	DailyAppleClaimed bool `gorm:"default:false"` // 蘋果任務是否已領獎
	DailyKillClaimed  bool `gorm:"default:false"` // 擊殺任務是否已領獎
}