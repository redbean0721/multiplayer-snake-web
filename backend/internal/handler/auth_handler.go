package handler

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"multiplayer-snake-web-backend/internal/models"
	"multiplayer-snake-web-backend/internal/store"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AuthHandler struct {
	DB                  *gorm.DB
	Session             *store.SessionManager
	DiscordClientID     string
	DiscordClientSecret string
	DiscordRedirectURI  string
	FrontendURL         string
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
		user = models.User{Username: req.Username, Provider: "guest"}
		h.DB.Create(&user)
	}

	token := h.Session.CreateSession(user.ID, user.Username)
	c.SetCookie("game_session", token, 86400, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"token": token, "username": user.Username})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	token, err := c.Cookie("game_session")
	if err == nil { h.Session.DeleteSession(token) }
	c.SetCookie("game_session", "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"message": "登出成功"})
}

// ✨ 新增：前端載入時驗證 Session 是否有效
func (h *AuthHandler) Me(c *gin.Context) {
	token, err := c.Cookie("game_session")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	sessionData, exists := h.Session.GetSession(token)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"username": sessionData.Username})
}

func (h *AuthHandler) GetRankings(c *gin.Context) {
	rankType := c.Query("type")
	var users []models.User
	var rankings []map[string]interface{}

	if rankType == "wealth" {
		h.DB.Order("coins desc").Limit(10).Find(&users)
		for _, u := range users {
			rankings = append(rankings, map[string]interface{}{"username": u.Username, "value": u.Coins})
		}
	} else {
		h.DB.Order("highest_score desc").Limit(10).Find(&users)
		for _, u := range users {
			rankings = append(rankings, map[string]interface{}{"username": u.Username, "value": u.HighestScore})
		}
	}
	c.JSON(http.StatusOK, rankings)
}

// ==========================================
// Discord OAuth2 登入實作
// ==========================================

func (h *AuthHandler) DiscordLogin(c *gin.Context) {
	authURL := fmt.Sprintf("https://discord.com/api/oauth2/authorize?client_id=%s&redirect_uri=%s&response_type=code&scope=identify",
		h.DiscordClientID, url.QueryEscape(h.DiscordRedirectURI))
	c.Redirect(http.StatusTemporaryRedirect, authURL)
}

func (h *AuthHandler) DiscordCallback(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		c.Redirect(http.StatusTemporaryRedirect, h.FrontendURL+"?error=登入取消")
		return
	}

	data := url.Values{}
	data.Set("client_id", h.DiscordClientID)
	data.Set("client_secret", h.DiscordClientSecret)
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", h.DiscordRedirectURI)

	req, _ := http.NewRequest("POST", "https://discord.com/api/oauth2/token", strings.NewReader(data.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		c.Redirect(http.StatusTemporaryRedirect, h.FrontendURL+"?error=授權失敗")
		return
	}
	defer resp.Body.Close()

	var tokenRes struct { AccessToken string `json:"access_token"` }
	body, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal(body, &tokenRes)

	reqUser, _ := http.NewRequest("GET", "https://discord.com/api/users/@me", nil)
	reqUser.Header.Add("Authorization", "Bearer "+tokenRes.AccessToken)
	respUser, err := client.Do(reqUser)
	if err != nil || respUser.StatusCode != 200 {
		c.Redirect(http.StatusTemporaryRedirect, h.FrontendURL+"?error=獲取資料失敗")
		return
	}
	defer respUser.Body.Close()

	var discordUser struct { ID string `json:"id"`; Username string `json:"username"` }
	bodyUser, _ := ioutil.ReadAll(respUser.Body)
	json.Unmarshal(bodyUser, &discordUser)

	var user models.User
	result := h.DB.Where("provider_id = ? AND provider = ?", discordUser.ID, "discord").First(&user)
	if result.Error != nil {
		user = models.User{Username: discordUser.Username, Provider: "discord", ProviderID: discordUser.ID}
		h.DB.Create(&user)
	}

	token := h.Session.CreateSession(user.ID, user.Username)
	c.SetCookie("game_session", token, 86400, "/", "", false, true)

	redirectURL := fmt.Sprintf("%s/?token=%s&username=%s", h.FrontendURL, token, url.QueryEscape(user.Username))
	c.Redirect(http.StatusTemporaryRedirect, redirectURL)
}