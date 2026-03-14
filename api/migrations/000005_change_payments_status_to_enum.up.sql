CREATE TYPE payment_status AS ENUM (
	'PENDING',
	'SUCCESS',
	'FAILED',
	'CANCELLED'
);

ALTER TABLE payments
ALTER COLUMN status
TYPE payment_status
USING status::payment_status;

ALTER TABLE payments
ALTER COLUMN status
SET DEFAULT 'PENDING';
