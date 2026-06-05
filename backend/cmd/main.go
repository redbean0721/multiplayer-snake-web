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
	db, err := gorm.Open(sqlite.Open("game.db"), &gorm.Config{})
	if err != nil { log.Fatal("failed to connect database") }

	db.AutoMigrate(&models.User{})

	hub := store.NewHub(db)
	sessionManager := store.NewSessionManager()

	r := gin.Default()

	config := cors.DefaultConfig()
	config.AllowOriginFunc = func(origin string) bool { return true }
	config.AllowCredentials = true
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	r.Use(cors.New(config))

	authHandler := handler.AuthHandler{DB: db, Session: sessionManager}

	api := r.Group("/api")
	{
		api.POST("/login/guest", authHandler.GuestLogin)
		api.POST("/logout", authHandler.Logout) // ✨ 註冊登出 API
		
		api.GET("/auth/discord/login", authHandler.DiscordLogin)
		api.GET("/auth/discord/callback", authHandler.DiscordCallback)

		api.GET("/ws", func(c *gin.Context) {
			handler.HandleWs(hub, sessionManager, c)
		})
	}

	r.Run("10.0.0.110:8080")
}