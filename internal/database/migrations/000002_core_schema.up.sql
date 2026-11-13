BEGIN;

ALTER TABLE users ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('customer', 'vendor', 'rider', 'admin'));
ALTER TABLE users ADD CONSTRAINT users_values_check CHECK (
    length(btrim(name)) > 0 AND length(btrim(email)) > 0 AND email = btrim(email)
    AND length(btrim(password_hash)) > 0 AND (phone IS NULL OR length(btrim(phone)) > 0)
);
CREATE UNIQUE INDEX users_email_lower_idx ON users(lower(email));

CREATE TABLE sessions (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE CHECK (token_hash ~ '^[0-9a-f]{64}$'),
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (expires_at > created_at),
    CHECK (revoked_at IS NULL OR revoked_at >= created_at)
);
CREATE INDEX sessions_user_id_idx ON sessions(user_id);
CREATE INDEX sessions_expires_at_idx ON sessions(expires_at);

ALTER TABLE vendors ADD COLUMN status TEXT NOT NULL DEFAULT 'pending';
ALTER TABLE vendors ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE vendors ADD CONSTRAINT vendors_status_check CHECK (status IN ('pending', 'approved', 'rejected', 'suspended'));
ALTER TABLE vendors ADD CONSTRAINT vendors_values_check CHECK (length(btrim(name)) > 0 AND length(btrim(address)) > 0);
CREATE INDEX vendors_status_idx ON vendors(status);

CREATE TABLE vendor_documents (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    vendor_id BIGINT NOT NULL REFERENCES vendors(id) ON DELETE CASCADE,
    document_type TEXT NOT NULL CHECK (length(btrim(document_type)) > 0),
    storage_key TEXT NOT NULL CHECK (length(btrim(storage_key)) > 0),
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (vendor_id, document_type)
);

ALTER TABLE products ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE products ADD CONSTRAINT products_vendor_id_id_key UNIQUE (vendor_id, id);
ALTER TABLE products ADD CONSTRAINT products_values_check CHECK (length(btrim(name)) > 0 AND price <> 'NaN'::numeric);

CREATE TABLE vendor_inventory (
    product_id BIGINT PRIMARY KEY,
    vendor_id BIGINT NOT NULL,
    quantity INTEGER NOT NULL DEFAULT 0 CHECK (quantity >= 0),
    reserved_quantity INTEGER NOT NULL DEFAULT 0 CHECK (reserved_quantity >= 0 AND reserved_quantity <= quantity),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (vendor_id, product_id) REFERENCES products(vendor_id, id) ON DELETE CASCADE
);
CREATE INDEX vendor_inventory_vendor_id_idx ON vendor_inventory(vendor_id);
INSERT INTO vendor_inventory (product_id, vendor_id, quantity, created_at)
SELECT id, vendor_id, stock, created_at FROM products;
ALTER TABLE products DROP COLUMN stock;

CREATE TABLE addresses (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    label TEXT NOT NULL DEFAULT 'Home' CHECK (length(btrim(label)) > 0),
    address_line TEXT NOT NULL CHECK (length(btrim(address_line)) > 0),
    city TEXT NOT NULL CHECK (length(btrim(city)) > 0),
    latitude NUMERIC(9,6) CHECK (latitude BETWEEN -90 AND 90),
    longitude NUMERIC(9,6) CHECK (longitude BETWEEN -180 AND 180),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK ((latitude IS NULL) = (longitude IS NULL)),
    UNIQUE (id, user_id)
);
CREATE INDEX addresses_user_id_idx ON addresses(user_id);

ALTER TABLE vendors ADD COLUMN address_id BIGINT;
ALTER TABLE vendors ADD CONSTRAINT vendors_address_owner_fk FOREIGN KEY (address_id, user_id) REFERENCES addresses(id, user_id);
CREATE INDEX vendors_address_owner_idx ON vendors(address_id, user_id);

ALTER TABLE orders ADD COLUMN address_id BIGINT;
ALTER TABLE orders ADD COLUMN currency TEXT NOT NULL DEFAULT 'KES';
ALTER TABLE orders ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE orders ADD CONSTRAINT orders_status_check CHECK (status IN ('pending', 'confirmed', 'preparing', 'dispatched', 'delivered', 'cancelled'));
ALTER TABLE orders ADD CONSTRAINT orders_values_check CHECK (length(btrim(delivery_address)) > 0 AND total <> 'NaN'::numeric AND currency ~ '^[A-Z]{3}$');
ALTER TABLE orders ADD CONSTRAINT orders_id_vendor_key UNIQUE (id, vendor_id);
ALTER TABLE orders ADD CONSTRAINT orders_review_key UNIQUE (id, customer_id, vendor_id);
ALTER TABLE orders ADD CONSTRAINT orders_currency_key UNIQUE (id, currency);
ALTER TABLE orders ADD CONSTRAINT orders_address_owner_fk FOREIGN KEY (address_id, customer_id) REFERENCES addresses(id, user_id);
CREATE INDEX orders_address_owner_idx ON orders(address_id, customer_id);
CREATE INDEX orders_status_created_at_idx ON orders(status, created_at);

ALTER TABLE order_items ADD COLUMN vendor_id BIGINT;
UPDATE order_items SET vendor_id = orders.vendor_id FROM orders WHERE orders.id = order_items.order_id;
ALTER TABLE order_items ALTER COLUMN vendor_id SET NOT NULL;
ALTER TABLE order_items ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE order_items ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE order_items ADD CONSTRAINT order_items_price_check CHECK (unit_price <> 'NaN'::numeric);
ALTER TABLE order_items ADD CONSTRAINT order_items_order_vendor_fk FOREIGN KEY (order_id, vendor_id) REFERENCES orders(id, vendor_id);
ALTER TABLE order_items ADD CONSTRAINT order_items_product_vendor_fk FOREIGN KEY (vendor_id, product_id) REFERENCES products(vendor_id, id);
CREATE INDEX order_items_vendor_product_idx ON order_items(vendor_id, product_id);

ALTER TABLE payments ADD COLUMN currency TEXT NOT NULL DEFAULT 'KES';
ALTER TABLE payments ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE payments ADD CONSTRAINT payments_status_check CHECK (status IN ('pending', 'processing', 'completed', 'failed', 'refunded', 'cancelled'));
ALTER TABLE payments ADD CONSTRAINT payments_finite_amount_check CHECK (amount <> 'NaN'::numeric);
ALTER TABLE payments ADD CONSTRAINT payments_order_currency_fk FOREIGN KEY (order_id, currency) REFERENCES orders(id, currency);
CREATE INDEX payments_status_created_at_idx ON payments(status, created_at);

ALTER TABLE deliveries ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE deliveries ADD CONSTRAINT deliveries_status_check CHECK (status IN ('pending', 'assigned', 'picked_up', 'delivered', 'failed', 'cancelled'));
ALTER TABLE deliveries ADD CONSTRAINT deliveries_completed_check CHECK ((status = 'delivered') = (delivered_at IS NOT NULL) AND (delivered_at IS NULL OR delivered_at >= created_at));
ALTER TABLE deliveries ADD CONSTRAINT deliveries_assignment_check CHECK (status NOT IN ('assigned', 'picked_up', 'delivered') OR rider_id IS NOT NULL);
CREATE INDEX deliveries_status_idx ON deliveries(status);

CREATE TABLE reviews (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_id BIGINT NOT NULL UNIQUE,
    customer_id BIGINT NOT NULL,
    vendor_id BIGINT NOT NULL,
    rating SMALLINT NOT NULL CHECK (rating BETWEEN 1 AND 5),
    comment TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (order_id, customer_id, vendor_id) REFERENCES orders(id, customer_id, vendor_id)
);
CREATE INDEX reviews_customer_id_idx ON reviews(customer_id);
CREATE INDEX reviews_vendor_created_at_idx ON reviews(vendor_id, created_at);

CREATE FUNCTION set_updated_at() RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    NEW.updated_at = clock_timestamp();
    RETURN NEW;
END;
$$;

CREATE TRIGGER users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER sessions_updated_at BEFORE UPDATE ON sessions FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER vendors_updated_at BEFORE UPDATE ON vendors FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER vendor_documents_updated_at BEFORE UPDATE ON vendor_documents FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER products_updated_at BEFORE UPDATE ON products FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER vendor_inventory_updated_at BEFORE UPDATE ON vendor_inventory FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER addresses_updated_at BEFORE UPDATE ON addresses FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER orders_updated_at BEFORE UPDATE ON orders FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER order_items_updated_at BEFORE UPDATE ON order_items FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER payments_updated_at BEFORE UPDATE ON payments FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER deliveries_updated_at BEFORE UPDATE ON deliveries FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER reviews_updated_at BEFORE UPDATE ON reviews FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMIT;
