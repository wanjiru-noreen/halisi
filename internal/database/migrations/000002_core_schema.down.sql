BEGIN;

DROP TRIGGER reviews_updated_at ON reviews;
DROP TRIGGER deliveries_updated_at ON deliveries;
DROP TRIGGER payments_updated_at ON payments;
DROP TRIGGER order_items_updated_at ON order_items;
DROP TRIGGER orders_updated_at ON orders;
DROP TRIGGER addresses_updated_at ON addresses;
DROP TRIGGER vendor_inventory_updated_at ON vendor_inventory;
DROP TRIGGER products_updated_at ON products;
DROP TRIGGER vendor_documents_updated_at ON vendor_documents;
DROP TRIGGER vendors_updated_at ON vendors;
DROP TRIGGER sessions_updated_at ON sessions;
DROP TRIGGER users_updated_at ON users;
DROP FUNCTION set_updated_at();

DROP TABLE reviews;
ALTER TABLE deliveries DROP CONSTRAINT deliveries_assignment_check, DROP CONSTRAINT deliveries_completed_check, DROP CONSTRAINT deliveries_status_check, DROP COLUMN updated_at;
DROP INDEX deliveries_status_idx;
ALTER TABLE payments DROP CONSTRAINT payments_order_currency_fk, DROP CONSTRAINT payments_finite_amount_check, DROP CONSTRAINT payments_status_check, DROP COLUMN currency, DROP COLUMN updated_at;
DROP INDEX payments_status_created_at_idx;
ALTER TABLE order_items DROP CONSTRAINT order_items_product_vendor_fk, DROP CONSTRAINT order_items_order_vendor_fk, DROP CONSTRAINT order_items_price_check, DROP COLUMN vendor_id, DROP COLUMN created_at, DROP COLUMN updated_at;
ALTER TABLE orders DROP CONSTRAINT orders_address_owner_fk, DROP CONSTRAINT orders_currency_key, DROP CONSTRAINT orders_review_key, DROP CONSTRAINT orders_id_vendor_key, DROP CONSTRAINT orders_values_check, DROP CONSTRAINT orders_status_check, DROP COLUMN address_id, DROP COLUMN currency, DROP COLUMN updated_at;
DROP INDEX orders_status_created_at_idx;
ALTER TABLE vendors DROP CONSTRAINT vendors_address_owner_fk, DROP COLUMN address_id;
DROP TABLE addresses;

ALTER TABLE products ADD COLUMN stock INTEGER NOT NULL DEFAULT 0 CHECK (stock >= 0);
UPDATE products SET stock = vendor_inventory.quantity FROM vendor_inventory WHERE vendor_inventory.product_id = products.id;
DROP TABLE vendor_inventory;
ALTER TABLE products DROP CONSTRAINT products_values_check, DROP CONSTRAINT products_vendor_id_id_key, DROP COLUMN updated_at;
DROP TABLE vendor_documents;
ALTER TABLE vendors DROP CONSTRAINT vendors_values_check, DROP CONSTRAINT vendors_status_check, DROP COLUMN status, DROP COLUMN updated_at;
DROP TABLE sessions;
DROP INDEX users_email_lower_idx;
ALTER TABLE users DROP CONSTRAINT users_values_check, DROP CONSTRAINT users_role_check, DROP COLUMN updated_at;

COMMIT;
