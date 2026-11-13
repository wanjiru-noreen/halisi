BEGIN;

ALTER TABLE deliveries ADD COLUMN vendor_id BIGINT;
ALTER TABLE deliveries ADD COLUMN customer_id BIGINT;
ALTER TABLE deliveries ADD COLUMN address_id BIGINT;
ALTER TABLE deliveries ADD COLUMN pickup_address TEXT;
ALTER TABLE deliveries ADD COLUMN dropoff_address TEXT;
ALTER TABLE deliveries ADD COLUMN pickup_latitude NUMERIC(9,6);
ALTER TABLE deliveries ADD COLUMN pickup_longitude NUMERIC(9,6);
ALTER TABLE deliveries ADD COLUMN dropoff_latitude NUMERIC(9,6);
ALTER TABLE deliveries ADD COLUMN dropoff_longitude NUMERIC(9,6);

-- Preserve historical deliveries, including legacy orders without a saved address.
UPDATE deliveries d SET vendor_id=o.vendor_id,customer_id=o.customer_id,
    address_id=o.address_id,pickup_address=v.address,dropoff_address=o.delivery_address,
    pickup_latitude=pa.latitude,pickup_longitude=pa.longitude,
    dropoff_latitude=da.latitude,dropoff_longitude=da.longitude
FROM orders o JOIN vendors v ON v.id=o.vendor_id
LEFT JOIN addresses pa ON pa.id=v.address_id
LEFT JOIN addresses da ON da.id=o.address_id
WHERE o.id=d.order_id;
ALTER TABLE deliveries ALTER COLUMN vendor_id SET NOT NULL;
ALTER TABLE deliveries ALTER COLUMN customer_id SET NOT NULL;
ALTER TABLE deliveries ALTER COLUMN pickup_address SET NOT NULL;
ALTER TABLE deliveries ALTER COLUMN dropoff_address SET NOT NULL;
ALTER TABLE orders ADD CONSTRAINT orders_delivery_identity_key UNIQUE(id,vendor_id,customer_id);
ALTER TABLE deliveries ADD CONSTRAINT deliveries_order_identity_fk
    FOREIGN KEY(order_id,vendor_id,customer_id) REFERENCES orders(id,vendor_id,customer_id);
ALTER TABLE deliveries ADD CONSTRAINT deliveries_address_owner_fk
    FOREIGN KEY(address_id,customer_id) REFERENCES addresses(id,user_id);
ALTER TABLE deliveries ADD CONSTRAINT deliveries_locations_check CHECK (
    length(btrim(pickup_address))>0 AND length(btrim(dropoff_address))>0
    AND (pickup_latitude IS NULL)=(pickup_longitude IS NULL)
    AND (dropoff_latitude IS NULL)=(dropoff_longitude IS NULL)
    AND pickup_latitude BETWEEN -90 AND 90 AND pickup_longitude BETWEEN -180 AND 180
    AND dropoff_latitude BETWEEN -90 AND 90 AND dropoff_longitude BETWEEN -180 AND 180
);
CREATE INDEX deliveries_vendor_id_idx ON deliveries(vendor_id,id);
CREATE INDEX deliveries_customer_id_idx ON deliveries(customer_id,id);
CREATE INDEX deliveries_address_owner_idx ON deliveries(address_id,customer_id);

CREATE FUNCTION prepare_delivery() RETURNS TRIGGER LANGUAGE plpgsql AS $$
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
CREATE TRIGGER deliveries_prepare BEFORE INSERT ON deliveries FOR EACH ROW EXECUTE FUNCTION prepare_delivery();

CREATE FUNCTION guard_delivery_update() RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    IF (NEW.id,NEW.order_id,NEW.vendor_id,NEW.customer_id,NEW.address_id,NEW.pickup_address,NEW.dropoff_address,
        NEW.pickup_latitude,NEW.pickup_longitude,NEW.dropoff_latitude,NEW.dropoff_longitude,NEW.created_at)
       IS DISTINCT FROM
       (OLD.id,OLD.order_id,OLD.vendor_id,OLD.customer_id,OLD.address_id,OLD.pickup_address,OLD.dropoff_address,
        OLD.pickup_latitude,OLD.pickup_longitude,OLD.dropoff_latitude,OLD.dropoff_longitude,OLD.created_at) THEN
        RAISE EXCEPTION 'delivery identity and location snapshots are immutable' USING ERRCODE='23514';
    END IF;
    IF NEW.status<>OLD.status AND NOT (
        (OLD.status='pending' AND NEW.status IN ('assigned','cancelled')) OR
        (OLD.status='assigned' AND NEW.status IN ('picked_up','failed','cancelled')) OR
        (OLD.status='picked_up' AND NEW.status IN ('delivered','failed'))
    ) THEN
        RAISE EXCEPTION 'invalid delivery status transition' USING ERRCODE='23514';
    END IF;
    IF (NEW.status<>OLD.status OR NEW.rider_id IS DISTINCT FROM OLD.rider_id)
       AND NEW.status IN ('assigned','picked_up','delivered') AND EXISTS (
        SELECT 1 FROM orders WHERE id=NEW.order_id AND status NOT IN ('confirmed','preparing','dispatched')
    ) THEN
        RAISE EXCEPTION 'only confirmed active orders can advance to delivery' USING ERRCODE='23514';
    END IF;
    IF OLD.status='delivered' AND NEW.delivered_at IS DISTINCT FROM OLD.delivered_at THEN
        RAISE EXCEPTION 'delivery completion time is immutable' USING ERRCODE='23514';
    END IF;
    IF NEW.rider_id IS DISTINCT FROM OLD.rider_id AND NEW.status NOT IN ('pending','assigned') THEN
        RAISE EXCEPTION 'cannot reassign an active or terminal delivery' USING ERRCODE='23514';
    END IF;
    IF NEW.status='pending' AND NEW.rider_id IS NOT NULL THEN
        RAISE EXCEPTION 'pending delivery must be unassigned' USING ERRCODE='23514';
    END IF;
    IF (NEW.rider_id IS DISTINCT FROM OLD.rider_id OR NEW.status<>OLD.status)
        AND NEW.status IN ('assigned','picked_up','delivered','failed') AND NOT EXISTS (
            SELECT 1 FROM users u JOIN rider_profiles rp ON rp.user_id=u.id
            WHERE u.id=NEW.rider_id AND u.role='rider' AND rp.status='approved'
        ) THEN
        RAISE EXCEPTION 'delivery requires an approved rider' USING ERRCODE='23514';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER deliveries_guard BEFORE UPDATE ON deliveries FOR EACH ROW EXECUTE FUNCTION guard_delivery_update();

CREATE FUNCTION create_confirmed_order_delivery() RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP='UPDATE' AND NEW.status IS NOT DISTINCT FROM OLD.status THEN
        RETURN NEW;
    END IF;
    IF NEW.status='confirmed' THEN
        INSERT INTO deliveries(order_id) VALUES(NEW.id) ON CONFLICT(order_id) DO NOTHING;
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER orders_create_delivery AFTER INSERT OR UPDATE OF status ON orders
FOR EACH ROW EXECUTE FUNCTION create_confirmed_order_delivery();

-- Backfill eligible confirmed orders once; existing records keep their identity and state.
INSERT INTO deliveries(order_id)
SELECT o.id FROM orders o JOIN vendors v ON v.id=o.vendor_id
JOIN addresses a ON a.id=o.address_id AND a.user_id=o.customer_id
WHERE o.status IN ('confirmed','preparing') AND v.status='approved' AND v.is_active
AND NOT EXISTS(SELECT 1 FROM deliveries d WHERE d.order_id=o.id);

COMMIT;
