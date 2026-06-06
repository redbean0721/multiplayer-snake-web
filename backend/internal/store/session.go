package store

import (
	"multiplayer-snake-web-backend/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SessionData struct {
	UserID   uint
	Username string
}

type SessionManager struct {
	DB *gorm.DB // ✨ 全面改為依賴資料庫
}

func NewSessionManager(db *gorm.DB) *SessionManager {
	return &SessionManager{
		DB: db,
	}
}

func (s *SessionManager) CreateSession(userID uint, username string) string {
	token := uuid.New().String()
	session := models.Session{
		Token:    token,
		UserID:   userID,
		Username: username,
	}
	s.DB.Create(&session) // 寫入 SQLite
	return token
}

func (s *SessionManager) GetSession(token string) (SessionData, bool) {
	var session models.Session
	result := s.DB.Where("token = ?", token).First(&session) // 從 SQLite 查詢
	if result.Error != nil {
		return SessionData{}, false
	}
	
	return SessionData{
		UserID:   session.UserID,
		Username: session.Username,
	}, true
}

func (s *SessionManager) DeleteSession(token string) {
	s.DB.Where("token = ?", token).Delete(&models.Session{})
}