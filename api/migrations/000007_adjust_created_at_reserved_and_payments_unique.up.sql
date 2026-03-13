UPDATE users
SET created_at = now()
WHERE created_at IS NULL;

ALTER TABLE users
ALTER COLUMN created_at SET DEFAULT now(),
ALTER COLUMN created_at SET NOT NULL;

UPDATE products
SET created_at = now()
WHERE created_at IS NULL;

ALTER TABLE products
ALTER COLUMN created_at SET DEFAULT now(),
ALTER COLUMN created_at SET NOT NULL;

UPDATE orders
SET created_at = now()
WHERE created_at IS NULL;

ALTER TABLE orders
ALTER COLUMN created_at SET DEFAULT now(),
ALTER COLUMN created_at SET NOT NULL;

UPDATE order_items
SET created_at = now()
WHERE created_at IS NULL;

ALTER TABLE order_items
ALTER COLUMN created_at SET DEFAULT now(),
ALTER COLUMN created_at SET NOT NULL;

UPDATE product_inventory
SET reserved = 0
WHERE reserved IS NULL;

ALTER TABLE product_inventory
ALTER COLUMN reserved SET DEFAULT 0,
ALTER COLUMN reserved SET NOT NULL;

ALTER TABLE payments
ADD CONSTRAINT payments_order_id_unique UNIQUE (order_id);
