package store

import (
	"sync"

	"github.com/gorilla/websocket"
)

type ChatHub struct {
	mu    sync.RWMutex
	rooms map[int64]map[*ChatClient]struct{}
}

type ChatClient struct {
	Conn      *websocket.Conn
	Send      chan []byte
	ChannelID int64
	UserID    uint64
}

func NewChatHub() *ChatHub {
	return &ChatHub{
		rooms: make(map[int64]map[*ChatClient]struct{}),
	}
}

func (h *ChatHub) AddClient(client *ChatClient) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.rooms[client.ChannelID]; !ok {
		h.rooms[client.ChannelID] = make(map[*ChatClient]struct{})
	}
	h.rooms[client.ChannelID][client] = struct{}{}
}

func (h *ChatHub) RemoveClient(client *ChatClient) {
	h.mu.Lock()
	defer h.mu.Unlock()

	clients, ok := h.rooms[client.ChannelID]
	if !ok {
		return
	}
	delete(clients, client)
	if len(clients) == 0 {
		delete(h.rooms, client.ChannelID)
	}
}

func (h *ChatHub) Broadcast(channelID int64, payload []byte) {
	h.mu.RLock()
	clients := h.rooms[channelID]
	h.mu.RUnlock()

	for client := range clients {
		select {
		case client.Send <- payload:
		default:
		}
	}
}
