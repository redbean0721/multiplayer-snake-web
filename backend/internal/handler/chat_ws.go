package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"multiplayer-snake-web-backend/internal/store"
	"multiplayer-snake-web-backend/pkg/request"
)

var chatWSUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (h *ChatHandler) WS(c *gin.Context) {
	userIDStr, err := c.Cookie("user_id")
	if err != nil || userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}

	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user cookie"})
		return
	}

	conn, err := chatWSUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	client := &store.ChatClient{
		Conn:   conn,
		Send:   make(chan []byte, 16),
		UserID: userID,
	}

	defer func() {
		if client.ChannelID != 0 {
			hub.RemoveClient(client)
		}
		_ = conn.Close()
	}()

	go func() {
		for msg := range client.Send {
			_ = conn.WriteMessage(websocket.TextMessage, msg)
		}
	}()

	for {
		var req request.ChatWSRequest
		if err := conn.ReadJSON(&req); err != nil {
			return
		}

		switch req.Type {
		case request.ChatWSEventJoin:
			if req.ChannelID == 0 {
				writeWSError(client, "channel_id is required")
				continue
			}
			if client.ChannelID != 0 {
				hub.RemoveClient(client)
			}
			client.ChannelID = req.ChannelID
			hub.AddClient(client)
			writeWSJSON(client, request.ChatWSResponse{
				Type:      request.ChatWSEventAck,
				ChannelID: client.ChannelID,
				UserID:    client.UserID,
				Message:   "joined",
			})
		case request.ChatWSEventMessage:
			if client.ChannelID == 0 {
				writeWSError(client, "join a channel first")
				continue
			}
			if req.Message == "" {
				writeWSError(client, "message is required")
				continue
			}
			writeWSJSON(client, request.ChatWSResponse{
				Type:      request.ChatWSEventAck,
				ChannelID: client.ChannelID,
				UserID:    client.UserID,
				Message:   "sent",
			})
			hub.Broadcast(client.ChannelID, mustJSON(request.ChatWSResponse{
				Type:      request.ChatWSEventMessage,
				ChannelID: client.ChannelID,
				UserID:    client.UserID,
				Message:   req.Message,
			}))
		case request.ChatWSEventPing:
			writeWSJSON(client, request.ChatWSResponse{
				Type:      request.ChatWSEventPong,
				ChannelID: client.ChannelID,
				UserID:    client.UserID,
				Message:   "pong",
			})
		default:
			writeWSError(client, "unknown event type")
		}
	}
}

func writeWSJSON(client *store.ChatClient, payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	select {
	case client.Send <- data:
	default:
	}
}

func writeWSError(client *store.ChatClient, message string) {
	writeWSJSON(client, request.ChatWSResponse{
		Type:    request.ChatWSEventError,
		Error:   message,
		UserID:  client.UserID,
		Message: message,
	})
}

func mustJSON(payload interface{}) []byte {
	data, _ := json.Marshal(payload)
	return data
}

var hub = store.NewChatHub()

func init() {
	_ = time.Now()
}
