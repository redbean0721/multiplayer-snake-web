package models

import (
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Username   string `gorm:"uniqueIndex;not null"`
	Provider   string `gorm:"not null"`
	ProviderID string 

	Coins        int `gorm:"default:0"`
	Stars        int `gorm:"default:0"`
	Diamonds     int `gorm:"default:0"`
	HighestScore int `gorm:"default:0"` 

	DailyApples       int  `gorm:"default:0"`
	DailyKills        int  `gorm:"default:0"`
	DailyAppleClaimed bool `gorm:"default:false"`
	DailyKillClaimed  bool `gorm:"default:false"`

	// ✨ 新增皮膚系統欄位
	CurrentSkin string `gorm:"default:''"`
	OwnedSkins  string `gorm:"default:'[]'"` // 存放 JSON 陣列如 '["#ef4444", "rainbow"]'
}