package chat

import (
	"github.com/gin-gonic/gin"

	"multiplayer-snake-web-backend/internal/handler"
)

func RegisterRoutes(group *gin.RouterGroup, h *handler.ChatHandler) {
	group.GET("/connect", h.Connect)
	group.GET("/ws", h.WS)
}
