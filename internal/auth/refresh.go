package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

var ErrInvalidRefreshToken = errors.New("invalid or expired refresh token")

func GenerateRefreshToken() (plain string, hash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}

	plain = hex.EncodeToString(b)
	hash = HashToken(plain)
	return plain, hash, nil
}

func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	hash := HashToken(req.RefreshToken)
	userID, err := h.users.GetUserIDByRefreshToken(hash)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid or expired refresh token")
		return
	}

	// Rotate: delete the old refresh token, issue a new one
	_ = h.users.DeleteRefreshToken(hash)

	newAccessToken, err := h.jwtManager.Generate(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	newPlainRefresh, newHashedRefresh, err := GenerateRefreshToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate refresh token")
		return
	}

	if err := h.users.SaveRefreshToken(userID, newHashedRefresh, time.Now().Add(h.refreshTTL)); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save refresh token")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"access_token":  newAccessToken,
		"refresh_token": newPlainRefresh,
	})
}
