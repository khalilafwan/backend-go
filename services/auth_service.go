package services

import (
	"backend-go/config"
	"backend-go/models"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var jwtSecretKey = []byte("my-secret-key") // Ganti dengan secret yang aman

// GenerateJWT generates a new JWT token for the given user
func GenerateJWT(ID int, username, role, status string) (string, error) {
	// Atur waktu kedaluwarsa token (misal 24 jam)
	expirationTime := time.Now().Add(24 * time.Hour)

	// Buat klaim
	claims := models.Claims{
		ID:       ID,
		Username: username,
		Role:     role,
		Status:   status,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	// Buat token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Tanda tangani dengan secret key
	signedToken, err := token.SignedString(jwtSecretKey)
	if err != nil {
		return "", err
	}

	return signedToken, nil
}

func ValidateJWT(tokenString string) (*models.Claims, error) {
	secret := []byte("my-secret-key")
	token, err := jwt.ParseWithClaims(tokenString, &models.Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*models.Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

func Authenticate(username, password string) (*models.User, error) {
	var user models.User

	query := `SELECT id, username, password, role, last_chat_id FROM users WHERE username = $1`
	err := config.DB.QueryRow(query, username).Scan(&user.ID, &user.Username, &user.Password, &user.Role, &user.LastChatID)
	if err == sql.ErrNoRows {
		return nil, errors.New("user tidak ditemukan")
	} else if err != nil {
		return nil, fmt.Errorf("kesalahan saat mengambil data pengguna: %v", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, errors.New("password tidak valid")
	}

	return &user, nil
}

func SetUserOnline(userID int) error {
	query := `UPDATE "users" SET status = 'online' WHERE id = $1`
	_, err := config.DB.Exec(query, userID)
	if err != nil {
		config.Log.Error("Gagal mengatur status online pengguna:", err)
		return err
	}

	config.Log.Info("User ID", userID, "set to online")
	return nil
}

// Logout user by setting status to "offline"
func Logout(userID int) error {
	return UpdateUserStatus(userID, "offline")
}

func SetUserOffline(userID int) error {
	query := `UPDATE "users" SET status = 'offline' WHERE id = $1`
	_, err := config.DB.Exec(query, userID)
	if err != nil {
		config.Log.Error("Gagal mengatur status offline pengguna:", err)
		return err
	}
	return nil
}

// UpdateUserStatus updates the status of a user in the database
func UpdateUserStatus(userID int, status string) error {
	query := `UPDATE users SET status = ? WHERE id = ?`
	_, err := config.DB.Exec(query, status, userID)
	return err
}
