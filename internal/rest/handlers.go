package rest

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/swiftahul20/expense-tracker/internal/auth"
	"github.com/swiftahul20/expense-tracker/internal/expense"
	"github.com/swiftahul20/expense-tracker/internal/report"
)

// types
type Handler struct {
	store expense.Store
}

type paginatedExpensesResponse struct {
	Expenses   []expense.Expense `json:"expenses"`
	Page       int               `json:"page"`
	Limit      int               `json:"limit"`
	Total      int               `json:"total"`
	TotalPages int               `json:"total_pages"`
}

type dashboardResponse struct {
	Expenses   []expense.Expense      `json:"expenses"`
	ByCategory []report.CategoryTotal `json:"by_category"`
	ByMonth    []report.MonthTotal    `json:"by_month"`
	ByDay      []report.DayTotal      `json:"by_day"`
}

// =================================

func (h *Handler) ListExpenses(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	result, err := h.store.List(userID, expense.ListParams{Page: page, Limit: limit})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	expenses := result.Expenses
	if category := r.URL.Query().Get("category"); category != "" {
		filtered := make([]expense.Expense, 0, len(expenses))
		for _, e := range expenses {
			if e.Category == category {
				filtered = append(filtered, e)
			}
		}
		expenses = filtered
	}

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	totalPages := (result.Total + limit - 1) / limit

	writeJSON(w, http.StatusOK, paginatedExpensesResponse{
		Expenses:   expenses,
		Page:       page,
		Limit:      limit,
		Total:      result.Total,
		TotalPages: totalPages,
	})
}

func NewHandler(store expense.Store) *Handler {
	return &Handler{store: store}
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func (h *Handler) GetExpense(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())

	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	e, err := h.store.GetByID(userID, id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, e)
}

func (h *Handler) CreateExpense(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())

	var e expense.Expense
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	created, err := h.store.Add(userID, e)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, created)
}

func (h *Handler) UpdateExpense(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())

	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var payload struct {
		Amount      *float64 `json:"amount"`
		Category    *string  `json:"category"`
		Description *string  `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	updates := expense.ExpenseUpdate{
		Amount:      payload.Amount,
		Category:    payload.Category,
		Description: payload.Description,
	}

	updated, err := h.store.Update(userID, id, updates)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

func (h *Handler) DeleteExpense(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())

	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	if err := h.store.Delete(userID, id); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) SummaryByCategory(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	expenses, err := h.store.ListAll(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, report.ByCategory(expenses))
}

func (h *Handler) SummaryByDay(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	expenses, err := h.store.ListAll(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, report.ByDay(expenses))
}

func (h *Handler) SummaryByMonth(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	expenses, err := h.store.ListAll(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, report.ByMonth(expenses))
}

func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())

	expenses, err := h.store.ListAll(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, dashboardResponse{
		Expenses:   expenses,
		ByCategory: report.ByCategory(expenses),
		ByMonth:    report.ByMonth(expenses),
		ByDay:      report.ByDay(expenses),
	})
}
