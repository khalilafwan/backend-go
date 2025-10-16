package controllers

import (
	"backend-go/models"
	"backend-go/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

// CreateUser handles adding a new user with a specific role and default status
func CreateUser(c *gin.Context) {
	var newUser models.User
	if err := c.ShouldBindJSON(&newUser); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	// Atur nilai default untuk role dan status jika kosong
	if newUser.Role == "" {
		newUser.Role = "user"
	}
	newUser.Status = "online" // User baru dianggap langsung login

	// Gunakan service yang mengembalikan user
	createdUser, err := services.CreatedUser(newUser.Username, newUser.Password, newUser.Role, newUser.Status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Generate token dari user hasil insert
	token, err := services.GenerateJWT(createdUser.ID, createdUser.Username, createdUser.Role, createdUser.Status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Token generation failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User created successfully",
		"token":   token,
		"user": gin.H{
			"id":       createdUser.ID,
			"username": createdUser.Username,
			"role":     createdUser.Role,
			"status":   createdUser.Status,
		},
	})
}
