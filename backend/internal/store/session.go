package store

import (
	"sync"
	"github.com/google/uuid"
)

type SessionData struct {
	UserID   uint
	Username string
}

type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]SessionData
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]SessionData),
	}
}

// 建立新 Session 並回傳 Token
func (s *SessionManager) CreateSession(userID uint, username string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	token := uuid.New().String()
	s.sessions[token] = SessionData{
		UserID:   userID,
		Username: username,
	}
	return token
}

// 驗證 Token 是否有效
func (s *SessionManager) GetSession(token string) (SessionData, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	data, exists := s.sessions[token]
	return data, exists
}