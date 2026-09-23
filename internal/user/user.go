package user

import (
	"errors"
	"strings"
	"time"
)

type User struct {
	ID           int       `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type Store interface {
	Create(email, passwordHash string) (User, error)
	GetByEmail(email string) (User, error)
	GetByID(id int) (User, error)
	SaveRefreshToken(userID int, tokenHash string, expiresAt time.Time) error
	GetUserIDByRefreshToken(tokenHash string) (int, error)
	DeleteRefreshToken(tokenHash string) error
}

var (
	ErrEmptyEmail             = errors.New("email is required")
	ErrWeakPassword           = errors.New("password must be at least 8 characters")
	ErrEmailTaken             = errors.New("email is already registered")
	ErrUserNotFound           = errors.New("user not found")
	ErrInvalidLogin           = errors.New("invalid email or password")
	ErrInvalidRefreshTokenRow = errors.New("invalid or expired refresh token")
)

func ValidateRegistration(email, password string) error {
	if strings.TrimSpace(email) == "" {
		return ErrEmptyEmail
	}
	if len(password) < 8 {
		return ErrWeakPassword
	}
	return nil
}
