package models

// Credentials structure for login request
type Credentials struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// User structure represents the user entity in the system
type User struct {
	ID         int     `json:"id"`
	Username   string  `json:"username"`
	Password   string  `json:"password"`     // Stored as a hash
	Role       string  `json:"role"`         // admin or user
	Status     string  `json:"status"`       // online, offline
	LastChatID *string `json:"last_chat_id"` // ID Chat terakhir
}
