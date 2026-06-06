package main

import (
	"log"
	"os"

	"multiplayer-snake-web-backend/internal/handler"
	"multiplayer-snake-web-backend/internal/models"
	"multiplayer-snake-web-backend/internal/store"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3" // ✨ 引入 Cron 套件
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	_ = godotenv.Load()

	db, err := gorm.Open(sqlite.Open("game.db"), &gorm.Config{})
	if err != nil { log.Fatal("failed to connect database") }

	db.AutoMigrate(&models.User{}, &models.Message{}, &models.Session{}, &models.Friend{})

	// ✨ 設定每日半夜 12 點自動重置任務進度
	c := cron.New()
	c.AddFunc("0 0 * * *", func() {
		db.Model(&models.User{}).Where("1 = 1").Updates(map[string]interface{}{
			"daily_apples": 0,
			"daily_kills": 0,
			"daily_apple_claimed": false,
			"daily_kill_claimed": false,
		})
		log.Println("🔔 系統通知：所有玩家每日任務已重置")
	})
	c.Start()
	defer c.Stop()

	hub := store.NewHub(db)
	sessionManager := store.NewSessionManager(db)

	r := gin.Default()
	config := cors.DefaultConfig()
	config.AllowOriginFunc = func(origin string) bool { return true }
	config.AllowCredentials = true
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	r.Use(cors.New(config))

	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" { frontendURL = "http://localhost:5173" }
	discordRedirectURI := os.Getenv("DISCORD_REDIRECT_URI")
	if discordRedirectURI == "" { discordRedirectURI = "http://localhost:8080/api/auth/discord/callback" }

	authHandler := handler.AuthHandler{
		DB: db, Session: sessionManager,
		DiscordClientID: os.Getenv("DISCORD_CLIENT_ID"), DiscordClientSecret: os.Getenv("DISCORD_CLIENT_SECRET"),
		DiscordRedirectURI: discordRedirectURI, FrontendURL: frontendURL,
	}

	api := r.Group("/api")
	{
		api.POST("/login/guest", authHandler.GuestLogin)
		api.POST("/logout", authHandler.Logout)
		api.GET("/me", authHandler.Me)
		api.GET("/rankings", authHandler.GetRankings) 

		api.GET("/friends", authHandler.GetFriends)
		api.POST("/friends/request", authHandler.SendFriendRequest)
		api.POST("/friends/accept", authHandler.AcceptFriendRequest)
		api.POST("/friends/reject", authHandler.RejectFriendRequest)
		api.DELETE("/friends/:username", authHandler.RemoveFriend)

		// ✨ 註冊任務系統 API
		api.GET("/tasks", authHandler.GetTasks)
		api.POST("/tasks/claim/:id", authHandler.ClaimTask)
		
		api.GET("/auth/discord/login", authHandler.DiscordLogin)
		api.GET("/auth/discord/callback", authHandler.DiscordCallback)
		api.GET("/ws", func(ctx *gin.Context) { handler.HandleWs(hub, sessionManager, ctx) })
	}

	r.Run(":8080")
}