package controllers

import (
	"backend-go/models"
	"backend-go/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Login handles user authentication and returns JWT token
func Login(c *gin.Context) {
	var credentials models.Credentials
	if err := c.ShouldBindJSON(&credentials); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Kredensial tidak valid"})
		return
	}

	user, err := services.Authenticate(credentials.Username, credentials.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Autentikasi gagal"})
		return
	}

	// 1. Update status ke online
	if err := services.SetUserOnline(user.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui status pengguna"})
		return
	}

	user.Status = "online"

	// 2. Generate JWT token
	token, err := services.GenerateJWT(user.ID, user.Username, user.Role, user.Status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghasilkan token"})
		return
	}

	// 3. Return response with token
	c.JSON(http.StatusOK, gin.H{
		"message":      "Login successful",
		"id":           user.ID,
		"token":        token,
		"username":     user.Username,
		"role":         user.Role,
		"status":       user.Status,
		"last_chat_id": user.LastChatID,
	})
}

// GetCurrentUser returns the currently logged-in user's info from JWT
func GetCurrentUser(c *gin.Context) {
	claims, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userClaims := claims.(*models.Claims)

	c.JSON(http.StatusOK, gin.H{
		"id":       userClaims.ID, // ✅ Sekarang ini tidak null
		"username": userClaims.Username,
		"role":     userClaims.Role,
		"status":   userClaims.Status,
	})
}

// Logout handles user logout (token-based logout typically handled on client)
func Logout(c *gin.Context) {
	claims, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User belum log in"})
		return
	}

	userClaims := claims.(*models.Claims)

	// Ubah status user jadi offline di DB
	err := services.SetUserOffline(userClaims.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logout berhasil"})
}
