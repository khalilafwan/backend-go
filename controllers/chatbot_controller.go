package controllers

import (
	"backend-go/config"
	"backend-go/models"
	"backend-go/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ChatbotHandler menangani permintaan interaksi chatbot
func ChatbotHandler(c *gin.Context) {
	var req models.ChatbotRequest

	// Validasi request JSON
	if err := c.ShouldBindJSON(&req); err != nil {
		config.Log.Error("Permintaan tidak valid:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Permintaan tidak valid"})
		return
	}

	// Gunakan UUID yang diterima dari request
	chatID := c.Param("chatID")
	if chatID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chat ID tidak ditemukan"})
		return
	}

	// Ambil user info dari context (hasil JWTAuthMiddleware)
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID tidak ditemukan dalam konteks"})
		return
	}

	usernameVal, exists := c.Get("username")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Username tidak ditemukan dalam konteks"})
		return
	}

	// Casting sesuai tipe sebenarnya
	userID, ok := userIDVal.(int)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User ID pengguna tidak valid"})
		return
	}

	username, ok := usernameVal.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Tipe username tidak valid"})
		return
	}

	// Proses chatbot
	response, err := services.ProcessChatbot(chatID, req.Message, userID, username)
	if err != nil {
		config.Log.Error("Kesalahan saat memproses chatbot:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

func GetChatByID(c *gin.Context) {
	chatID := c.Param("chatID")
	userID := c.MustGet("userID").(int)

	chat, err := services.GetChatByID(chatID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if chat == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chat tidak ditemukan"})
		return
	}

	c.JSON(http.StatusOK, chat)
}

func GetUserChats(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID tidak ditemukan dalam konteks"})
		return
	}

	chats, err := services.FetchUserChatList(userID.(int))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data chat"})
		return
	}

	// Pastikan selalu return array, meskipun kosong
	if chats == nil {
		chats = []services.ChatListItem{}
	}

	c.JSON(http.StatusOK, gin.H{"data": chats})
}

func GetFullChatHistory(c *gin.Context) {
	chatID := c.Param("chatID")
	userID := c.MustGet("userID").(int)

	messages, err := services.GetFullChatHistory(chatID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil riwayat obrolan lengkap"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"messages": messages})
}

func UpdateLastChatIDHandler(c *gin.Context) {
	var req struct {
		UserID int    `json:"user_id"`
		ChatID string `json:"chat_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Permintaan tidak valid"})
		return
	}

	err := services.UpdateLastChatID(req.UserID, req.ChatID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal update last_chat_id"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "last_chat_id berhasil diupdate"})
}

func RenameChatHandler(c *gin.Context) {
	chatID := c.Param("chatID")
	userID := c.MustGet("userID").(int)

	var req struct {
		NewTitle string `json:"new_title"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.NewTitle == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "new_title tidak valid"})
		return
	}

	err := services.RenameChatTitle(chatID, userID, req.NewTitle)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Chat telah berhasil diganti namanya"})
}

func DeleteChatHandler(c *gin.Context) {
	chatID := c.Param("chatID")
	userID := c.MustGet("userID").(int)

	err := services.DeleteChat(chatID, userID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Obrolan berhasil dihapus"})
}
