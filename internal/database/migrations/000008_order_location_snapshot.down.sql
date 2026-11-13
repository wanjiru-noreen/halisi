BEGIN;

-- Restore the version-7 function before dropping its replacement's columns.
CREATE OR REPLACE FUNCTION prepare_delivery() RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.status<>'pending' OR NEW.rider_id IS NOT NULL THEN
        RAISE EXCEPTION 'new deliveries must be pending and unassigned' USING ERRCODE='23514';
    END IF;
    SELECT o.vendor_id,o.customer_id,o.address_id,v.address,o.delivery_address,
           pa.latitude,pa.longitude,da.latitude,da.longitude
    INTO NEW.vendor_id,NEW.customer_id,NEW.address_id,NEW.pickup_address,NEW.dropoff_address,
         NEW.pickup_latitude,NEW.pickup_longitude,NEW.dropoff_latitude,NEW.dropoff_longitude
    FROM orders o JOIN vendors v ON v.id=o.vendor_id
    JOIN addresses da ON da.id=o.address_id AND da.user_id=o.customer_id
    LEFT JOIN addresses pa ON pa.id=v.address_id
    WHERE o.id=NEW.order_id AND o.status IN ('confirmed','preparing')
      AND v.status='approved' AND v.is_active
    FOR SHARE OF o,v,da;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'delivery requires a confirmed order, approved active vendor and customer address' USING ERRCODE='23514';
    END IF;
    RETURN NEW;
END;
$$;

ALTER TABLE orders DROP CONSTRAINT orders_delivery_coordinates_check;
ALTER TABLE orders DROP COLUMN delivery_latitude, DROP COLUMN delivery_longitude;

COMMIT;
