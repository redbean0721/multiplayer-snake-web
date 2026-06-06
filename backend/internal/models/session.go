package models

import (
	"time"
)

type Session struct {
	Token     string `gorm:"primaryKey"`
	UserID    uint   `gorm:"not null"`
	Username  string `gorm:"not null"`
	CreatedAt time.Time
}