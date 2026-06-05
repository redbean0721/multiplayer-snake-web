package handler

import (
	"encoding/json"
	"log"
	"multiplayer-snake-web-backend/internal/store"
	"multiplayer-snake-web-backend/pkg/request"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

type ChatPayload struct {
	ID      int64  `json:"id"`
	User    string `json:"user"`
	Content string `json:"content"`
	Time    string `json:"time"`
}

// ✨ 加上 sm *store.SessionManager 參數
func HandleWs(hub *store.Hub, sm *store.SessionManager, c *gin.Context) {
	// 1. 取得 Token (優先看 Query，沒有的話找 Cookie)
	token := c.Query("token")
	if token == "" {
		cookieToken, err := c.Cookie("game_session")
		if err == nil {
			token = cookieToken
		}
	}

	// 2. 驗證 Session 狀態
	sessionData, exists := sm.GetSession(token)
	if !exists {
		log.Println("拒絕未授權的 WebSocket 連線")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// 3. 升級為 WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	// ✨ 4. 直接使用 Session 中的名稱，保證安全不可竄改
	client := &store.Client{Conn: conn, Name: sessionData.Username}
	hub.Register(client)
	defer hub.Unregister(client)

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var wsMsg request.WsMessage
		if err := json.Unmarshal(msg, &wsMsg); err != nil {
			continue
		}

		switch wsMsg.Type {
		case "ping":
			client.SendRaw(msg)

		case "chat":
			var chatData ChatPayload
			if err := json.Unmarshal(wsMsg.Payload, &chatData); err == nil {
				chatData.ID = time.Now().UnixNano()
				chatData.Time = time.Now().Format("15:04")
				// 強制覆蓋為 Session 的名字，防止偽造身分聊天
				chatData.User = client.Name
				hub.Broadcast("chat", chatData)
			}

		case "start_game":
			// ✨ 因為 client.Name 已經綁定了，不需要前端再傳名字過來
			hub.SpawnSnake(client)

		case "move":
			var m store.Point
			if err := json.Unmarshal(wsMsg.Payload, &m); err == nil {
				hub.ChangeDirection(client, m.X, m.Y)
			}
		}
	}
}