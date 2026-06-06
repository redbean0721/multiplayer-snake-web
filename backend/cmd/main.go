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
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	_ = godotenv.Load()

	db, err := gorm.Open(sqlite.Open("game.db"), &gorm.Config{})
	if err != nil { log.Fatal("failed to connect database") }

	// ✨ 加入 models.Session 進行資料表生成
	db.AutoMigrate(&models.User{}, &models.Message{}, &models.Session{})

	hub := store.NewHub(db)
	// ✨ 把 DB 傳進去
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
		DB:                  db, 
		Session:             sessionManager,
		DiscordClientID:     os.Getenv("DISCORD_CLIENT_ID"),
		DiscordClientSecret: os.Getenv("DISCORD_CLIENT_SECRET"),
		DiscordRedirectURI:  discordRedirectURI,
		FrontendURL:         frontendURL,
	}

	api := r.Group("/api")
	{
		api.POST("/login/guest", authHandler.GuestLogin)
		api.POST("/logout", authHandler.Logout)
		
		api.GET("/me", authHandler.Me) // ✨ 註冊身分驗證 API
		api.GET("/rankings", authHandler.GetRankings) 
		
		api.GET("/auth/discord/login", authHandler.DiscordLogin)
		api.GET("/auth/discord/callback", authHandler.DiscordCallback)

		api.GET("/ws", func(c *gin.Context) { handler.HandleWs(hub, sessionManager, c) })
	}

	r.Run(":8080")
}