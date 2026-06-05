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

func (h *AuthHandler) GuestLogin(c *gin.Context) {
	var req GuestLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "請輸入使用者名稱"})
		return
	}

	var user models.User
	result := h.DB.Where("username = ? AND provider = ?", req.Username, "guest").First(&user)
	if result.Error != nil {
		user = models.User{
			Username: req.Username,
			Provider: "guest",
		}
		h.DB.Create(&user)
	}

	token := h.Session.CreateSession(user.ID, user.Username)

	c.SetCookie("game_session", token, 86400, "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{
		"token":    token,
		"username": user.Username,
	})
}

// ✨ 新增：登出邏輯
func (h *AuthHandler) Logout(c *gin.Context) {
	// 1. 取得 Cookie 中的 Token
	token, err := c.Cookie("game_session")
	if err == nil {
		// 2. 從記憶體中銷毀
		h.Session.DeleteSession(token)
	}

	// 3. 讓瀏覽器的 Cookie 立即過期 (MaxAge 設為 -1)
	c.SetCookie("game_session", "", -1, "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{"message": "登出成功"})
}

func (h *AuthHandler) DiscordLogin(c *gin.Context) {
}

func (h *AuthHandler) DiscordCallback(c *gin.Context) {
}