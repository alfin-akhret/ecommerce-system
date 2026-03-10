# ecommerce-system

Backend e-commerce (work in progress) written in Go. The repo currently contains a small REST API with a clean-ish layering style (`handler -> service -> repository -> database`) backed by Postgres. Redis is scaffolded but not wired yet.

## What Exists Today

- API server (Go + `chi`) with:
  - `GET /health`
  - `POST /users` (create user; password is stored as `bcrypt` hash)
- Postgres schema migrations for:
  - `users`, `products`, `product_inventory`, `orders`, `order_items`, `payments`
- Local infrastructure via Docker Compose: Postgres + Redis

## Architecture (Code Map)

The request path is:

`HTTP -> handler -> service -> repository -> Postgres`

Key locations:

- `api/cmd/server/main.go`: HTTP server bootstrap + routes
- `api/internal/app/app.go`: dependency wiring (config, DB, handlers)
- `api/internal/handler/`: HTTP layer (JSON decode/encode)
- `api/internal/service/`: business logic (e.g. password hashing)
- `api/internal/repository/`: database queries
- `api/internal/database/`: DB clients (`pgxpool`, Redis client)
- `api/internal/model/`: domain models
- `api/migrations/`: SQL schema migrations (currently manual/external-tool driven)
- `infrastructure/docker/docker-compose.yml`: local Postgres + Redis

## API

### Health Check

- `GET /health`
- Response: `200 OK` with body `OK cool`

### Create User

- `POST /users`
- Content-Type: `application/json`
- Body:

```json
{
  "name": "NewUser",
  "email": "newuser@gmail.com",
  "password": "ChangeMe123!"
}
```

- Response: `201 Created`

This endpoint is implemented in:
- `api/internal/handler/user_handler.go` (`CreateUser`)
- `api/internal/service/user_service.go` (`CreateUser`)
- `api/internal/repository/user_repository.go` (`Create`)

You can also use the scratch file `api/api_test.http` to try the endpoints from an IDE HTTP client.

## Configuration

The API reads env vars (with defaults) in `api/internal/config/config.go`:

- `PORT` (default `8080`)
- `DATABASE_URL` (default `postgres://postgres:postgres@localhost:5432/ecommerce`)
- `REDIS_ADDR` (default `localhost:6379`) (not used yet)

## Local Development

### 1) Start Postgres + Redis

From repo root:

```bash
docker compose -f infrastructure/docker/docker-compose.yml up -d
```

### 2) Apply DB schema

This repo includes plain SQL migration files under `api/migrations/`.
You can apply them with `psql` (or any migration tool you prefer).

Example using Docker + `psql`:

```bash
docker exec -i ecommerce-postgres psql -U postgres -d ecommerce < api/migrations/000001_init_schema.up.sql
```

To drop tables:

```bash
docker exec -i ecommerce-postgres psql -U postgres -d ecommerce < api/migrations/000001_init_schema.down.sql
```

### 3) Run the API server

```bash
cd api
go run ./cmd/server
```

Server prints the port and listens on `http://localhost:8080` by default.

## Notes / Next Work

- The DB schema already includes products, inventory, orders, and payments, but the Go implementation currently only exposes user creation.
- `github.com/gin-gonic/gin` is present in `api/go.mod`, but the running server uses `chi` right now.
