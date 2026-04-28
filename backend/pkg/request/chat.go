package request

type ChatRoomSendMessageRequest_Model struct {
	ChannelID int64  `json:"channel_id" binding:"required"`
	Message   string `json:"message" binding:"required,min=1,max=2000"`
}

type ChatWSEventType string

const (
	ChatWSEventJoin    ChatWSEventType = "join"
	ChatWSEventMessage ChatWSEventType = "message"
	ChatWSEventPing    ChatWSEventType = "ping"
	ChatWSEventPong    ChatWSEventType = "pong"
	ChatWSEventAck     ChatWSEventType = "ack"
	ChatWSEventError   ChatWSEventType = "error"
)

type ChatWSRequest struct {
	Type      ChatWSEventType `json:"type" binding:"required"`
	ChannelID int64           `json:"channel_id,omitempty"`
	Message   string          `json:"message,omitempty"`
}

type ChatWSResponse struct {
	Type      ChatWSEventType `json:"type"`
	ChannelID int64           `json:"channel_id,omitempty"`
	UserID    uint64          `json:"user_id,omitempty"`
	Message   string          `json:"message,omitempty"`
	Error     string          `json:"error,omitempty"`
}
