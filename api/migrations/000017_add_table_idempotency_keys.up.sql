CREATE TABLE idempotency_keys (
    key uuid primary key,
    user_id uuid not null,
    status varchar(50) not null,
    expired_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
