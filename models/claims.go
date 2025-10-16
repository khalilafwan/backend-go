package models

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims defines the structure of the JWT claims
type Claims struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	Status   string `json:"status"`
	jwt.RegisteredClaims
}

// Valid checks if the token is still valid (implements jwt.Claims)
func (c Claims) Valid() error {
	if c.ExpiresAt == nil || c.ExpiresAt.Before(time.Now()) {
		return errors.New("token is expired")
	}
	return nil
}
