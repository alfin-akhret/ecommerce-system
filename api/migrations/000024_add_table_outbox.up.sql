CREATE TABLE outbox (
    id UUID PRIMARY KEY,
    event_type VARCHAR(255),
    payload JSONB,
    created_at TIMESTAMPTZ,
    published_at TIMESTAMPTZ NULL
);
