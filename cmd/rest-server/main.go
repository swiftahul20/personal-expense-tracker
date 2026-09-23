package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/swiftahul20/expense-tracker/internal/auth"
	"github.com/swiftahul20/expense-tracker/internal/config"
	"github.com/swiftahul20/expense-tracker/internal/expense"
	"github.com/swiftahul20/expense-tracker/internal/ratelimit"
	"github.com/swiftahul20/expense-tracker/internal/rest"
	"github.com/swiftahul20/expense-tracker/internal/user"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to load config:", err)
		os.Exit(1)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to connect to database:", err)
		os.Exit(1)
	}
	if err := pool.Ping(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "failed to ping database:", err)
		os.Exit(1)
	}

	expenseStore := expense.NewPostgresStore(pool)
	userStore := user.NewPostgresStore(pool)
	jwtManager := auth.NewJWTManager(cfg.JWTSecret, cfg.JWTTTL)

	expenseHandler := rest.NewHandler(expenseStore)
	authHandler := auth.NewHandler(userStore, jwtManager, cfg.RefreshTTL)

	loginLimiter := ratelimit.New(5, 15*time.Minute)
	router := rest.NewRouter(expenseHandler, authHandler, jwtManager, loginLimiter)

	log.Println("REST server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}
