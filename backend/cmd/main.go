package main

import (
	"github.com/gin-gonic/gin"
	"multiplayer-snake-web-backend/internal/handler"
	"multiplayer-snake-web-backend/internal/store"
)

func main() {
	r := gin.Default()
	hub := store.NewHub() // 初始化 Hub

	// 統一入口
	r.GET("/api/ws", func(c *gin.Context) {
		handler.HandleWs(hub, c)
	})

	r.Run("10.0.0.110:8080")
}