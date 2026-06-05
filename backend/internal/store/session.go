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

func (s *SessionManager) GetSession(token string) (SessionData, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	data, exists := s.sessions[token]
	return data, exists
}

// ✨ 新增：登出時銷毀伺服器記憶體中的 Session
func (s *SessionManager) DeleteSession(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, token)
}