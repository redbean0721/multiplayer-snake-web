package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"multiplayer-snake-web-backend/pkg/response"
)

type ChatHandler struct{}

func NewChatHandler() *ChatHandler {
	return &ChatHandler{}
}

// Connect is a minimal endpoint to verify the user can enter chat.
// It reads the user identity from the cookie named user_id.
func (h *ChatHandler) Connect(c *gin.Context) {
	userIDStr, err := c.Cookie("user_id")
	if err != nil || userIDStr == "" {
		response.JSONResponse(c, http.StatusUnauthorized, "unauthenticated", 401)
		return
	}

	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		response.JSONResponse(c, http.StatusUnauthorized, "invalid user cookie", 401)
		return
	}

	response.JSONResponse(c, http.StatusOK, "chat connected", gin.H{
		"connected": true,
		"user_id":   userID,
	})
}
