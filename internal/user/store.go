package user

import "time"

type RefreshTokenStore interface {
	SaveRefreshToken(userID int, tokenHash string, expiresAt time.Time) error
	GetUserIDByRefreshToken(tokenHash string) (int, error)
	DeleteRefreshToken(tokenHash string) error
}
