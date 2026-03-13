ALTER TABLE order_items
ADD CONSTRAINT check_quantity_positive
CHECK (quantity > 0);