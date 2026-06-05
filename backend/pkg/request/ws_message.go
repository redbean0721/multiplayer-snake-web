package request

import "encoding/json"

// WsMessage 是所有透過 WebSocket 傳遞訊息的通用格式
type WsMessage struct {
    Type    string          `json:"type"`    // e.g., "ping", "chat", "move"
    Payload json.RawMessage `json:"payload"` // 具體數據
}