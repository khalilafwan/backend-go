package routes

import (
	"backend-go/controllers"
	"backend-go/middleware"

	"github.com/gin-gonic/gin"
)

// SetupRoutes mengatur semua rute utama aplikasi
func SetupRoutes(r *gin.Engine) {

	// Rute untuk chatbot
	r.POST("/chatbot", controllers.ChatbotHandler)

	// Rute untuk autentikasi dan manajemen user
	authGroup := r.Group("/auth")
	{
		authGroup.POST("/login", controllers.Login)                                           // Login user
		authGroup.POST("/logout", middleware.JWTAuthMiddleware(), controllers.Logout)         // Logout user
		authGroup.POST("/register", controllers.CreateUser)                                   // Registrasi atau buat user baru
		authGroup.POST("/chatbot", controllers.ChatbotHandler)                                // Chatbot dapat diakses oleh semua user yang terautentikasi
		authGroup.GET("/current", middleware.JWTAuthMiddleware(), controllers.GetCurrentUser) // Untuk Ambil Data user yang sedang login
		authGroup.POST("/last-chat", controllers.UpdateLastChatIDHandler)
	}

	// Rute untuk fitur teks chatbot
	chatGroup := r.Group("/chat")
	chatGroup.Use(middleware.JWTAuthMiddleware())
	{
		chatGroup.POST("/:chatID", controllers.ChatbotHandler)
		chatGroup.GET("/:chatID/full", controllers.GetFullChatHistory)
		chatGroup.GET("/:chatID", controllers.GetChatByID)
		chatGroup.GET("/list", controllers.GetUserChats)
		chatGroup.PUT("/:chatID", controllers.RenameChatHandler)
		chatGroup.DELETE("/:chatID", controllers.DeleteChatHandler)
	}

	// Rute untuk fitur voice message (speech-to-text dan text-to-speech)
	voiceGroup := r.Group("/voice")
	voiceGroup.Use(middleware.JWTAuthMiddleware())
	{
		voiceGroup.POST("/upload", controllers.UploadVoiceHandler)     // Upload audio dan transkripsi
		voiceGroup.GET("/:chatID", controllers.GetVoiceMessagesByID)   // Ambil voice messages per chat
		voiceGroup.GET("/audio/:filename", controllers.ServeAudioFile) // Serve audio TTS dari S3 atau local
	}

	admin := r.Group("/admin", middleware.JWTAuthMiddleware(), controllers.AdminOnly())
	{
		admin.GET("/metrics", controllers.GetAdminMetricsHandler)
		admin.GET("/conversations", controllers.GetRecentConversationsHandler)
	}

}
