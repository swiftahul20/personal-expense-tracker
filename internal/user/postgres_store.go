package user

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) Create(email, passwordHash string) (User, error) {
	ctx := context.Background()
	var u User

	query := `INSERT INTO users (email, password_hash) VALUES ($1, $2)
	          RETURNING id, email, password_hash, created_at`

	err := s.pool.QueryRow(ctx, query, email, passwordHash).
		Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		return User{}, fmt.Errorf("creating user: %w", err)
	}

	return u, nil
}

func (s *PostgresStore) GetByEmail(email string) (User, error) {
	ctx := context.Background()
	var u User

	query := `SELECT id, email, password_hash, created_at FROM users WHERE email = $1`

	err := s.pool.QueryRow(ctx, query, email).
		Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return User{}, ErrUserNotFound
		}
		return User{}, fmt.Errorf("fetching user: %w", err)
	}

	return u, nil
}

// geybyid
func (s *PostgresStore) GetByID(id int) (User, error) {
	ctx := context.Background()
	var u User

	err := s.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, created_at FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return User{}, ErrUserNotFound
		}
		return User{}, fmt.Errorf("fetching user: %w", err)
	}

	return u, nil
}

// / refresh token
func (s *PostgresStore) SaveRefreshToken(userID int, tokenHash string, expiresAt time.Time) error {
	ctx := context.Background()
	_, err := s.pool.Exec(ctx,
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		userID, tokenHash, expiresAt,
	)
	return err
}

func (s *PostgresStore) GetUserIDByRefreshToken(tokenHash string) (int, error) {
	ctx := context.Background()
	var userID int
	var expiresAt time.Time

	err := s.pool.QueryRow(ctx,
		`SELECT user_id, expires_at FROM refresh_tokens WHERE token_hash = $1`, tokenHash,
	).Scan(&userID, &expiresAt)
	if err != nil {
		return 0, ErrInvalidRefreshTokenRow
	}

	if time.Now().After(expiresAt) {
		return 0, ErrInvalidRefreshTokenRow
	}

	return userID, nil
}

func (s *PostgresStore) DeleteRefreshToken(tokenHash string) error {
	ctx := context.Background()
	_, err := s.pool.Exec(ctx, `DELETE FROM refresh_tokens WHERE token_hash = $1`, tokenHash)
	return err
}
