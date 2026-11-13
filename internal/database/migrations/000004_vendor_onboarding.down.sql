BEGIN;
DROP TRIGGER order_items_sellable_vendor ON order_items;
DROP TRIGGER orders_sellable_vendor ON orders;
DROP FUNCTION require_sellable_vendor();
DROP TABLE vendor_review_events;
ALTER TABLE vendor_documents DROP COLUMN original_name,DROP COLUMN content_type,DROP COLUMN size_bytes,DROP COLUMN sha256;
ALTER TABLE vendors DROP CONSTRAINT vendors_active_approved,DROP COLUMN contact_email,DROP COLUMN contact_phone,DROP COLUMN city,DROP COLUMN operating_hours,DROP COLUMN is_active,DROP COLUMN submitted_at,DROP COLUMN review_reason,DROP COLUMN reviewed_by,DROP COLUMN reviewed_at;
COMMIT;
