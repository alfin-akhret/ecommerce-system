ALTER TABLE orders
ALTER COLUMN status
TYPE varchar(50)
USING status::text;

DROP TYPE order_status;
