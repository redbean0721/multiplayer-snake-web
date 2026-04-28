package main

import (
	"github.com/gin-gonic/gin"

	"multiplayer-snake-web-backend/internal/handler"
	chatrouter "multiplayer-snake-web-backend/internal/router/chat"
)

func main() {
	router := gin.Default()
	chatHandler := handler.NewChatHandler()

	chatGroup := router.Group("/api/v1/chat")
	chatrouter.RegisterRoutes(chatGroup, chatHandler)

	router.Run(":8080")
}
