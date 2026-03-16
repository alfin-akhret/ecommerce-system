ALTER TABLE payments
ADD COLUMN amount numeric(12,2) NOT NULL DEFAULT 0.0,
ADD COLUMN created_at timestamp NOT NULL DEFAULT now(),
ADD COLUMN updated_at timestamp NOT NULL DEFAULT now(),
ALTER COLUMN paid_at SET DEFAULT NULL;
