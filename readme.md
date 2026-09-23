# Expense Tracker API

A REST API for tracking personal expenses, built in Go while learning the language. Started as a CLI tool, evolved into a full REST API backed by PostgreSQL with JWT authentication, refresh token rotation, and per-user data isolation.

## Features

- **Full CRUD** for expenses (amount, category, sub-category, description, date)
- **Pagination** on the expense list (page/limit, capped at 100 per page, total count included)
- **Category, monthly, and daily summaries** — each includes both totals and the underlying list of expenses for that group
- **Combined dashboard endpoint** — expenses + all three summaries in a single response
- **JWT authentication** — short-lived access tokens + long-lived refresh tokens, rotated on every use
- **Logout** — revokes a refresh token server-side
- **Current user endpoint** — fetch the authenticated user's own profile
- **Per-user data isolation** — enforced at the database query level, not just in application logic
- **Rate limiting** on login attempts (per-IP, in-memory)
- **Input validation** on all writes (amount > 0, required category, no future dates)
- **Dockerized** — the API and PostgreSQL both run via a single `docker compose up`

## Tech Stack

- **Language:** Go
- **Router:** [chi](https://github.com/go-chi/chi)
- **Database:** PostgreSQL, via [pgx](https://github.com/jackc/pgx)
- **Auth:** [golang-jwt](https://github.com/golang-jwt/jwt) + bcrypt password hashing
- **Config:** environment variables via `.env` ([godotenv](https://github.com/joho/godotenv))
- **Containerization:** Docker, Docker Compose (multi-stage build)

## Architecture

The project follows a layered structure with domain logic decoupled from storage and transport:

```
cmd/
└── rest-server/       # entry point, wires config, DB pool, handlers, routes
internal/
├── expense/           # domain model, Store interface, Postgres implementation, validation
├── user/              # user domain, Store interface, Postgres implementation
├── auth/              # password hashing, JWT issuing/verification, refresh tokens, middleware
├── ratelimit/         # in-memory per-IP rate limiter
├── report/            # types (types.go) + pure aggregation logic (summary.go) — by category/month/day
├── rest/              # HTTP handlers and route wiring
└── config/            # environment-based configuration loading
```

**Key design decision:** both `expense.Store` and `user.Store` are interfaces, with PostgreSQL as the only current implementation. Handlers depend only on these interfaces, never on Postgres directly — this keeps storage swappable and testable without touching business logic.

## Getting Started

### Prerequisites

- Docker and Docker Compose

### Setup

1. Clone the repo and create a `.env` file in the project root:

   ```
   DATABASE_URL=postgres://expense_user:expense_pass@postgres:5432/expense_tracker
   JWT_SECRET=<a long random string>
   JWT_TTL_HOURS=1
   REFRESH_TTL_DAYS=7
   ```

2. Start everything:

   ```bash
   docker compose up --build
   ```

   This builds the Go server image and starts both the API and PostgreSQL, waiting for the database to be healthy before the API starts.

3. Create the schema (first run only):

   ```bash
   docker exec -it expense-postgres psql -U expense_user -d expense_tracker
   ```

   ```sql
   CREATE TABLE users (
       id SERIAL PRIMARY KEY,
       email VARCHAR(255) UNIQUE NOT NULL,
       password_hash VARCHAR(255) NOT NULL,
       created_at TIMESTAMPTZ NOT NULL DEFAULT now()
   );

   CREATE TABLE expenses (
       id SERIAL PRIMARY KEY,
       user_id INTEGER REFERENCES users(id),
       amount NUMERIC(12, 2) NOT NULL,
       category VARCHAR(100) NOT NULL,
       sub_category VARCHAR(100) NOT NULL DEFAULT '',
       description TEXT,
       date TIMESTAMPTZ NOT NULL
   );

   CREATE TABLE refresh_tokens (
       id SERIAL PRIMARY KEY,
       user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
       token_hash VARCHAR(255) NOT NULL UNIQUE,
       expires_at TIMESTAMPTZ NOT NULL,
       created_at TIMESTAMPTZ NOT NULL DEFAULT now()
   );
   ```

The API is now available at `http://localhost:8080`.

### Stopping

```bash
docker compose down       # stop containers, keep data
docker compose down -v    # stop containers and wipe the database volume
```

## API Reference

### Auth

| Method | Endpoint         | Auth required | Description                                                               |
| ------ | ---------------- | :-----------: | ------------------------------------------------------------------------- |
| POST   | `/auth/register` |      No       | Create an account                                                         |
| POST   | `/auth/login`    |      No       | Log in, receive access + refresh tokens (rate-limited)                    |
| POST   | `/auth/refresh`  |      No       | Exchange a refresh token for a new token pair (rotates the refresh token) |
| POST   | `/auth/logout`   |      No       | Revoke a refresh token                                                    |
| GET    | `/auth/me`       |      Yes      | Get the authenticated user's profile                                      |

### Expenses

All routes below require `Authorization: Bearer <access_token>`.

| Method | Endpoint         | Description                                                                           |
| ------ | ---------------- | ------------------------------------------------------------------------------------- |
| GET    | `/expenses`      | List the authenticated user's expenses, paginated (`?page=`, `?limit=`, `?category=`) |
| POST   | `/expenses`      | Create an expense                                                                     |
| GET    | `/expenses/{id}` | Get a single expense                                                                  |
| PUT    | `/expenses/{id}` | Partially update an expense                                                           |
| DELETE | `/expenses/{id}` | Delete an expense                                                                     |

`GET /expenses` response shape:

```json
{
  "expenses": [ ... ],
  "page": 1,
  "limit": 20,
  "total": 47,
  "total_pages": 3
}
```

### Summaries

| Method | Endpoint            | Description                                                 |
| ------ | ------------------- | ----------------------------------------------------------- |
| GET    | `/summary/category` | Totals grouped by category, including each group's expenses |
| GET    | `/summary/month`    | Totals grouped by month, including each group's expenses    |
| GET    | `/summary/day`      | Totals grouped by day, including each group's expenses      |
| GET    | `/dashboard`        | Expenses + all three summaries combined in one response     |

## Example: Register → Login → Create Expense

```bash
# Register
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email": "you@example.com", "password": "password123"}'

# Response includes access_token and refresh_token

# Create an expense
curl -X POST http://localhost:8080/expenses \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <access_token>" \
  -d '{"amount": 50000, "category": "food", "sub_category": "dine-in", "description": "Lunch", "date": "2026-09-21T00:00:00Z"}'
```

## Notes

This project started as a CLI tool for learning Go fundamentals (structs, interfaces, file I/O) before evolving into this REST API. A GraphQL API was originally planned alongside REST for comparison purposes but was dropped in favor of focusing on backend fundamentals (auth, security, containerization).
