# ecommerce-system

Compact Go backend for an e-commerce system. It models the core shopping flow: users authenticate, browse products, manage a cart, create orders, reserve stock, and complete payments.

## Overview

The main service is a REST API built with Go and `chi`. It uses Postgres for durable data, Redis for cart storage, and background workers for payment expiration and idempotency cleanup.

High-level request flow:

```text
HTTP -> handler -> service -> repository -> database
```

Core domains:

- `auth`: JWT login, authenticated user context, admin middleware
- `user`: registration and user lookup
- `product`: product catalog and stock reservation/release/confirmation
- `cart`: Redis-backed shopping cart
- `order`: order creation, checkout, cancellation, and idempotency handling
- `payment`: payment creation, callbacks, success/failure processing, expiration worker

## Repo Layout

```text
api/                  Main Go API service
api/cmd/api/          HTTP server entrypoint and route setup
api/internal/         Application domains and wiring
api/migrations/       Postgres schema migrations
api/pkg/helper/       Shared HTTP, logging, metrics, and tracing helpers
fake-payment-api/     Minimal fake payment provider for local callbacks
infrastructure/       Docker Compose and observability config
docs/                 Development notes and flow documentation
```

## Infrastructure

Local Docker config includes:

- Postgres
- Redis
- Prometheus, Grafana, and Alertmanager
- Jaeger tracing
- Elasticsearch, Kibana, and Filebeat
- API container

The API exposes `/health` for health checks and `/metrics` for Prometheus scraping.

## Main Flows

- Checkout reserves product stock, creates an order, and creates a pending payment.
- Payment success confirms stock and marks the order as paid.
- Payment failure or expiration cancels the order and releases reserved stock.
- Payment callbacks are signed with HMAC in the fake payment provider flow.

## Useful Paths

- API entrypoint: `api/cmd/api/main.go`
- Dependency wiring: `api/internal/app/app.go`
- Docker Compose: `infrastructure/docker/docker-compose.yml`
- Development notes: `docs/`
