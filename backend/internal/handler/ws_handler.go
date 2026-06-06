package handler

import (
	"encoding/json"
	"log"
	"multiplayer-snake-web-backend/internal/models"
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
	hub.SyncResources(client)
	
	var messages []models.Message
	hub.DB.Order("created_at desc").Limit(30).Find(&messages)

	var history []ChatPayload
	for _, m := range messages {
		history = append(history, ChatPayload{
			ID:      int64(m.ID),
			User:    m.Username,
			Content: m.Content,
			Time:    m.CreatedAt.Format("15:04"),
		})
	}
	client.SendJSON("chat_history", history)

	sysJoinMsg := ChatPayload{
		ID:      time.Now().UnixNano(),
		User:    "系統",
		Content: "玩家 " + client.Name + " 已進入大廳",
		Time:    time.Now().Format("15:04"),
	}
	hub.Broadcast("chat", sysJoinMsg)

	defer func() {
		sysLeaveMsg := ChatPayload{
			ID:      time.Now().UnixNano(),
			User:    "系統",
			Content: "玩家 " + client.Name + " 離開了",
			Time:    time.Now().Format("15:04"),
		}
		hub.Broadcast("chat", sysLeaveMsg)
		hub.Unregister(client)
	}()

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
				newMsg := models.Message{
					Username: client.Name,
					Content:  chatData.Content,
				}
				hub.DB.Create(&newMsg)

				chatData.ID = int64(newMsg.ID)
				chatData.Time = newMsg.CreatedAt.Format("15:04")
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
			
		// ✨ 新增：前端要求更新資源時，立刻推送最新的錢包狀態
		case "sync_resources":
			hub.SyncResources(client)
		}
	}
}