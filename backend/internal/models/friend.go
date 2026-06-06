package models

import (
	"gorm.io/gorm"
)

type Friend struct {
	gorm.Model
	Requester string `gorm:"index;not null"` // 發送邀請的人
	Target    string `gorm:"index;not null"` // 接收邀請的人
	Status    string `gorm:"default:'pending'"` // 狀態："pending" 或 "accepted"
}