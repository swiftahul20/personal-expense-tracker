package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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

type HealthHandler struct {
	pool *pgxpool.Pool
}

// =================================

// ListExpenses godoc
// @Description  Returns a paginated list of the authenticated user's expenses
// @Tags         Expenses
// @Produce      json
// @Security     BearerAuth
// @Param        page      query    int    false  "Page number (default 1)"
// @Param        limit     query    int    false  "Items per page (default 20, max 100)"
// @Param        category  query    string false  "Filter by category"
// @Success      200 {object} paginatedExpensesResponse
// @Failure      401 {object} map[string]string
// @Router       /expenses [get]
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

// @Tags         Expenses
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Expense ID"
// @Success      200 {object} expense.Expense
// @Failure      401 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Router       /expenses/{id} [get]
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

// @Description  Creates an expense for the authenticated user
// @Tags         Expenses
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        expense body expense.Expense true "Expense to create"
// @Success      201 {object} expense.Expense
// @Failure      400 {object} map[string]string
// @Router       /expenses [post]
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

// @Description  Partially updates an expense — only provided fields are changed
// @Tags         Expenses
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Expense ID"
// @Param        updates body expense.ExpenseUpdate true "Fields to update"
// @Success      200 {object} expense.Expense
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Router       /expenses/{id} [put]
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

// @Description  Deletes an expense belonging to the authenticated user
// @Tags         Expenses
// @Security     BearerAuth
// @Param        id path int true "Expense ID"
// @Success      204 "No Content"
// @Failure      401 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Router       /expenses/{id} [delete]
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

// @Description  Returns totals grouped by category, including each group's expenses
// @Tags         Summary
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} report.CategoryTotal
// @Failure      401 {object} map[string]string
// @Router       /summary/category [get]
func (h *Handler) SummaryByCategory(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	expenses, err := h.store.ListAll(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, report.ByCategory(expenses))
}

// @Description  Returns totals grouped by day, including each group's expenses
// @Tags         Summary
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} report.DayTotal
// @Failure      401 {object} map[string]string
// @Router       /summary/day [get]
func (h *Handler) SummaryByDay(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	expenses, err := h.store.ListAll(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, report.ByDay(expenses))
}

// @Description  Returns totals grouped by month, including each group's expenses
// @Tags         Summary
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} report.MonthTotal
// @Failure      401 {object} map[string]string
// @Router       /summary/month [get]
func (h *Handler) SummaryByMonth(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	expenses, err := h.store.ListAll(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, report.ByMonth(expenses))
}

// @Description  Returns expenses plus all three summaries combined in a single response
// @Tags         Summary
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} dashboardResponse
// @Failure      401 {object} map[string]string
// @Router       /dashboard [get]
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

// api check
func NewHealthHandler(pool *pgxpool.Pool) *HealthHandler {
	return &HealthHandler{pool: pool}
}

func (h *HealthHandler) Check(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := h.pool.Ping(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status":   "unhealthy",
			"database": "unreachable",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":   "healthy",
		"database": "connected",
	})
}
