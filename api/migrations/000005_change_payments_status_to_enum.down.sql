ALTER TABLE payments
ALTER COLUMN status
TYPE varchar(50)
USING status::text;

ALTER TABLE payments
ALTER COLUMN status
DROP DEFAULT;

DROP TYPE payment_status;
