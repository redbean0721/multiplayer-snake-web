package store

import (
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"
)

// Client 封裝了 WebSocket 連線與互斥鎖，保證併發寫入安全
type Client struct {
	Conn *websocket.Conn
	Mu   sync.Mutex
}

// 安全地寫入 JSON 格式
func (c *Client) SendJSON(msgType string, payload interface{}) {
	data, _ := json.Marshal(map[string]interface{}{"type": msgType, "payload": payload})
	c.Mu.Lock()
	defer c.Mu.Unlock()
	c.Conn.WriteMessage(websocket.TextMessage, data)
}

// 安全地寫入原始資料 (用於 Ping/Pong)
func (c *Client) SendRaw(data []byte) {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	c.Conn.WriteMessage(websocket.TextMessage, data)
}

type Hub struct {
	Clients map[*Client]bool
	Mu      sync.RWMutex // 保護 Map 的鎖
}

func NewHub() *Hub {
	return &Hub{Clients: make(map[*Client]bool)}
}

func (h *Hub) Register(c *Client) {
	h.Mu.Lock()
	h.Clients[c] = true
	h.Mu.Unlock()
}

func (h *Hub) Unregister(c *Client) {
	h.Mu.Lock()
	if _, ok := h.Clients[c]; ok {
		delete(h.Clients, c)
		c.Conn.Close()
	}
	h.Mu.Unlock()
}

// 廣播給所有人
func (h *Hub) Broadcast(msgType string, payload interface{}) {
	h.Mu.RLock()
	defer h.Mu.RUnlock()
	for client := range h.Clients {
		client.SendJSON(msgType, payload)
	}
}