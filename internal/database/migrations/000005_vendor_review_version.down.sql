BEGIN;
DROP TRIGGER vendors_review_version ON vendors;
DROP FUNCTION increment_vendor_review_version();
ALTER TABLE vendor_review_events DROP COLUMN review_version;
ALTER TABLE vendors DROP COLUMN review_version;
COMMIT;
