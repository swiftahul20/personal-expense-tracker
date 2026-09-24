package expense

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// types
type ListParams struct {
	Page     int
	Limit    int
	Category string
	Search   string
	From     *time.Time
	To       *time.Time
}

type ListResult struct {
	Expenses []Expense
	Total    int
}

type Store interface {
	Add(userID int, e Expense) (Expense, error)
	List(userID int, params ListParams) (ListResult, error)
	ListAll(userID int) ([]Expense, error)
	ExportAll(userID int, params ListParams) ([]Expense, error)
	GetByID(userID, id int) (Expense, error)
	Delete(userID, id int) error
	Update(userID, id int, updates ExpenseUpdate) (Expense, error)
}

type ExpenseUpdate struct {
	Amount      *float64
	Category    *string
	Description *string
}

type PostgresStore struct {
	pool *pgxpool.Pool
}

// =================================

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) Add(userID int, e Expense) (Expense, error) {
	if err := e.Validate(); err != nil {
		return Expense{}, err
	}

	ctx := context.Background()
	query := `INSERT INTO expenses (user_id, amount, category, description, date)
	          VALUES ($1, $2, $3, $4, $5) RETURNING id`

	err := s.pool.QueryRow(ctx, query, userID, e.Amount, e.Category, e.Description, e.Date).Scan(&e.ID)
	if err != nil {
		return Expense{}, fmt.Errorf("inserting expense: %w", err)
	}
	return e, nil
}

func (s *PostgresStore) List(userID int, params ListParams) (ListResult, error) {
	ctx := context.Background()

	if params.Page < 1 {
		params.Page = 1
	}
	if params.Limit < 1 || params.Limit > 100 {
		params.Limit = 20
	}
	offset := (params.Page - 1) * params.Limit

	where, args, argN := buildFilterQuery(userID, params)

	var total int
	countQuery := "SELECT COUNT(*) FROM expenses " + where
	if err := s.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return ListResult{}, fmt.Errorf("counting expenses: %w", err)
	}

	query := fmt.Sprintf(
		"SELECT id, amount, category, sub_category, description, date FROM expenses %s ORDER BY date DESC LIMIT $%d OFFSET $%d",
		where, argN, argN+1,
	)
	args = append(args, params.Limit, offset)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return ListResult{}, fmt.Errorf("querying expenses: %w", err)
	}
	defer rows.Close()

	var expenses []Expense
	for rows.Next() {
		var e Expense
		if err := rows.Scan(&e.ID, &e.Amount, &e.Category, &e.SubCategory, &e.Description, &e.Date); err != nil {
			return ListResult{}, fmt.Errorf("scanning expense row: %w", err)
		}
		expenses = append(expenses, e)
	}
	if err := rows.Err(); err != nil {
		return ListResult{}, err
	}

	return ListResult{Expenses: expenses, Total: total}, nil
}

func (s *PostgresStore) ListAll(userID int) ([]Expense, error) {
	ctx := context.Background()
	query := `SELECT id, amount, category, sub_category, description, date
	          FROM expenses WHERE user_id = $1 ORDER BY date`

	rows, err := s.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("querying expenses: %w", err)
	}
	defer rows.Close()

	var expenses []Expense
	for rows.Next() {
		var e Expense
		if err := rows.Scan(&e.ID, &e.Amount, &e.Category, &e.SubCategory, &e.Description, &e.Date); err != nil {
			return nil, fmt.Errorf("scanning expense row: %w", err)
		}
		expenses = append(expenses, e)
	}
	return expenses, rows.Err()
}

func (s *PostgresStore) Delete(userID int, id int) error {
	ctx := context.Background()
	query := `DELETE FROM expenses WHERE user_id = $1 AND id = $2`

	tag, err := s.pool.Exec(ctx, query, userID, id)
	if err != nil {
		return fmt.Errorf("deleting expense: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("expense with id %d not found", id)
	}
	return nil
}

func (s *PostgresStore) Update(userID int, id int, updates ExpenseUpdate) (Expense, error) {
	ctx := context.Background()

	var current Expense
	err := s.pool.QueryRow(ctx,
		`SELECT id, amount, category, description, date FROM expenses WHERE user_id = $1 AND id = $2`, userID, id,
	).Scan(&current.ID, &current.Amount, &current.Category, &current.Description, &current.Date)
	if err != nil {
		return Expense{}, fmt.Errorf("expense with id %d not found", id)
	}

	if updates.Amount != nil {
		current.Amount = *updates.Amount
	}
	if updates.Category != nil {
		current.Category = *updates.Category
	}
	if updates.Description != nil {
		current.Description = *updates.Description
	}

	if err := current.Validate(); err != nil {
		return Expense{}, err
	}

	_, err = s.pool.Exec(ctx,
		`UPDATE expenses SET amount = $1, category = $2, description = $3 WHERE user_id = $4 AND id = $5`,
		current.Amount, current.Category, current.Description, userID, id,
	)
	if err != nil {
		return Expense{}, fmt.Errorf("updating expense: %w", err)
	}

	return current, nil
}

func (s *PostgresStore) GetByID(userID int, id int) (Expense, error) {
	ctx := context.Background()
	var e Expense

	err := s.pool.QueryRow(ctx,
		`SELECT id, amount, category, description, date FROM expenses WHERE user_id = $1 AND id = $2`, userID, id,
	).Scan(&e.ID, &e.Amount, &e.Category, &e.Description, &e.Date)
	if err != nil {
		return Expense{}, fmt.Errorf("expense with id %d not found", id)
	}

	return e, nil
}

func buildFilterQuery(userID int, params ListParams) (where string, args []interface{}, nextArgN int) {
	where = "WHERE user_id = $1"
	args = []interface{}{userID}
	argN := 2

	if params.Category != "" {
		where += fmt.Sprintf(" AND category = $%d", argN)
		args = append(args, params.Category)
		argN++
	}
	if params.Search != "" {
		where += fmt.Sprintf(" AND description ILIKE $%d", argN)
		args = append(args, "%"+params.Search+"%")
		argN++
	}
	if params.From != nil {
		where += fmt.Sprintf(" AND date >= $%d", argN)
		args = append(args, *params.From)
		argN++
	}
	if params.To != nil {
		where += fmt.Sprintf(" AND date <= $%d", argN)
		args = append(args, *params.To)
		argN++
	}

	return where, args, argN
}

func (s *PostgresStore) ExportAll(userID int, params ListParams) ([]Expense, error) {
	ctx := context.Background()

	where, args, _ := buildFilterQuery(userID, params)

	query := "SELECT id, amount, category, sub_category, description, date FROM expenses " + where + " ORDER BY date DESC"

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying expenses: %w", err)
	}
	defer rows.Close()

	var expenses []Expense
	for rows.Next() {
		var e Expense
		if err := rows.Scan(&e.ID, &e.Amount, &e.Category, &e.SubCategory, &e.Description, &e.Date); err != nil {
			return nil, fmt.Errorf("scanning expense row: %w", err)
		}
		expenses = append(expenses, e)
	}
	return expenses, rows.Err()
}
