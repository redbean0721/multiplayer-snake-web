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

func (h *AuthHandler) getSessionUser(c *gin.Context) (store.SessionData, bool) {
	token, err := c.Cookie("game_session")
	if err != nil { return store.SessionData{}, false }
	return h.Session.GetSession(token)
}

type GuestLoginRequest struct { Username string `json:"username" binding:"required"` }

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

func (h *AuthHandler) Me(c *gin.Context) {
	sessionData, ok := h.getSessionUser(c)
	if !ok {
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
// ✨ 雙向好友系統 API 實作
// ==========================================

func (h *AuthHandler) GetFriends(c *gin.Context) {
	session, ok := h.getSessionUser(c)
	if !ok { c.JSON(http.StatusUnauthorized, gin.H{"error": "請先登入"}); return }

	var records []models.Friend
	h.DB.Where("requester = ? OR target = ?", session.Username, session.Username).Find(&records)

	friends := []string{}
	pending := []string{}

	for _, r := range records {
		if r.Status == "accepted" {
			if r.Requester == session.Username {
				friends = append(friends, r.Target)
			} else {
				friends = append(friends, r.Requester)
			}
		} else if r.Status == "pending" && r.Target == session.Username {
			// 只有自己是被邀請方時，才顯示在「待確認邀請」中
			pending = append(pending, r.Requester)
		}
	}
	
	c.JSON(http.StatusOK, gin.H{"friends": friends, "pending_invites": pending})
}

func (h *AuthHandler) SendFriendRequest(c *gin.Context) {
	session, ok := h.getSessionUser(c)
	if !ok { return }

	var req struct { FriendName string `json:"friend_name" binding:"required"` }
	if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "請提供好友名稱"}); return }
	if req.FriendName == session.Username { c.JSON(http.StatusBadRequest, gin.H{"error": "不能加自己為好友"}); return }

	var targetUser models.User
	if err := h.DB.Where("username = ?", req.FriendName).First(&targetUser).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "伺服器上找不到此玩家"})
		return
	}

	var existing models.Friend
	err := h.DB.Where("(requester = ? AND target = ?) OR (requester = ? AND target = ?)", 
		session.Username, req.FriendName, req.FriendName, session.Username).First(&existing).Error

	if err == nil {
		if existing.Status == "accepted" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "你們已經是好友了"})
		} else {
			if existing.Requester == session.Username {
				c.JSON(http.StatusBadRequest, gin.H{"error": "已經發送過邀請，請等待對方確認"})
			} else {
				// 發現對方早就發過邀請給你，直接自動變成同意！
				existing.Status = "accepted"
				h.DB.Save(&existing)
				c.JSON(http.StatusOK, gin.H{"message": "對方已邀請過你，已自動成為好友！"})
			}
		}
		return
	}

	h.DB.Create(&models.Friend{Requester: session.Username, Target: req.FriendName, Status: "pending"})
	c.JSON(http.StatusOK, gin.H{"message": "好友邀請已送出！"})
}

func (h *AuthHandler) AcceptFriendRequest(c *gin.Context) {
	session, ok := h.getSessionUser(c)
	if !ok { return }

	var req struct { Requester string `json:"requester"` }
	c.ShouldBindJSON(&req)

	result := h.DB.Model(&models.Friend{}).
		Where("requester = ? AND target = ? AND status = ?", req.Requester, session.Username, "pending").
		Update("status", "accepted")

	if result.RowsAffected == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "找不到此邀請"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已接受好友邀請"})
}

func (h *AuthHandler) RejectFriendRequest(c *gin.Context) {
	session, ok := h.getSessionUser(c)
	if !ok { return }

	var req struct { Requester string `json:"requester"` }
	c.ShouldBindJSON(&req)

	h.DB.Where("requester = ? AND target = ? AND status = ?", req.Requester, session.Username, "pending").Delete(&models.Friend{})
	c.JSON(http.StatusOK, gin.H{"message": "已拒絕邀請"})
}

func (h *AuthHandler) RemoveFriend(c *gin.Context) {
	session, ok := h.getSessionUser(c)
	if !ok { return }

	friendName := c.Param("username")
	h.DB.Where("(requester = ? AND target = ?) OR (requester = ? AND target = ?)", 
		session.Username, friendName, friendName, session.Username).Delete(&models.Friend{})
	c.JSON(http.StatusOK, gin.H{"message": "已刪除該好友"})
}

// ==========================================
// Discord OAuth2
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

// ==========================================
// ✨ 每日任務系統 API 實作
// ==========================================

func (h *AuthHandler) GetTasks(c *gin.Context) {
	session, ok := h.getSessionUser(c)
	if !ok { c.JSON(http.StatusUnauthorized, gin.H{"error": "請先登入"}); return }

	var user models.User
	h.DB.Where("username = ?", session.Username).First(&user)

	// 動態產出任務列表
	tasks := []map[string]interface{}{
		{
			"id": "apple", "desc": "今日累計吃 50 顆蘋果",
			"progress": user.DailyApples, "target": 50,
			"reward_text": "🪙 500 金幣", "claimed": user.DailyAppleClaimed,
		},
		{
			"id": "kill", "desc": "今日累計擊殺 3 條蛇",
			"progress": user.DailyKills, "target": 3,
			"reward_text": "⭐ 5 星星", "claimed": user.DailyKillClaimed,
		},
	}
	
	c.JSON(http.StatusOK, tasks)
}

func (h *AuthHandler) ClaimTask(c *gin.Context) {
	session, ok := h.getSessionUser(c)
	if !ok { c.JSON(http.StatusUnauthorized, gin.H{"error": "請先登入"}); return }

	taskID := c.Param("id")
	var user models.User
	h.DB.Where("username = ?", session.Username).First(&user)

	if taskID == "apple" {
		if user.DailyApples >= 50 && !user.DailyAppleClaimed {
			h.DB.Model(&user).Updates(map[string]interface{}{
				"daily_apple_claimed": true,
				"coins": gorm.Expr("coins + ?", 500),
			})
			c.JSON(http.StatusOK, gin.H{"message": "成功領取 500 金幣！"})
			return
		}
	} else if taskID == "kill" {
		if user.DailyKills >= 3 && !user.DailyKillClaimed {
			h.DB.Model(&user).Updates(map[string]interface{}{
				"daily_kill_claimed": true,
				"stars": gorm.Expr("stars + ?", 5),
			})
			c.JSON(http.StatusOK, gin.H{"message": "成功領取 5 星星！"})
			return
		}
	}
	
	c.JSON(http.StatusBadRequest, gin.H{"error": "任務未達成或已領取"})
}