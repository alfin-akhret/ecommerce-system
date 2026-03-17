# ecommerce-system

Backend e-commerce (work in progress) written in Go. The repo currently contains a small REST API with a clean-ish layering style (`handler -> service -> repository -> database`) backed by Postgres. Redis is scaffolded but not wired yet.

## What Exists Today

- API server (Go + `chi`) with:
  - `GET /health`
  - `POST /users` (create user; password is stored as `bcrypt` hash)
  - `POST /register` (create user)
  - `POST /login` (JWT login)
  - `GET /me` (protected, requires `Authorization: Bearer <token>`)
  - `GET /users/{id}`
  - `POST /products` (create product + inventory)
  - `GET /products`
  - `GET /products/{id}`
  - `PATCH /products/{id}/stock`
  - `POST /products/{id}/reserve`
  - `POST /products/{id}/release`
  - `POST /products/{id}/confirm`
  - `GET /orders` (protected)
  - `GET /orders/{id}` (protected)
  - `POST /checkout` (protected)
  - `POST /payments` (protected)
  - `GET /payments/{payment_id}` (protected)
  - `PATCH /payments/{payment_id}` (protected)
  - `POST /payments/{payment_id}/success`
  - `POST /payments/{payment_id}/fail`
  - `POST /payments/callback`
- Postgres schema migrations for:
  - `users`, `products`, `product_inventory`, `orders`, `order_items`, `payments`
- Local infrastructure via Docker Compose: Postgres + Redis

## Architecture (Code Map)

The request path is:

`HTTP -> handler -> service -> repository -> Postgres`

Key locations:

- `api/cmd/api/main.go`: HTTP server bootstrap + routes
- `api/internal/app/app.go`: dependency wiring (config, DB, handlers)
- `api/internal/auth/`: auth domain (JWT, middleware, handler)
- `api/internal/user/`: user domain (model, dto, repository, service, handler)
- `api/internal/product/`: product + inventory domain
- `api/internal/order/`: order domain
- `api/pkg/database/`: DB clients (`pgxpool`, Redis client)
- `api/pkg/helper/`: shared response helpers
- `api/migrations/`: SQL schema migrations (currently manual/external-tool driven)
- `infrastructure/docker/docker-compose.yml`: local Postgres + Redis

## API

## Payment Expiration Flow

```
PaymentExpirationWorker Start
    |
    v
Tick Interval
    |
    v
ExpirePayments (set status=EXPIRED)
    |
    v
Expired payments found?
    |-- no --> (wait next tick)
    |
    +-- yes --> Publish "payment.expired"
                  |
                  v
             OrderService.CancelOrder
                  |
                  v
         Update order status = CANCELLED
                  |
                  v
            Release reserved stock
```

## Checkout + Payment Success Flow

```
Client POST /checkout
    |
    v
OrderService.Checkout
    |
    v
Reserve stock per item
    |
    v
Create order + order items
    |
    v
Create payment (status=PENDING)
    |
    v
Respond with payment_url
    |
    v
Client completes payment
    |
    v
POST /payments/{payment_id}/success
    |
    v
ProcessPaymentSuccess
    |
    v
Update payment status = SUCCESS + paid_at
    |
    v
Confirm order stock
    |
    v
Update order status = PAID
```

## Payment Failed / Expired Flow

```
Payment FAILED
    |
    v
POST /payments/{payment_id}/fail
    |
    v
ProcessPaymentFailed
    |
    v
Update payment status = FAILED
    |
    v
Release reserved stock
    |
    v
Update order status = CANCELLED

PaymentExpiredEvent
    |
    v
OrderService.CancelOrder
    |
    v
Update order status = CANCELLED
    |
    v
Release reserved stock
```

Notes:
- Handlers are designed to be idempotent where possible; retries should not double‑process a payment.
- Expiration events are best‑effort in this prototype (in‑memory publisher). In production, use an outbox/retry mechanism.

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
- `api/internal/user/handler.go` (`CreateUser`)
- `api/internal/user/service.go` (`CreateUser`)
- `api/internal/user/repository.go` (`Create`)

### Register

- `POST /register`
- Content-Type: `application/json`
- Body:

```json
{
  "name": "NewUser",
  "email": "newuser@gmail.com",
  "password": "ChangeMe123!"
}
```

- Response: `200 OK`

### Login

- `POST /login`
- Content-Type: `application/json`
- Body:

```json
{
  "email": "newuser@gmail.com",
  "password": "ChangeMe123!"
}
```

- Response: `200 OK` with `{ "token": "<jwt>" }`

### Me

- `GET /me`
- Header: `Authorization: Bearer <token>`
- Response: `200 OK` with `{ "user_id": "<id>" }`

You can also use the scratch file `api/api_test.http` to try the endpoints from an IDE HTTP client.

### Products

- `POST /products`
- Content-Type: `application/json`
- Body:

```json
{
  "name": "T-Shirt",
  "description": "Cotton tee",
  "price": 19.99,
  "stock": 50
}
```

- Response: `201 Created`

### Reserve / Release / Confirm Stock

- `POST /products/{id}/reserve`
- `POST /products/{id}/release`
- `POST /products/{id}/confirm`
- Content-Type: `application/json`
- Body:

```json
{
  "qty": 2
}
```

### Checkout

- `POST /checkout`
- Header: `Authorization: Bearer <token>`
- Content-Type: `application/json`
- Body:

```json
{
  "payment_method": "MIDTRANS",
  "items": [
    { "product_id": "<product-id>", "quantity": 2 }
  ]
}
```

- Response: `200 OK` with `order_id`, `total_amount`, and `payment_url`.

### List Orders

- `GET /orders`
- Header: `Authorization: Bearer <token>`
- Response: `200 OK` with list of the user's orders.

### Get Order By ID

- `GET /orders/{id}`
- Header: `Authorization: Bearer <token>`
- Response: `200 OK` with order detail and items.

### Payments

- `POST /payments`
  - Header: `Authorization: Bearer <token>`
  - Body:

```json
{
  "order_id": "<order-id>",
  "amount": 125000,
  "payment_method": "MIDTRANS"
}
```

- `GET /payments/{payment_id}`
  - Header: `Authorization: Bearer <token>`

- `PATCH /payments/{payment_id}`
  - Header: `Authorization: Bearer <token>`
  - Body:

```json
{
  "status": "SUCCESS"
}
```

- `POST /payments/{payment_id}/success` (no auth)
- `POST /payments/{payment_id}/fail` (no auth)
- `POST /payments/callback` (no auth)
  - Body:

```json
{
  "payment_id": "<payment-id>",
  "status": "SUCCESS"
}
```

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
go run ./cmd/api
```

Server prints the port and listens on `http://localhost:8080` by default.

## Notes / Next Work
