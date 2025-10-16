package services

import (
	"backend-go/config"
	"backend-go/models"
	"errors"

	"golang.org/x/crypto/bcrypt"
)

// // CreateUser creates a new user and hashes the password, setting initial status to offline
// func CreatedUser(username, password, role, status string) error {
// 	// Hash password using bcrypt
// 	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
// 	if err != nil {
// 		config.Log.Error("Error hashing password: ", err)
// 		return errors.New("failed to hash password")
// 	}

// 	// Insert the new user into the database with default status "offline"
// 	query := `INSERT INTO "users" (username, password, role, status) VALUES ($1, $2, $3, $4)`
// 	_, err = config.DB.Exec(query, username, string(hashedPassword), role, status)
// 	if err != nil {
// 		config.Log.Error("Error creating user: ", err)
// 		return err
// 	}

// 	return nil
// }

// CreatedUser creates a new user, hashes the password, and returns the user
func CreatedUser(username, password, role, status string) (models.User, error) {
	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		config.Log.Error("Error hashing password: ", err)
		return models.User{}, errors.New("failed to hash password")
	}

	// Insert new user and RETURNING id
	var createdUser models.User
	query := `INSERT INTO "users" (username, password, role, status)
			  VALUES ($1, $2, $3, $4)
			  RETURNING id, username, role, status`

	err = config.DB.QueryRow(query, username, string(hashedPassword), role, status).
		Scan(&createdUser.ID, &createdUser.Username, &createdUser.Role, &createdUser.Status)

	if err != nil {
		config.Log.Error("Error creating user: ", err)
		return models.User{}, err
	}

	return createdUser, nil
}
