ALTER TABLE payments
DROP COLUMN amount,
DROP COLUMN created_at,
DROP COLUMN updated_at,
ALTER COLUMN paid_at DROP DEFAULT;
