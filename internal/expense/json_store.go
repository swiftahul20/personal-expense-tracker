package expense

import (
	"encoding/json"
	"fmt"
	"os"
)

type JSONStore struct {
	filePath string
}

func NewJSONStore(filePath string) *JSONStore {
	return &JSONStore{filePath: filePath}
}

func (s *JSONStore) load() ([]Expense, error) {
	data, err := os.ReadFile(s.filePath)
	if os.IsNotExist(err) {
		return []Expense{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading store file: %w", err)
	}

	var expenses []Expense
	if len(data) == 0 {
		return expenses, nil
	}
	if err := json.Unmarshal(data, &expenses); err != nil {
		return nil, fmt.Errorf("parsing store file: %w", err)
	}
	return expenses, nil
}

func (s *JSONStore) save(expenses []Expense) error {
	data, err := json.MarshalIndent(expenses, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding expenses: %w", err)
	}
	if err := os.WriteFile(s.filePath, data, 0644); err != nil {
		return fmt.Errorf("writing store file: %w", err)
	}
	return nil
}

func (s *JSONStore) Add(e Expense) (Expense, error) {
	if err := e.Validate(); err != nil {
		return Expense{}, err
	}

	expenses, err := s.load()
	if err != nil {
		return Expense{}, err
	}

	e.ID = nextID(expenses)
	expenses = append(expenses, e)

	if err := s.save(expenses); err != nil {
		return Expense{}, err
	}
	return e, nil
}

func (s *JSONStore) List() ([]Expense, error) {
	return s.load()
}

func (s *JSONStore) Delete(id int) error {
	expenses, err := s.load()
	if err != nil {
		return err
	}

	filtered := make([]Expense, 0, len(expenses))
	found := false
	for _, e := range expenses {
		if e.ID == id {
			found = true
			continue
		}
		filtered = append(filtered, e)
	}

	if !found {
		return fmt.Errorf("expense with id %d not found", id)
	}

	return s.save(filtered)
}

func nextID(expenses []Expense) int {
	maxID := 0
	for _, e := range expenses {
		if e.ID > maxID {
			maxID = e.ID
		}
	}
	return maxID + 1
}

func (s *JSONStore) Update(id int, updates ExpenseUpdate) (Expense, error) {
	expenses, err := s.load()
	if err != nil {
		return Expense{}, err
	}

	index := -1
	for i, e := range expenses {
		if e.ID == id {
			index = i
			break
		}
	}
	if index == -1 {
		return Expense{}, fmt.Errorf("expense with id %d not found", id)
	}

	updated := expenses[index]
	if updates.Amount != nil {
		updated.Amount = *updates.Amount
	}
	if updates.Category != nil {
		updated.Category = *updates.Category
	}
	if updates.Description != nil {
		updated.Description = *updates.Description
	}

	if err := updated.Validate(); err != nil {
		return Expense{}, err
	}

	expenses[index] = updated
	if err := s.save(expenses); err != nil {
		return Expense{}, err
	}

	return updated, nil
}

func (s *JSONStore) GetByID(id int) (Expense, error) {
	expenses, err := s.load()
	if err != nil {
		return Expense{}, err
	}
	for _, e := range expenses {
		if e.ID == id {
			return e, nil
		}
	}
	return Expense{}, fmt.Errorf("expense with id %d not found", id)
}
