package auth

import (
	"time"

	"github.com/swiftahul20/expense-tracker/internal/user"
)

type Handler struct {
	users      user.Store
	jwtManager *JWTManager
	refreshTTL time.Duration
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	User         user.User `json:"user"`
}
