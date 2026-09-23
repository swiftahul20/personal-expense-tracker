package expense

import (
	"errors"
	"strings"
	"time"
)

// types
type Expense struct {
	ID          int       `json:"id"`
	Amount      float64   `json:"amount"`
	Category    string    `json:"category"`
	SubCategory string    `json:"sub_category,omitempty"`
	Description string    `json:"description"`
	Date        time.Time `json:"date"`
}

// =======================================

var (
	ErrInvalidAmount = errors.New("amount must be greater than 0")
	ErrEmptyCategory = errors.New("category is required")
	ErrFutureDate    = errors.New("date cannot be in the future")
)

func (e *Expense) Validate() error {
	if e.Amount <= 0 {
		return ErrInvalidAmount
	}

	e.Category = strings.TrimSpace(e.Category)
	if e.Category == "" {
		return ErrEmptyCategory
	}

	e.Description = strings.TrimSpace(e.Description)

	if e.Date.After(time.Now().Add(24 * time.Hour)) {
		return ErrFutureDate
	}

	return nil
}
