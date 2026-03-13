ALTER TABLE order_items
ADD CONSTRAINT unique_order_product
UNIQUE (order_id, product_id);
