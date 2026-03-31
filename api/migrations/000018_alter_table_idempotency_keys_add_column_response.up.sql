ALTER TABLE idempotency_keys
ADD COLUMN response JSONB DEFAULT '{}'::jsonb;