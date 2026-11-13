BEGIN;
DROP TRIGGER orders_create_delivery ON orders;
DROP FUNCTION create_confirmed_order_delivery();
DROP TRIGGER deliveries_guard ON deliveries;
DROP FUNCTION guard_delivery_update();
DROP TRIGGER deliveries_prepare ON deliveries;
DROP FUNCTION prepare_delivery();
ALTER TABLE deliveries DROP CONSTRAINT deliveries_order_identity_fk;
ALTER TABLE orders DROP CONSTRAINT orders_delivery_identity_key;
ALTER TABLE deliveries DROP CONSTRAINT deliveries_address_owner_fk;
ALTER TABLE deliveries DROP CONSTRAINT deliveries_locations_check;
DROP INDEX deliveries_vendor_id_idx;
DROP INDEX deliveries_customer_id_idx;
DROP INDEX deliveries_address_owner_idx;
ALTER TABLE deliveries DROP COLUMN vendor_id, DROP COLUMN customer_id, DROP COLUMN address_id,
    DROP COLUMN pickup_address, DROP COLUMN dropoff_address,
    DROP COLUMN pickup_latitude, DROP COLUMN pickup_longitude,
    DROP COLUMN dropoff_latitude, DROP COLUMN dropoff_longitude;
COMMIT;
