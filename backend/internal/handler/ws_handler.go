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

func HandleWs(hub *store.Hub, sm *store.SessionManager, c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		cookieToken, err := c.Cookie("game_session")
		if err == nil { token = cookieToken }
	}

	sessionData, exists := sm.GetSession(token)
	if !exists {
		log.Println("拒絕未授權的 WebSocket 連線")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil { return }

	client := &store.Client{Conn: conn, Name: sessionData.Username}
	hub.Register(client)
	
	// ✨ 連線成功後，立刻抓取資料庫的財產，傳給前端更新 UI
	hub.SyncResources(client)
	
	defer hub.Unregister(client)

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil { break }

		var wsMsg request.WsMessage
		if err := json.Unmarshal(msg, &wsMsg); err != nil { continue }

		switch wsMsg.Type {
		case "ping":
			client.SendRaw(msg)
		case "chat":
			var chatData ChatPayload
			if err := json.Unmarshal(wsMsg.Payload, &chatData); err == nil {
				chatData.ID = time.Now().UnixNano()
				chatData.Time = time.Now().Format("15:04")
				chatData.User = client.Name
				hub.Broadcast("chat", chatData)
			}
		case "start_game":
			hub.SpawnSnake(client)
		case "move":
			var m store.Point
			if err := json.Unmarshal(wsMsg.Payload, &m); err == nil {
				hub.ChangeDirection(client, m.X, m.Y)
			}
		}
	}
}