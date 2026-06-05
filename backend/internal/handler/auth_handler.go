package handler

import (
	"multiplayer-snake-web-backend/internal/models"
	"multiplayer-snake-web-backend/internal/store"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AuthHandler struct {
	DB      *gorm.DB
	Session *store.SessionManager
}

type GuestLoginRequest struct {
	Username string `json:"username" binding:"required"`
}

// 訪客登入
func (h *AuthHandler) GuestLogin(c *gin.Context) {
	var req GuestLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "請輸入使用者名稱"})
		return
	}

	var user models.User
	// 尋找是否已有同名訪客，沒有就建立一個
	result := h.DB.Where("username = ? AND provider = ?", req.Username, "guest").First(&user)
	if result.Error != nil {
		user = models.User{
			Username: req.Username,
			Provider: "guest",
		}
		h.DB.Create(&user)
	}

	// 核發 Session Token
	token := h.Session.CreateSession(user.ID, user.Username)

	// ✨ 設定 HttpOnly Cookie (名稱: game_session, 期限: 24小時)
	c.SetCookie("game_session", token, 86400, "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{
		"token":    token,
		"username": user.Username,
	})
}

func (h *AuthHandler) DiscordLogin(c *gin.Context) {
	// 稍後實作
}

func (h *AuthHandler) DiscordCallback(c *gin.Context) {
	// 稍後實作
}