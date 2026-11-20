BEGIN;
ALTER TABLE vendors ADD COLUMN contact_email TEXT NOT NULL DEFAULT '';
ALTER TABLE vendors ADD COLUMN contact_phone TEXT NOT NULL DEFAULT '';
ALTER TABLE vendors ADD COLUMN city TEXT NOT NULL DEFAULT '';
ALTER TABLE vendors ADD COLUMN operating_hours JSONB NOT NULL DEFAULT '{}';
ALTER TABLE vendors ADD COLUMN is_active BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE vendors ADD COLUMN submitted_at TIMESTAMPTZ;
ALTER TABLE vendors ADD COLUMN review_reason TEXT NOT NULL DEFAULT '';
ALTER TABLE vendors ADD COLUMN reviewed_by BIGINT REFERENCES users(id);
ALTER TABLE vendors ADD COLUMN reviewed_at TIMESTAMPTZ;
ALTER TABLE vendors ADD CONSTRAINT vendors_active_approved CHECK (NOT is_active OR status='approved');
-- Legacy approvals did not validate the required profile/documents.
UPDATE vendors SET status='pending' WHERE status='approved';
CREATE INDEX vendors_search_active_idx ON vendors(city,name) WHERE status='approved' AND is_active;
CREATE INDEX vendors_reviewed_by_idx ON vendors(reviewed_by);
ALTER TABLE vendor_documents ADD COLUMN original_name TEXT NOT NULL DEFAULT '';
ALTER TABLE vendor_documents ADD COLUMN content_type TEXT NOT NULL DEFAULT '';
ALTER TABLE vendor_documents ADD COLUMN size_bytes BIGINT NOT NULL DEFAULT 0 CHECK (size_bytes BETWEEN 0 AND 5242880);
ALTER TABLE vendor_documents ADD COLUMN sha256 TEXT NOT NULL DEFAULT '';
CREATE TABLE vendor_review_events (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    vendor_id BIGINT NOT NULL REFERENCES vendors(id),
    administrator_id BIGINT NOT NULL REFERENCES users(id),
    status TEXT NOT NULL CHECK (status IN ('pending','approved','rejected','suspended')),
    reason TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX vendor_review_events_vendor_idx ON vendor_review_events(vendor_id,created_at);
CREATE INDEX vendor_review_events_admin_idx ON vendor_review_events(administrator_id);
CREATE FUNCTION require_sellable_vendor() RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    PERFORM id FROM vendors WHERE id=NEW.vendor_id AND status='approved' AND is_active FOR SHARE;
    IF NOT FOUND THEN RAISE EXCEPTION 'vendor is not approved and active' USING ERRCODE='23514'; END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER orders_sellable_vendor BEFORE INSERT OR UPDATE OF vendor_id ON orders FOR EACH ROW EXECUTE FUNCTION require_sellable_vendor();
CREATE TRIGGER order_items_sellable_vendor BEFORE INSERT OR UPDATE OF vendor_id ON order_items FOR EACH ROW EXECUTE FUNCTION require_sellable_vendor();
COMMIT;
