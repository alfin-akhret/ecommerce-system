CREATE TYPE order_status AS ENUM (
	'PENDING',
	'PENDING_PAYMENT',
	'PAID',
	'CANCELLED',
	'FAILED'
);

ALTER TABLE orders
ALTER COLUMN status
TYPE order_status
USING status::order_status;
