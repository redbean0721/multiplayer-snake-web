package main

import (
	"log"
	"multiplayer-snake-web-backend/internal/handler"
	"multiplayer-snake-web-backend/internal/models"
	"multiplayer-snake-web-backend/internal/store"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	// 1. 初始化 SQLite 資料庫
	db, err := gorm.Open(sqlite.Open("game.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect database")
	}

	// 自動遷移 Schema
	db.AutoMigrate(&models.User{})

	// 2. 初始化核心管理器
	hub := store.NewHub()
	sessionManager := store.NewSessionManager()

	// 3. 設定路由與 Handler
	r := gin.Default()

	// ✨ 修正 CORS 設定，支援跨域 Cookie (Credentials)
	config := cors.DefaultConfig()
	config.AllowOriginFunc = func(origin string) bool { return true } // 動態允許所有來源
	config.AllowCredentials = true                                    // 允許攜帶 Cookie 與 Authorization Header
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	r.Use(cors.New(config))

	authHandler := handler.AuthHandler{DB: db, Session: sessionManager}

	// API 路由群組
	api := r.Group("/api")
	{
		// 認證相關
		api.POST("/login/guest", authHandler.GuestLogin)
		api.GET("/auth/discord/login", authHandler.DiscordLogin)
		api.GET("/auth/discord/callback", authHandler.DiscordCallback)

		// WebSocket 入口 (✨ 將 sessionManager 傳入以驗證身分)
		api.GET("/ws", func(c *gin.Context) {
			handler.HandleWs(hub, sessionManager, c)
		})
	}

	r.Run("10.0.0.110:8080")
}