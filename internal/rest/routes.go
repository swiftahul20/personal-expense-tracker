package rest

import (
	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/swiftahul20/expense-tracker/internal/auth"
	"github.com/swiftahul20/expense-tracker/internal/ratelimit"

	httpSwagger "github.com/swaggo/http-swagger"

	_ "github.com/swiftahul20/expense-tracker/docs"
)

func NewRouter(h *Handler, authHandler *auth.Handler, jwtManager *auth.JWTManager, loginLimiter *ratelimit.Limiter, healthHandler *HealthHandler, log *slog.Logger) *chi.Mux {
	r := chi.NewRouter()
	r.Use(StructuredLogger(log))
	r.Get("/health", healthHandler.Check)
	r.Get("/swagger/*", httpSwagger.WrapHandler)

	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", authHandler.Register)
		r.With(loginLimiter.Middleware).Post("/login", authHandler.Login)
		r.Post("/refresh", authHandler.Refresh)
		r.Post("/logout", authHandler.Logout)
		r.With(jwtManager.Middleware).Get("/me", authHandler.Me)
	})

	r.Group(func(r chi.Router) {
		r.Use(jwtManager.Middleware)

		r.Route("/expenses", func(r chi.Router) {
			r.Get("/", h.ListExpenses)
			r.Post("/", h.CreateExpense)
			r.Get("/{id}", h.GetExpense)
			r.Put("/{id}", h.UpdateExpense)
			r.Delete("/{id}", h.DeleteExpense)
		})

		r.Route("/summary", func(r chi.Router) {
			r.Get("/category", h.SummaryByCategory)
			r.Get("/day", h.SummaryByDay)
			r.Get("/month", h.SummaryByMonth)
		})

		r.Get("/dashboard", h.Dashboard)
	})

	return r
}
